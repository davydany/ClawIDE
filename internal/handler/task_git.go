package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

// TaskGitStatus returns the git status of .clawide/tasks.md within the project, reusing the
// clawideGitStatus helper shared with note_git.go. The subpath is a single file, which git
// handles naturally.
// GET /api/tasks/git-status?project_id=<id>
func (h *Handlers) TaskGitStatus(w http.ResponseWriter, r *http.Request) {
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
	resp := clawideGitStatus(project.Path, ".clawide/tasks.md")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// TaskGitCommit stages and commits changes to .clawide/tasks.md with a caller-provided message.
// POST /api/tasks/commit
func (h *Handlers) TaskGitCommit(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("TaskGitCommit error in %s: %v", project.Path, err)
		http.Error(w, "failed to commit: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
