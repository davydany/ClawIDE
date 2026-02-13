package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/version"
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
		"auto_update_check":  true,
	}

	for k, v := range updates {
		if allowedFields[k] {
			// Map old claude_command key to agent_command
			if k == "claude_command" {
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
