package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/davydany/ClawIDE/internal/git"
)

// clawideDirName is the project-local configuration directory.
const clawideDirName = ".clawide"

// gitStatusResponse wraps file status list with metadata for the git status endpoint.
type gitStatusResponse struct {
	IsGitRepo  bool             `json:"is_git_repo"`
	IsIgnored  bool             `json:"is_ignored"`
	Files      []git.FileStatus `json:"files"`
	HasConflict bool            `json:"has_conflict"`
}

// gitCommitResponse is returned after a successful commit.
type gitCommitResponse struct {
	Status     string `json:"status"`
	CommitHash string `json:"commit_hash,omitempty"`
}

// clawidGitCommitRequest is the JSON body for clawide commit endpoints.
type clawideGitCommitRequest struct {
	ProjectID string   `json:"project_id"`
	Files     []string `json:"files"`
	Message   string   `json:"message"`
}

// NoteGitStatus returns the git status of files under .clawide/notes/ for
// the project identified by the project_id query parameter.
// GET /api/notes/git-status?project_id=...
func (h *Handlers) NoteGitStatus(w http.ResponseWriter, r *http.Request) {
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

	resp := clawideGitStatus(project.Path, filepath.Join(clawideDirName, "notes"))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// NoteGitCommit stages and commits selected files under .clawide/notes/.
// POST /api/notes/commit
func (h *Handlers) NoteGitCommit(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("Error committing notes in %s: %v", project.Path, err)
		http.Error(w, "failed to commit: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- shared helpers ---

// clawideGitStatus returns the git status for a subpath within a project.
func clawideGitStatus(projectPath, subPath string) gitStatusResponse {
	resp := gitStatusResponse{
		Files: []git.FileStatus{},
	}

	if !git.IsGitRepo(projectPath) {
		return resp
	}
	resp.IsGitRepo = true

	// Check if .clawide is gitignored
	if git.IsPathIgnored(projectPath, clawideDirName) {
		resp.IsIgnored = true
		return resp
	}

	files, err := git.StatusForPath(projectPath, subPath)
	if err != nil {
		log.Printf("Error getting git status for %s in %s: %v", subPath, projectPath, err)
		return resp
	}
	if files != nil {
		resp.Files = files
	}

	// Check for merge conflicts (U status)
	for _, f := range resp.Files {
		if f.Status == "U" {
			resp.HasConflict = true
			break
		}
	}

	return resp
}

// clawideGitCommit stages the given files and creates a commit.
func clawideGitCommit(projectPath string, files []string, message string) (gitCommitResponse, error) {
	if !git.IsGitRepo(projectPath) {
		return gitCommitResponse{}, fmt.Errorf("not a git repository")
	}

	if err := git.Add(projectPath, files); err != nil {
		return gitCommitResponse{}, err
	}

	if err := git.Commit(projectPath, message); err != nil {
		return gitCommitResponse{}, err
	}

	hash, _ := git.LastCommitHash(projectPath)
	return gitCommitResponse{
		Status:     "committed",
		CommitHash: hash,
	}, nil
}
