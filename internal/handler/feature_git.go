package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/davydany/ClawIDE/internal/git"
	"github.com/davydany/ClawIDE/internal/middleware"
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

// FeatureMerge merges the feature branch into the main branch, then
// cleans up the feature (worktree, sessions, branch, store record).
// POST /projects/{id}/features/{fid}/api/merge
func (h *Handlers) FeatureMerge(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	// 1. Detect the main integration branch.
	mainBranch, err := git.DetectMainBranch(project.Path)
	if err != nil {
		http.Error(w, "could not detect main branch: "+err.Error(), http.StatusInternalServerError)
		return
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
