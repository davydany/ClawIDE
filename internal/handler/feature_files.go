package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// FeatureListFiles handles GET /projects/{id}/features/{fid}/api/files
// It delegates to the shared file listing logic using the feature's worktree
// path as root.
func (h *Handlers) FeatureListFiles(w http.ResponseWriter, r *http.Request) {
	featureID := chi.URLParam(r, "fid")
	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}
	listFilesForRoot(w, r, feature.WorktreePath)
}

// FeatureReadFile handles GET /projects/{id}/features/{fid}/api/file
func (h *Handlers) FeatureReadFile(w http.ResponseWriter, r *http.Request) {
	featureID := chi.URLParam(r, "fid")
	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}
	readFileFromRoot(w, r, feature.WorktreePath)
}

// FeatureWriteFile handles PUT /projects/{id}/features/{fid}/api/file
func (h *Handlers) FeatureWriteFile(w http.ResponseWriter, r *http.Request) {
	featureID := chi.URLParam(r, "fid")
	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}
	writeFileToRoot(w, r, feature.WorktreePath)
}
