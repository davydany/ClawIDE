package handler

import (
	"encoding/json"
	"net/http"

	"github.com/davydany/ClawIDE/internal/updater"
)

// CheckForUpdate triggers a fresh GitHub API check and returns the result.
func (h *Handlers) CheckForUpdate(w http.ResponseWriter, r *http.Request) {
	if h.updater == nil {
		http.Error(w, "updater not available", http.StatusServiceUnavailable)
		return
	}

	state := h.updater.Check()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// UpdateStatus returns the cached update state without making an API call.
func (h *Handlers) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	if h.updater == nil {
		http.Error(w, "updater not available", http.StatusServiceUnavailable)
		return
	}

	state := h.updater.State()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// ApplyUpdate starts the update install process in a background goroutine.
func (h *Handlers) ApplyUpdate(w http.ResponseWriter, r *http.Request) {
	if h.updater == nil {
		http.Error(w, "updater not available", http.StatusServiceUnavailable)
		return
	}

	if updater.IsInstalling() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "An install is already in progress.",
		})
		return
	}

	if updater.IsDocker() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Running in Docker. Use docker pull to update instead.",
		})
		return
	}

	state := h.updater.State()
	if !state.UpdateAvailable {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "No update available.",
		})
		return
	}

	// Start install in background
	go func() {
		if err := h.updater.Install(); err != nil {
			// Log the error; the process may have already exited on success
			println("[updater] install failed:", err.Error())
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "installing",
		"message": "Update to " + state.LatestVersion + " started. ClawIDE will restart automatically.",
	})
}
