package handler

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/davydany/ClawIDE/internal/editor"
)

// AvailableEditors returns detected editors and the current preference.
// GET /api/editors/available
func (h *Handlers) AvailableEditors(w http.ResponseWriter, r *http.Request) {
	editors := editor.DetectAvailable()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"editors":          editors,
		"preferred_editor": h.cfg.PreferredEditor,
	})
}

// OpenEditor opens the preferred editor in the given project directory.
// POST /api/editor/open
func (h *Handlers) OpenEditor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Directory string `json:"directory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if h.cfg.PreferredEditor == "" {
		http.Error(w, "No preferred editor configured", http.StatusBadRequest)
		return
	}

	if req.Directory == "" {
		http.Error(w, "directory is required", http.StatusBadRequest)
		return
	}

	// Security: resolve to absolute and ensure it's under ProjectsDir.
	absDir, err := filepath.Abs(req.Directory)
	if err != nil {
		http.Error(w, "Invalid directory path", http.StatusBadRequest)
		return
	}
	absProjects, err := filepath.Abs(h.cfg.ProjectsDir)
	if err != nil {
		http.Error(w, "Server configuration error", http.StatusInternalServerError)
		return
	}

	// Allow directory if it starts with the projects dir (covers worktrees
	// that live in sibling *-worktrees directories).
	if !strings.HasPrefix(absDir, absProjects) {
		// Also allow the exact parent of ProjectsDir for worktree dirs
		parentDir := filepath.Dir(absProjects)
		if !strings.HasPrefix(absDir, parentDir) {
			http.Error(w, "Directory is outside the allowed projects path", http.StatusForbidden)
			return
		}
	}

	if err := editor.OpenEditor(h.cfg.PreferredEditor, absDir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// OpenFolder opens the given directory in the OS file explorer.
// POST /api/editor/open-folder
func (h *Handlers) OpenFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Directory string `json:"directory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Directory == "" {
		http.Error(w, "directory is required", http.StatusBadRequest)
		return
	}

	// Security: resolve to absolute and ensure it's under ProjectsDir.
	absDir, err := filepath.Abs(req.Directory)
	if err != nil {
		http.Error(w, "Invalid directory path", http.StatusBadRequest)
		return
	}
	absProjects, err := filepath.Abs(h.cfg.ProjectsDir)
	if err != nil {
		http.Error(w, "Server configuration error", http.StatusInternalServerError)
		return
	}

	if !strings.HasPrefix(absDir, absProjects) {
		parentDir := filepath.Dir(absProjects)
		if !strings.HasPrefix(absDir, parentDir) {
			http.Error(w, "Directory is outside the allowed projects path", http.StatusForbidden)
			return
		}
	}

	if err := editor.OpenFileExplorer(absDir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
