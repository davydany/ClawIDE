package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/davydany/ClawIDE/internal/git"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/go-chi/chi/v5"
)

// reviewFilesResponse is the JSON response for the review files endpoint.
type reviewFilesResponse struct {
	Files         []git.DiffEntry    `json:"files"`
	Stats         git.DiffStatResult `json:"stats"`
	MainBranch    string             `json:"main_branch"`
	FeatureBranch string             `json:"feature_branch"`
}

// reviewAnnotation represents a single AI review annotation.
type reviewAnnotation struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	EndLine  int    `json:"end_line,omitempty"`
	Comment  string `json:"comment"`
	Severity string `json:"severity"`
}

// reviewAnnotationsResponse is the JSON response for the annotations endpoint.
type reviewAnnotationsResponse struct {
	Status      string             `json:"status"`
	Annotations []reviewAnnotation `json:"annotations"`
}

// FeatureReviewFiles returns the list of changed files between the feature
// branch and the main branch, along with diff statistics.
// GET /projects/{id}/features/{fid}/api/review/files
func (h *Handlers) FeatureReviewFiles(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	mainBranch, err := git.DetectMainBranch(project.Path)
	if err != nil {
		log.Printf("Error detecting main branch for %s: %v", project.Path, err)
		http.Error(w, "could not detect main branch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	files, err := git.DiffNameStatus(project.Path, mainBranch, feature.BranchName)
	if err != nil {
		log.Printf("Error getting diff for %s: %v", project.Path, err)
		http.Error(w, "failed to get diff: "+err.Error(), http.StatusInternalServerError)
		return
	}

	stats, err := git.DiffStat(project.Path, mainBranch, feature.BranchName)
	if err != nil {
		log.Printf("Error getting diff stats for %s: %v", project.Path, err)
		// Non-fatal, proceed with empty stats
		stats = git.DiffStatResult{}
	}

	if files == nil {
		files = []git.DiffEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviewFilesResponse{
		Files:         files,
		Stats:         stats,
		MainBranch:    mainBranch,
		FeatureBranch: feature.BranchName,
	})
}

// FeatureReviewFileContent returns the content of a file at a specific ref
// (main or feature branch).
// GET /projects/{id}/features/{fid}/api/review/file-content?path=...&ref=main|feature
func (h *Handlers) FeatureReviewFileContent(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	refParam := r.URL.Query().Get("ref")
	if refParam == "" {
		refParam = "feature"
	}

	var actualRef string
	switch refParam {
	case "main":
		mainBranch, err := git.DetectMainBranch(project.Path)
		if err != nil {
			http.Error(w, "could not detect main branch: "+err.Error(), http.StatusInternalServerError)
			return
		}
		actualRef = mainBranch
	case "feature":
		actualRef = feature.BranchName
	default:
		http.Error(w, "ref must be 'main' or 'feature'", http.StatusBadRequest)
		return
	}

	content, isBinary, err := git.ShowFile(project.Path, actualRef, filePath)
	if err != nil {
		log.Printf("Error showing file %s at %s: %v", filePath, actualRef, err)
		http.Error(w, "failed to read file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if isBinary {
		http.Error(w, "binary file", http.StatusConflict)
		return
	}

	// File doesn't exist at this ref
	if content == "" && err == nil {
		w.Header().Set("X-File-Status", "not-found")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(content))
}

// FeatureReviewAnnotations returns AI review annotations from
// .clawide-review.json in the feature worktree.
// GET /projects/{id}/features/{fid}/api/review/annotations
func (h *Handlers) FeatureReviewAnnotations(w http.ResponseWriter, r *http.Request) {
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	annotationsPath := filepath.Join(feature.WorktreePath, ".clawide-review.json")
	data, err := os.ReadFile(annotationsPath)
	if err != nil {
		// File doesn't exist yet — not started
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reviewAnnotationsResponse{
			Status:      "not-started",
			Annotations: []reviewAnnotation{},
		})
		return
	}

	var annotations []reviewAnnotation
	if err := json.Unmarshal(data, &annotations); err != nil {
		log.Printf("Error parsing annotations file %s: %v", annotationsPath, err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reviewAnnotationsResponse{
			Status:      "error",
			Annotations: []reviewAnnotation{},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviewAnnotationsResponse{
		Status:      "complete",
		Annotations: annotations,
	})
}
