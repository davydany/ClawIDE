package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/davydany/ClawIDE/internal/git"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
)

// statusResponse wraps the file status list for JSON encoding.
type statusResponse struct {
	Files []git.FileStatus `json:"files"`
}

// commitRequest is the JSON body for the commit endpoint.
type commitRequest struct {
	Files   []string `json:"files"`
	Message string   `json:"message"`
}

// FeatureGitStatus returns the git status of the feature's worktree as JSON.
// GET /projects/{id}/features/{fid}/api/status
func (h *Handlers) FeatureGitStatus(w http.ResponseWriter, r *http.Request) {
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	files, err := git.Status(feature.WorktreePath)
	if err != nil {
		log.Printf("Error getting git status for %s: %v", feature.WorktreePath, err)
		http.Error(w, "failed to get git status", http.StatusInternalServerError)
		return
	}

	if files == nil {
		files = []git.FileStatus{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statusResponse{Files: files})
}

// FeatureGitCommit stages the selected files and creates a commit in the
// feature's worktree.
// POST /projects/{id}/features/{fid}/api/commit
func (h *Handlers) FeatureGitCommit(w http.ResponseWriter, r *http.Request) {
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	var req commitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
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

	// Stage the selected files.
	if err := git.Add(feature.WorktreePath, req.Files); err != nil {
		log.Printf("Error staging files in %s: %v", feature.WorktreePath, err)
		http.Error(w, "failed to stage files: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit.
	if err := git.Commit(feature.WorktreePath, req.Message); err != nil {
		log.Printf("Error committing in %s: %v", feature.WorktreePath, err)
		http.Error(w, "failed to commit: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "committed"})
}

// FeaturePullMain fetches origin and merges a branch into the workspace.
// For feature-type workspaces, it pulls the project's active/main branch.
// For branch-type workspaces, it accepts an optional source_branch parameter.
// POST /projects/{id}/features/{fid}/api/pull-main
func (h *Handlers) FeaturePullMain(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	var branch string

	if feature.IsClone() {
		// Branch-type: accept source_branch from request body.
		var req struct {
			SourceBranch string `json:"source_branch"`
		}
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&req)
		}
		branch = req.SourceBranch
	}

	// Fall back to project's active branch or auto-detect.
	if branch == "" {
		branch = project.ActiveBranch
	}
	if branch == "" {
		detected, err := git.DetectMainBranch(project.Path)
		if err != nil {
			http.Error(w, "could not detect main branch: "+err.Error(), http.StatusInternalServerError)
			return
		}
		branch = detected
	}

	if err := git.PullFromBranch(feature.WorktreePath, "origin", branch); err != nil {
		log.Printf("Error pulling %s in feature %s: %v", branch, feature.WorktreePath, err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "pulled"})
}

// FeatureMerge merges the workspace branch into a target branch.
// For feature-type workspaces: merges into main in the project root, then
// cleans up the worktree, branch, sessions, and store record.
// For branch-type workspaces: merges into a specified target branch within
// the clone, pushes to origin, and keeps the workspace intact.
// POST /projects/{id}/features/{fid}/api/merge
func (h *Handlers) FeatureMerge(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	if feature.IsClone() {
		h.branchMerge(w, r, project, feature)
		return
	}
	h.featureMerge(w, r, project, feature)
}

// featureMerge handles the merge-and-cleanup flow for worktree-backed features.
func (h *Handlers) featureMerge(w http.ResponseWriter, r *http.Request, project model.Project, feature model.Feature) {
	featureID := feature.ID

	// 1. Resolve the base branch (project's active branch or auto-detect).
	mainBranch := project.ActiveBranch
	if mainBranch == "" {
		detected, err := git.DetectMainBranch(project.Path)
		if err != nil {
			http.Error(w, "could not detect main branch: "+err.Error(), http.StatusInternalServerError)
			return
		}
		mainBranch = detected
	}

	// 2. Save the current branch so we can restore it later.
	originalBranch, _ := git.CurrentBranch(project.Path)

	// 3. Checkout main in the project root.
	if err := git.CheckoutBranch(project.Path, mainBranch); err != nil {
		http.Error(w, "failed to checkout main branch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Merge the feature branch.
	if err := git.Merge(project.Path, feature.BranchName); err != nil {
		// Restore original branch on conflict.
		if originalBranch != "" {
			git.CheckoutBranch(project.Path, originalBranch)
		}
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// 5. Restore the original branch if it wasn't the feature branch.
	if originalBranch != "" && originalBranch != feature.BranchName {
		git.CheckoutBranch(project.Path, originalBranch)
	}

	// 6. Destroy all feature PTY sessions.
	sessions := h.store.GetFeatureSessions(featureID)
	for _, sess := range sessions {
		if sess.Layout != nil {
			for _, paneID := range sess.Layout.CollectLeaves() {
				if err := h.ptyManager.DestroySession(paneID); err != nil {
					log.Printf("Error destroying pane %s: %v", paneID, err)
				}
			}
		}
	}

	// 7. Remove the git worktree.
	if err := git.RemoveWorktree(project.Path, feature.WorktreePath); err != nil {
		log.Printf("Error removing worktree %s: %v", feature.WorktreePath, err)
	}

	// 8. Delete the feature branch.
	if err := git.DeleteBranch(project.Path, feature.BranchName); err != nil {
		log.Printf("Error deleting branch %s: %v", feature.BranchName, err)
	}

	// 9. Delete the feature from the store (cascades to sessions).
	if err := h.store.DeleteFeature(featureID); err != nil {
		log.Printf("Error deleting feature from store: %v", err)
		http.Error(w, "failed to clean up feature record", http.StatusInternalServerError)
		return
	}

	// 10. Redirect to project workspace.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "merged",
		"redirect": "/projects/" + project.ID + "/",
	})
}

// branchMerge handles the merge flow for clone-backed branch workspaces.
// It merges the branch into a user-specified target within the clone and
// pushes to origin. The workspace is kept intact.
func (h *Handlers) branchMerge(w http.ResponseWriter, r *http.Request, project model.Project, feature model.Feature) {
	var req struct {
		TargetBranch string `json:"target_branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetBranch == "" {
		http.Error(w, "target_branch is required", http.StatusBadRequest)
		return
	}

	clonePath := feature.WorktreePath

	// 1. Fetch latest from origin.
	if err := git.FetchAll(clonePath); err != nil {
		log.Printf("Error fetching in clone %s: %v", clonePath, err)
	}

	// 2. Checkout target branch in the clone.
	if err := git.CheckoutBranch(clonePath, req.TargetBranch); err != nil {
		http.Error(w, "failed to checkout target branch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Merge the workspace branch into the target.
	if err := git.Merge(clonePath, feature.BranchName); err != nil {
		// Abort and switch back to the workspace branch.
		git.CheckoutBranch(clonePath, feature.BranchName)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// 4. Push the target branch to origin.
	if err := git.PushBranch(clonePath, "origin", req.TargetBranch); err != nil {
		// Switch back to workspace branch even on push failure.
		git.CheckoutBranch(clonePath, feature.BranchName)
		http.Error(w, "merge succeeded but push failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Switch back to the workspace branch.
	git.CheckoutBranch(clonePath, feature.BranchName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "merged"})
}
