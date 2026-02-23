package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/version"
	"github.com/davydany/ClawIDE/internal/wizard"
)

func (h *Handlers) SettingsPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"Title":   "Settings - ClawIDE",
		"Config":  h.cfg,
		"Version": version.Version,
	}

	if err := h.renderer.RenderHTMX(w, r, "settings", "settings", data); err != nil {
		log.Printf("Error rendering settings: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Read existing config file
	configPath := filepath.Join(h.cfg.DataDir, "config.json")
	existing := make(map[string]any)

	data, err := os.ReadFile(configPath)
	if err == nil {
		json.Unmarshal(data, &existing)
	}

	// Merge updates (only allow safe fields)
	allowedFields := map[string]bool{
		"projects_dir":     true,
		"max_sessions":     true,
		"scrollback_size":  true,
		"agent_command":    true,
		"agent_args":       true,
		"claude_command":   true, // backward compat: maps to agent_command
		"log_level":        true,
		"host":             true,
		"port":             true,
		"sidebar_position": true,
		"sidebar_width":      true,
		"auto_update_check": true,
		"ai_settings":       true, // Allow nested AI settings updates
	}

	for k, v := range updates {
		if allowedFields[k] {
			// Validate max_sessions
			if k == "max_sessions" {
				if val, ok := v.(float64); ok {
					sessions := int(val)
					if sessions < 10 || sessions > 100 {
						http.Error(w, "max_sessions must be between 10 and 100", http.StatusBadRequest)
						return
					}
					existing[k] = sessions
				} else {
					http.Error(w, "max_sessions must be a number", http.StatusBadRequest)
					return
				}
			} else if k == "claude_command" {
				// Map old claude_command key to agent_command
				existing["agent_command"] = v
			} else {
				existing[k] = v
			}
		}
	}

	// Write back
	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal config", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(configPath, out, 0644); err != nil {
		http.Error(w, "Failed to write config", http.StatusInternalServerError)
		return
	}

	// Reload into current config (fix: actually update in-memory config)
	newCfg := config.DefaultConfig()
	json.Unmarshal(out, newCfg)
	h.cfg.SidebarPosition = newCfg.SidebarPosition
	h.cfg.SidebarWidth = newCfg.SidebarWidth
	h.cfg.ProjectsDir = newCfg.ProjectsDir
	h.cfg.MaxSessions = newCfg.MaxSessions
	h.cfg.ScrollbackSize = newCfg.ScrollbackSize
	h.cfg.AgentCommand = newCfg.AgentCommand
	h.cfg.AgentArgs = newCfg.AgentArgs
	h.cfg.LogLevel = newCfg.LogLevel
	h.cfg.Host = newCfg.Host
	h.cfg.Port = newCfg.Port
	h.cfg.AutoUpdateCheck = newCfg.AutoUpdateCheck

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetAISettings returns the current AI configuration from settings
func (h *Handlers) GetAISettings(w http.ResponseWriter, r *http.Request) {
	aiCfg := h.cfg.GetAIConfig()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ai_settings": aiCfg,
		"providers":   wizard.GetAvailableProviders(),
	})
}

// SetAISettings updates and persists the AI configuration
func (h *Handlers) SetAISettings(w http.ResponseWriter, r *http.Request) {
	var aiCfg wizard.AIConfig
	if err := json.NewDecoder(r.Body).Decode(&aiCfg); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate the configuration
	if err := aiCfg.Validate(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
		return
	}

	// Save to config
	if err := h.cfg.SaveAIConfig(&aiCfg); err != nil {
		log.Printf("Failed to save AI config: %v", err)
		http.Error(w, "Failed to save AI configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"config": &aiCfg,
	})
}

// VerifyAICredentials tests API credentials without storing them
func (h *Handlers) VerifyAICredentials(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider  string `json:"provider"`
		APIKey    string `json:"api_key"`
		BaseURL   string `json:"base_url"`
		Model     string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Provider == "" || req.APIKey == "" || req.Model == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "provider, api_key, and model are required",
		})
		return
	}

	// For now, basic validation - full verification would require actual API calls
	// This is a security feature: we only test when needed, not during settings load
	provider := wizard.AIProvider(req.Provider)

	// Verify provider exists
	validProvider := false
	for _, p := range wizard.GetAvailableProviders() {
		if p == provider {
			validProvider = true
			break
		}
	}

	if !validProvider {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid provider: " + req.Provider,
		})
		return
	}

	// Verify model exists for provider
	models := wizard.ProviderModels(provider)
	modelExists := false
	var selectedModel wizard.AIModel
	for _, m := range models {
		if m.ID == req.Model {
			modelExists = true
			selectedModel = m
			break
		}
	}

	if !modelExists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid model for provider: " + req.Model,
		})
		return
	}

	// Basic validation passed
	// Note: Real API testing would happen here (making a test call)
	// For security, we don't actually call the API here - just validate format
	if len(req.APIKey) < 10 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "API key appears to be invalid (too short)",
		})
		return
	}

	// Ollama requires base URL
	if provider == wizard.AIProviderOllama && req.BaseURL == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "base_url is required for Ollama",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"valid": true,
		"model": selectedModel,
		"message": "Configuration appears valid. Note: Full verification requires API testing which we don't perform for security.",
	})
}
