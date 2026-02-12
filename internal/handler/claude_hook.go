package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type detectResponse struct {
	Detected       bool   `json:"detected"`
	Path           string `json:"path,omitempty"`
	HookConfigured bool   `json:"hook_configured"`
}

func (h *Handlers) DetectClaudeCLI(w http.ResponseWriter, r *http.Request) {
	resp := detectResponse{
		HookConfigured: h.cfg.ClaudeHookConfigured,
	}

	path, err := exec.LookPath("claude")
	if err == nil {
		resp.Detected = true
		resp.Path = path
	}

	// Also check if hook exists in settings
	if !resp.HookConfigured {
		resp.HookConfigured = checkClaudeSettingsForHook()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) SetupClaudeHook(w http.ResponseWriter, r *http.Request) {
	hooksDir := h.cfg.HooksDir()
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("failed to create hooks dir: %v", err), http.StatusInternalServerError)
		return
	}

	scriptPath := filepath.Join(hooksDir, "claude-stop-hook.sh")
	script := generateHookScript(h.cfg.Port)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		http.Error(w, fmt.Sprintf("failed to write hook script: %v", err), http.StatusInternalServerError)
		return
	}

	// Update Claude's settings.json
	if err := addClaudeHookSetting(scriptPath); err != nil {
		log.Printf("Warning: failed to update Claude settings: %v", err)
		http.Error(w, fmt.Sprintf("hook script created but failed to update Claude settings: %v", err), http.StatusInternalServerError)
		return
	}

	h.cfg.ClaudeHookConfigured = true

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success":     true,
		"script_path": scriptPath,
	})
}

func (h *Handlers) RemoveClaudeHook(w http.ResponseWriter, r *http.Request) {
	hooksDir := h.cfg.HooksDir()
	scriptPath := filepath.Join(hooksDir, "claude-stop-hook.sh")

	// Remove hook script
	os.Remove(scriptPath)

	// Remove from Claude settings
	if err := removeClaudeHookSetting(); err != nil {
		log.Printf("Warning: failed to clean Claude settings: %v", err)
	}

	h.cfg.ClaudeHookConfigured = false

	w.WriteHeader(http.StatusNoContent)
}

func generateHookScript(port int) string {
	return fmt.Sprintf(`#!/bin/bash
# ClawIDE Claude Code stop hook
# Sends a notification to ClawIDE when Claude Code finishes a task.
# This script is called by Claude Code's hook system on the "Stop" event.

# Read the hook event JSON from stdin
HOOK_JSON=$(cat)

# Use CLAWIDE env vars if available (set by ClawIDE terminal sessions)
PROJECT_ID="${CLAWIDE_PROJECT_ID:-}"
SESSION_ID="${CLAWIDE_SESSION_ID:-}"
FEATURE_ID="${CLAWIDE_FEATURE_ID:-}"
PANE_ID="${CLAWIDE_PANE_ID:-}"
API_URL="${CLAWIDE_API_URL:-http://localhost:%d}"

# Extract stop reason from hook JSON (if jq is available)
STOP_REASON=""
if command -v jq &>/dev/null; then
    STOP_REASON=$(echo "$HOOK_JSON" | jq -r '.stop_hook_reason // "completed"' 2>/dev/null)
fi
STOP_REASON="${STOP_REASON:-completed}"

# Build notification title
TITLE="Claude Code ${STOP_REASON}"

# Send notification to ClawIDE (fire-and-forget)
curl -s -X POST "${API_URL}/api/notifications" \
    -H "Content-Type: application/json" \
    -d "{
        \"title\": \"${TITLE}\",
        \"body\": \"Claude Code has finished in $(basename \"$(pwd)\")\",
        \"source\": \"claude\",
        \"level\": \"success\",
        \"project_id\": \"${PROJECT_ID}\",
        \"session_id\": \"${SESSION_ID}\",
        \"feature_id\": \"${FEATURE_ID}\",
        \"pane_id\": \"${PANE_ID}\",
        \"cwd\": \"$(pwd)\"
    }" &>/dev/null &

exit 0
`, port)
}

func checkClaudeSettingsForHook() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return false
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return false
	}
	hooks, ok := settings["hooks"]
	if !ok {
		return false
	}
	hooksMap, ok := hooks.(map[string]any)
	if !ok {
		return false
	}
	stopHooks, ok := hooksMap["Stop"]
	if !ok {
		return false
	}
	stopList, ok := stopHooks.([]any)
	if !ok {
		return false
	}
	for _, hook := range stopList {
		// New format: {"hooks": [{"command": "...", "type": "command"}]}
		if hookObj, ok := hook.(map[string]any); ok {
			if innerHooks, ok := hookObj["hooks"].([]any); ok {
				for _, ih := range innerHooks {
					if ihMap, ok := ih.(map[string]any); ok {
						if cmd, ok := ihMap["command"].(string); ok {
							if filepath.Base(cmd) == "claude-stop-hook.sh" {
								return true
							}
						}
					}
				}
			}
		}
		// Legacy format: bare string (for detecting old configs)
		if hookStr, ok := hook.(string); ok {
			if filepath.Base(hookStr) == "claude-stop-hook.sh" {
				return true
			}
		}
	}
	return false
}

func addClaudeHookSetting(scriptPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("creating .claude dir: %w", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	var settings map[string]any

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading settings: %w", err)
		}
		settings = make(map[string]any)
	} else {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parsing settings: %w", err)
		}
	}

	// Ensure hooks.Stop array exists and add our script
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		hooks = make(map[string]any)
	}

	stopHooks, ok := hooks["Stop"].([]any)
	if !ok {
		stopHooks = []any{}
	}

	// Check if already present (in new object format)
	for _, h := range stopHooks {
		if hookObj, ok := h.(map[string]any); ok {
			if innerHooks, ok := hookObj["hooks"].([]any); ok {
				for _, ih := range innerHooks {
					if ihMap, ok := ih.(map[string]any); ok {
						if cmd, ok := ihMap["command"].(string); ok && cmd == scriptPath {
							return nil // Already configured
						}
					}
				}
			}
		}
		// Also check legacy string format to avoid duplicates
		if hStr, ok := h.(string); ok && hStr == scriptPath {
			return nil
		}
	}

	// Use the new Claude Code hook format with matchers
	hookEntry := map[string]any{
		"hooks": []any{
			map[string]any{
				"command": scriptPath,
				"type":    "command",
			},
		},
	}

	stopHooks = append(stopHooks, hookEntry)
	hooks["Stop"] = stopHooks
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	return os.WriteFile(settingsPath, out, 0644)
}

func removeClaudeHookSetting() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil // No settings file, nothing to clean
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return nil
	}

	stopHooks, ok := hooks["Stop"].([]any)
	if !ok {
		return nil
	}

	// Filter out our hook in both new object format and legacy string format
	var filtered []any
	for _, h := range stopHooks {
		// New format: {"hooks": [{"command": "...", "type": "command"}]}
		if hookObj, ok := h.(map[string]any); ok {
			if innerHooks, ok := hookObj["hooks"].([]any); ok {
				isOurs := false
				for _, ih := range innerHooks {
					if ihMap, ok := ih.(map[string]any); ok {
						if cmd, ok := ihMap["command"].(string); ok {
							if filepath.Base(cmd) == "claude-stop-hook.sh" {
								isOurs = true
								break
							}
						}
					}
				}
				if isOurs {
					continue
				}
			}
		}
		// Legacy format: bare string
		if hookStr, ok := h.(string); ok && filepath.Base(hookStr) == "claude-stop-hook.sh" {
			continue
		}
		filtered = append(filtered, h)
	}

	if len(filtered) == 0 {
		delete(hooks, "Stop")
	} else {
		hooks["Stop"] = filtered
	}
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, out, 0644)
}
