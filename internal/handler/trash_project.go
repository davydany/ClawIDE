package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/davydany/ClawIDE/internal/fsutil"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
)

// ListTrashedProjects returns all trashed projects as JSON.
// GET /api/trash/projects
func (h *Handlers) ListTrashedProjects(w http.ResponseWriter, r *http.Request) {
	items := h.store.GetTrashedProjects()
	if items == nil {
		items = []model.TrashedProject{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// RestoreTrashedProject moves a trashed project directory back to its
// original path and re-adds the Project record to the active store.
// POST /api/trash/projects/{tid}/restore
func (h *Handlers) RestoreTrashedProject(w http.ResponseWriter, r *http.Request) {
	trashID := chi.URLParam(r, "tid")

	tp, ok := h.store.GetTrashedProject(trashID)
	if !ok {
		http.Error(w, "trash item not found", http.StatusNotFound)
		return
	}

	// Verify the original path's parent still exists and the target is free.
	parent := filepath.Dir(tp.OriginalPath)
	if _, err := os.Stat(parent); err != nil {
		http.Error(w, "original parent directory no longer exists: "+parent, http.StatusConflict)
		return
	}
	if _, err := os.Stat(tp.OriginalPath); err == nil {
		http.Error(w, "original path is now occupied: "+tp.OriginalPath, http.StatusConflict)
		return
	}

	// Verify the trashed files still exist on disk.
	if _, err := os.Stat(tp.TrashedPath); err != nil {
		http.Error(w, "trashed files are missing on disk", http.StatusInternalServerError)
		return
	}

	if err := fsutil.MoveDir(tp.TrashedPath, tp.OriginalPath); err != nil {
		log.Printf("Error restoring trashed project %s: %v", tp.ID, err)
		http.Error(w, "failed to restore project: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Best-effort: remove the (now empty) trash parent dir wrapping the move.
	if wrapper := filepath.Dir(tp.TrashedPath); wrapper != "" {
		_ = os.Remove(wrapper)
	}

	// Rehydrate the project in the store.
	restored := tp.Project
	restored.UpdatedAt = time.Now()
	if err := h.store.AddProject(restored); err != nil {
		log.Printf("Error re-adding restored project: %v", err)
		http.Error(w, "files restored but failed to update state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.store.DeleteTrashedProject(trashID); err != nil {
		log.Printf("Error removing trash entry after restore: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"project_id": restored.ID,
	})
}

// PermanentlyDeleteTrashedProject removes a trashed project's files from
// disk and drops the trash record. Irreversible.
// DELETE /api/trash/projects/{tid}
func (h *Handlers) PermanentlyDeleteTrashedProject(w http.ResponseWriter, r *http.Request) {
	trashID := chi.URLParam(r, "tid")

	tp, ok := h.store.GetTrashedProject(trashID)
	if !ok {
		http.Error(w, "trash item not found", http.StatusNotFound)
		return
	}

	// Remove the full trash wrapper dir (trash/projects/<trashID>/) so the
	// enclosed basename folder is gone too.
	wrapper := filepath.Dir(tp.TrashedPath)
	if err := os.RemoveAll(wrapper); err != nil {
		log.Printf("Error permanently deleting trashed project files at %s: %v", wrapper, err)
		http.Error(w, "failed to delete trashed files: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.store.DeleteTrashedProject(trashID); err != nil {
		log.Printf("Error removing trash entry: %v", err)
		http.Error(w, "failed to delete trash entry", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
