package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func (h *Handlers) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	h.cfg.OnboardingCompleted = true

	if err := h.saveConfigFlag("onboarding_completed", true); err != nil {
		log.Printf("Error saving onboarding state: %v", err)
		http.Error(w, "Failed to save onboarding state", http.StatusInternalServerError)
		return
	}

	redirectURL := "/"
	if r.FormValue("start_tour") == "true" {
		redirectURL = "/?tour=dashboard"
	}

	// Handle HTMX requests
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (h *Handlers) CompleteWorkspaceTour(w http.ResponseWriter, r *http.Request) {
	h.cfg.WorkspaceTourCompleted = true

	if err := h.saveConfigFlag("workspace_tour_completed", true); err != nil {
		log.Printf("Error saving workspace tour state: %v", err)
		http.Error(w, "Failed to save workspace tour state", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) ResetOnboarding(w http.ResponseWriter, r *http.Request) {
	h.cfg.OnboardingCompleted = false
	h.cfg.WorkspaceTourCompleted = false

	if err := h.saveConfigFlags(map[string]any{
		"onboarding_completed":     false,
		"workspace_tour_completed": false,
	}); err != nil {
		log.Printf("Error saving onboarding state: %v", err)
		http.Error(w, "Failed to save onboarding state", http.StatusInternalServerError)
		return
	}

	// Handle HTMX requests
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) saveConfigFlag(key string, value any) error {
	return h.saveConfigFlags(map[string]any{key: value})
}

func (h *Handlers) saveConfigFlags(flags map[string]any) error {
	configPath := filepath.Join(h.cfg.DataDir, "config.json")
	existing := make(map[string]any)

	data, err := os.ReadFile(configPath)
	if err == nil {
		json.Unmarshal(data, &existing)
	}

	for k, v := range flags {
		existing[k] = v
	}

	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, out, 0644)
}
