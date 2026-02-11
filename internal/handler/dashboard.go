package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	// Show welcome screen if onboarding not completed
	if !h.cfg.OnboardingCompleted {
		data := map[string]any{
			"Title": "Welcome to ClawIDE",
		}
		if err := h.renderer.RenderHTMX(w, r, "welcome", "welcome", data); err != nil {
			log.Printf("Error rendering welcome: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	projects := h.store.GetProjects()

	// Scan projects_dir for discoverable folders
	var discovered []DirEntry
	if h.cfg.ProjectsDir != "" {
		entries, err := os.ReadDir(h.cfg.ProjectsDir)
		if err == nil {
			// Build set of registered paths for fast lookup
			registered := make(map[string]bool)
			for _, p := range projects {
				registered[p.Path] = true
			}

			for _, e := range entries {
				if !e.IsDir() || e.Name()[0] == '.' {
					continue
				}
				fullPath := filepath.Join(h.cfg.ProjectsDir, e.Name())
				if !registered[fullPath] {
					discovered = append(discovered, DirEntry{
						Name: e.Name(),
						Path: fullPath,
					})
				}
			}
		} else {
			log.Printf("Warning: could not scan projects dir %s: %v", h.cfg.ProjectsDir, err)
		}
	}

	data := map[string]any{
		"Title":      "ClawIDE - Dashboard",
		"Projects":   projects,
		"Discovered": discovered,
		"StartTour":  r.URL.Query().Get("tour") == "dashboard",
	}

	if err := h.renderer.RenderHTMX(w, r, "project-list", "project-list", data); err != nil {
		log.Printf("Error rendering dashboard: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
