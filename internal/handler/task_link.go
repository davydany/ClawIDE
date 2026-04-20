package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/davydany/ClawIDE/internal/git"
	"github.com/go-chi/chi/v5"
)

// LinkTaskBranch sets or clears the branch a task is linked to. The branch must exist in the
// resolved project (either as a local branch or as a branch of an existing worktree). An empty
// string unlinks. Global-scope tasks cannot be linked because they have no project to resolve
// branches against.
//
// PUT /api/tasks/{taskID}/linked-branch?project_id=<id>
// Body: {"branch": "feature/foo"}   // empty string unlinks
// Resp 200: {"task_id": "...", "linked_branch": "...", "worktree_path": "..."}
func (h *Handlers) LinkTaskBranch(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "task ID required", http.StatusBadRequest)
		return
	}

	var body struct {
		Branch string `json:"branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	body.Branch = strings.TrimSpace(body.Branch)

	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "linking a task to a branch requires a project scope (project_id query param)", http.StatusBadRequest)
		return
	}
	project, ok := h.store.GetProject(projectID)
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}
	taskStore, err := h.getProjectTaskStore(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	worktreePath := ""
	if body.Branch != "" {
		if !git.IsGitRepo(project.Path) {
			http.Error(w, "project is not a git repository", http.StatusBadRequest)
			return
		}
		if !branchExists(project.Path, body.Branch) {
			http.Error(w, "branch not found: "+body.Branch, http.StatusNotFound)
			return
		}
		worktreePath, _ = worktreePathForBranch(project.Path, body.Branch)
	}

	task, err := taskStore.SetLinkedBranch(taskID, body.Branch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"task_id":       task.ID,
		"linked_branch": task.LinkedBranch,
		"worktree_path": worktreePath,
	})
}

// branchExists returns true if the given branch is listable in the repo — either a local
// branch or a worktree's branch. Remote-only branches count as valid link targets because
// the user may create a worktree for them later.
func branchExists(repoPath, branch string) bool {
	branches, err := git.ListBranches(repoPath)
	if err != nil {
		log.Printf("LinkTaskBranch: ListBranches error: %v", err)
		return false
	}
	for _, b := range branches {
		if b.Name == branch {
			return true
		}
		// Remote branches are listed as "origin/feature/foo" — accept the unqualified form too.
		if b.IsRemote && b.Remote != "" && strings.TrimPrefix(b.Name, b.Remote+"/") == branch {
			return true
		}
	}
	return false
}

// worktreePathForBranch looks up the filesystem path of a worktree checked out to branch.
// Returns ("", false) if no worktree currently has that branch checked out.
func worktreePathForBranch(repoPath, branch string) (string, bool) {
	worktrees, err := git.ListWorktrees(repoPath)
	if err != nil {
		return "", false
	}
	for _, wt := range worktrees {
		if wt.Branch == branch {
			return wt.Path, true
		}
	}
	return "", false
}
