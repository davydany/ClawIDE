package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
)

// BookmarkGitStatus returns the git status of files under .clawide/bookmarks/
// for the project identified by the project_id query parameter.
// GET /api/bookmarks/git-status?project_id=...
func (h *Handlers) BookmarkGitStatus(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	project, ok := h.store.GetProject(projectID)
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	resp := clawideGitStatus(project.Path, filepath.Join(clawideDirName, "bookmarks"))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BookmarkGitCommit stages and commits selected files under .clawide/bookmarks/.
// POST /api/bookmarks/commit
func (h *Handlers) BookmarkGitCommit(w http.ResponseWriter, r *http.Request) {
	var req clawideGitCommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.ProjectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}
	if len(req.Files) == 0 {
		http.Error(w, "no files selected", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "commit message is required", http.StatusBadRequest)
		return
	}

	project, ok := h.store.GetProject(req.ProjectID)
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	resp, err := clawideGitCommit(project.Path, req.Files, req.Message)
	if err != nil {
		log.Printf("Error committing bookmarks in %s: %v", project.Path, err)
		http.Error(w, "failed to commit: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
