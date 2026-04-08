package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/davydany/ClawIDE/internal/git"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ListTrashedFeatures returns all trashed features as JSON.
// GET /api/trash
func (h *Handlers) ListTrashedFeatures(w http.ResponseWriter, r *http.Request) {
	items := h.store.GetTrashedFeatures()
	if items == nil {
		items = []model.TrashedFeature{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// RestoreTrashedFeature restores a feature from the trash bin by recreating
// its worktree from the preserved branch.
// POST /api/trash/{tid}/restore
func (h *Handlers) RestoreTrashedFeature(w http.ResponseWriter, r *http.Request) {
	trashID := chi.URLParam(r, "tid")

	tf, ok := h.store.GetTrashedFeature(trashID)
	if !ok {
		http.Error(w, "trash item not found", http.StatusNotFound)
		return
	}

	// Check if the project still exists.
	project, projectExists := h.store.GetProject(tf.ProjectID)
	if !projectExists {
		http.Error(w, "project no longer exists — cannot restore", http.StatusConflict)
		return
	}

	// Check if the branch still exists.
	branches, err := git.ListBranches(project.Path)
	if err != nil {
		log.Printf("Error listing branches for restore: %v", err)
		http.Error(w, "failed to check branches", http.StatusInternalServerError)
		return
	}
	branchFound := false
	for _, b := range branches {
		if b.Name == tf.Feature.BranchName {
			branchFound = true
			break
		}
	}
	if !branchFound {
		http.Error(w, "branch '"+tf.Feature.BranchName+"' no longer exists in the repository", http.StatusConflict)
		return
	}

	// Recreate the working directory based on workspace type.
	var workDir string
	if tf.Feature.IsClone() {
		workDir = git.CloneDir(project.Path, tf.Feature.BranchName)
		if err := git.CloneLocal(project.Path, workDir, tf.Feature.BranchName); err != nil {
			log.Printf("Error recreating clone for restore: %v", err)
			http.Error(w, "failed to recreate clone: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		workDir = git.WorktreeDir(project.Path, tf.Feature.BranchName)
		if err := git.CreateWorktree(project.Path, tf.Feature.BranchName, workDir); err != nil {
			log.Printf("Error recreating worktree for restore: %v", err)
			http.Error(w, "failed to recreate worktree: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Restore the feature with a new ID and updated paths.
	now := time.Now()
	restored := tf.Feature
	restored.ID = uuid.New().String()
	restored.WorktreePath = workDir
	restored.UpdatedAt = now

	if err := h.store.AddFeature(restored); err != nil {
		log.Printf("Error restoring feature: %v", err)
		http.Error(w, "failed to restore feature", http.StatusInternalServerError)
		return
	}

	// Create an initial session in the restored feature workspace.
	paneID := uuid.New().String()
	sess := model.Session{
		ID:        uuid.New().String(),
		ProjectID: project.ID,
		FeatureID: restored.ID,
		Name:      "Session " + now.Format("15:04"),
		WorkDir:   workDir,
		Layout:    model.NewAgentPane(paneID),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.store.AddSession(sess); err != nil {
		log.Printf("Error creating session for restored feature: %v", err)
	}

	// Remove from trash.
	if err := h.store.DeleteTrashedFeature(trashID); err != nil {
		log.Printf("Error removing trash entry after restore: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"feature_id": restored.ID,
		"project_id": project.ID,
	})
}

// PermanentlyDeleteTrashedFeature permanently removes a trashed feature
// and deletes its git branch.
// DELETE /api/trash/{tid}
func (h *Handlers) PermanentlyDeleteTrashedFeature(w http.ResponseWriter, r *http.Request) {
	trashID := chi.URLParam(r, "tid")

	tf, ok := h.store.GetTrashedFeature(trashID)
	if !ok {
		http.Error(w, "trash item not found", http.StatusNotFound)
		return
	}

	// Best-effort branch deletion.
	if err := git.DeleteBranch(tf.ProjectPath, tf.Feature.BranchName); err != nil {
		log.Printf("Warning: could not delete branch %s: %v", tf.Feature.BranchName, err)
	}

	if err := h.store.DeleteTrashedFeature(trashID); err != nil {
		log.Printf("Error permanently deleting trash item: %v", err)
		http.Error(w, "failed to delete", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
