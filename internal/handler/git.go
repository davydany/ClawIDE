package handler

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/davydany/ClawIDE/internal/git"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/go-chi/chi/v5"
)

// worktreeResponse is the JSON envelope for the worktree list endpoint.
type worktreeResponse struct {
	Worktrees []worktreeItem `json:"worktrees"`
}

// worktreeItem adds an ID field (base64-encoded path) to the git.Worktree
// so the frontend can reference individual worktrees in delete requests.
type worktreeItem struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Branch string `json:"branch"`
	HEAD   string `json:"head"`
	IsMain bool   `json:"is_main"`
}

// branchResponse is the JSON envelope for the branch list endpoint.
type branchResponse struct {
	Branches []git.Branch `json:"branches"`
}

// ListWorktrees returns a JSON array of worktrees for the project's repo.
func (h *Handlers) ListWorktrees(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	if !git.IsGitRepo(project.Path) {
		http.Error(w, "project path is not a git repository", http.StatusBadRequest)
		return
	}

	worktrees, err := git.ListWorktrees(project.Path)
	if err != nil {
		log.Printf("Error listing worktrees for %s: %v", project.Path, err)
		http.Error(w, "failed to list worktrees", http.StatusInternalServerError)
		return
	}

	items := make([]worktreeItem, 0, len(worktrees))
	for _, wt := range worktrees {
		items = append(items, worktreeItem{
			ID:     base64.URLEncoding.EncodeToString([]byte(wt.Path)),
			Path:   wt.Path,
			Branch: wt.Branch,
			HEAD:   wt.HEAD,
			IsMain: wt.IsMain,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(worktreeResponse{Worktrees: items})
}

// CreateWorktree creates a new git worktree for the branch specified in
// the "branch" form field. The worktree is placed in the conventional
// directory: {project}-worktrees/{branch}/
func (h *Handlers) CreateWorktree(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	if !git.IsGitRepo(project.Path) {
		http.Error(w, "project path is not a git repository", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	branch := r.FormValue("branch")
	if branch == "" {
		http.Error(w, "branch is required", http.StatusBadRequest)
		return
	}

	// Use empty targetDir to apply the conventional directory layout
	if err := git.CreateWorktree(project.Path, branch, ""); err != nil {
		log.Printf("Error creating worktree for branch %q in %s: %v", branch, project.Path, err)
		http.Error(w, "failed to create worktree: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "created",
		"branch": branch,
		"path":   git.WorktreeDir(project.Path, branch),
	})
}

// DeleteWorktree removes a worktree identified by the {wid} URL parameter,
// which is a base64 URL-encoded absolute path to the worktree directory.
func (h *Handlers) DeleteWorktree(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	if !git.IsGitRepo(project.Path) {
		http.Error(w, "project path is not a git repository", http.StatusBadRequest)
		return
	}

	wid := chi.URLParam(r, "wid")
	if wid == "" {
		http.Error(w, "worktree ID is required", http.StatusBadRequest)
		return
	}

	worktreePath, err := base64.URLEncoding.DecodeString(wid)
	if err != nil {
		http.Error(w, "invalid worktree ID", http.StatusBadRequest)
		return
	}

	if err := git.RemoveWorktree(project.Path, string(worktreePath)); err != nil {
		log.Printf("Error removing worktree %s: %v", string(worktreePath), err)
		http.Error(w, "failed to remove worktree: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
}

// CheckoutBranch switches the project's repo to the branch specified in the
// "branch" form field. POST /projects/{id}/api/checkout
func (h *Handlers) CheckoutBranch(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	if !git.IsGitRepo(project.Path) {
		http.Error(w, "project path is not a git repository", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	branch := r.FormValue("branch")
	if branch == "" {
		http.Error(w, "branch is required", http.StatusBadRequest)
		return
	}

	if err := git.CheckoutBranch(project.Path, branch); err != nil {
		log.Printf("Error checking out branch %q in %s: %v", branch, project.Path, err)
		http.Error(w, "failed to checkout branch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"branch": branch,
	})
}

// CreateBranch creates a new branch and checks it out. Form fields: "name"
// (branch name) and "base" (optional base branch).
// POST /projects/{id}/api/branches
func (h *Handlers) CreateBranch(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	if !git.IsGitRepo(project.Path) {
		http.Error(w, "project path is not a git repository", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	base := r.FormValue("base")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if err := git.CreateBranch(project.Path, name, base); err != nil {
		log.Printf("Error creating branch %q in %s: %v", name, project.Path, err)
		http.Error(w, "failed to create branch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "created",
		"branch": name,
	})
}

// ListBranches returns a JSON array of local and remote branches for the
// project's repository.
func (h *Handlers) ListBranches(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	if !git.IsGitRepo(project.Path) {
		http.Error(w, "project path is not a git repository", http.StatusBadRequest)
		return
	}

	branches, err := git.ListBranches(project.Path)
	if err != nil {
		log.Printf("Error listing branches for %s: %v", project.Path, err)
		http.Error(w, "failed to list branches", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(branchResponse{Branches: branches})
}
