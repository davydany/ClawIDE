package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/wizard"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ShowWizard renders the project wizard page or returns languages/frameworks as JSON.
func (h *Handlers) ShowWizard(w http.ResponseWriter, r *http.Request) {
	languages := wizard.SupportedLanguages()

	// If it's an API request (Accept: application/json), return JSON
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"languages":   languages,
			"projects_dir": h.cfg.ProjectsDir,
		})
		return
	}

	// Render wizard HTML page
	data := map[string]any{
		"Title":       "New Project - ClawIDE",
		"Languages":   languages,
		"ProjectsDir": h.cfg.ProjectsDir,
	}
	if err := h.renderer.RenderHTMX(w, r, "wizard", "wizard", data); err != nil {
		log.Printf("Error rendering wizard: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// GetWizardLanguages returns the supported languages and frameworks as JSON.
func (h *Handlers) GetWizardLanguages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"languages":   wizard.SupportedLanguages(),
		"projects_dir": h.cfg.ProjectsDir,
	})
}

// createWizardRequest is the JSON body for project creation.
type createWizardRequest struct {
	ProjectName     string `json:"project_name"`
	Language        string `json:"language"`
	Framework       string `json:"framework"`
	OutputDir       string `json:"output_dir"`
	Description     string `json:"description"`
	DocPRD          string `json:"doc_prd"`
	DocUIUX         string `json:"doc_uiux"`
	DocArchitecture string `json:"doc_architecture"`
	DocOther        string `json:"doc_other"`
}

// wizardStatusResponse is the JSON response for job status polling.
type wizardStatusResponse struct {
	JobID     string           `json:"job_id"`
	Status    wizard.JobStatus `json:"status"`
	Steps     []wizard.JobStep `json:"steps"`
	Error     string           `json:"error,omitempty"`
	OutputDir string           `json:"output_dir,omitempty"`
	ProjectID string           `json:"project_id,omitempty"`
}

// CreateProjectFromWizard validates the request, creates an async generation job,
// and returns the job ID for status polling.
func (h *Handlers) CreateProjectFromWizard(w http.ResponseWriter, r *http.Request) {
	var body createWizardRequest

	// Support both JSON and form data
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
		body = createWizardRequest{
			ProjectName:     r.FormValue("project_name"),
			Language:        r.FormValue("language"),
			Framework:       r.FormValue("framework"),
			OutputDir:       r.FormValue("output_dir"),
			Description:     r.FormValue("description"),
			DocPRD:          r.FormValue("doc_prd"),
			DocUIUX:         r.FormValue("doc_uiux"),
			DocArchitecture: r.FormValue("doc_architecture"),
			DocOther:        r.FormValue("doc_other"),
		}
	}

	// Default output directory to configured projects dir
	if body.OutputDir == "" {
		body.OutputDir = h.cfg.ProjectsDir
	}

	wizReq := wizard.WizardRequest{
		ProjectName:     body.ProjectName,
		Language:        body.Language,
		Framework:       body.Framework,
		OutputDir:       body.OutputDir,
		Description:     body.Description,
		DocPRD:          body.DocPRD,
		DocUIUX:         body.DocUIUX,
		DocArchitecture: body.DocArchitecture,
		DocOther:        body.DocOther,
	}

	// Validate synchronously before creating the job
	validation := wizard.Validate(wizReq)
	if !validation.IsValid() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": validation.ErrorMap(),
		})
		return
	}

	// Create and track the job
	job := h.wizardJobs.Add(wizReq)

	// Run generation asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := h.wizardGenerator.Generate(ctx, job); err != nil {
			log.Printf("Wizard generation failed for job %s: %v", job.ID, err)
			return
		}

		// On success, register the project in the store
		snap := job.Snapshot()
		if snap.Status == wizard.JobStatusCompleted && snap.OutputDir != "" {
			now := time.Now()
			project := model.Project{
				ID:        uuid.New().String(),
				Name:      wizReq.ProjectName,
				Path:      snap.OutputDir,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := h.store.AddProject(project); err != nil {
				log.Printf("Warning: project generated but failed to register in store: %v", err)
			} else {
				// Store project ID on the job for frontend redirect
				job.Complete(snap.OutputDir)
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": job.ID,
	})
}

// GetWizardStatus returns the current status of a wizard generation job.
func (h *Handlers) GetWizardStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	job, err := h.wizardJobs.Get(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	snap := job.Snapshot()

	// Try to find project ID if generation completed
	var projectID string
	if snap.Status == wizard.JobStatusCompleted && snap.OutputDir != "" {
		for _, p := range h.store.GetProjects() {
			if p.Path == snap.OutputDir {
				projectID = p.ID
				break
			}
		}
	}

	resp := wizardStatusResponse{
		JobID:     snap.ID,
		Status:    snap.Status,
		Steps:     snap.Steps,
		Error:     snap.Error,
		OutputDir: snap.OutputDir,
		ProjectID: projectID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ValidateWizardField performs inline validation on a single field for real-time feedback.
func (h *Handlers) ValidateWizardField(w http.ResponseWriter, r *http.Request) {
	var body createWizardRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Default output directory
	if body.OutputDir == "" {
		body.OutputDir = h.cfg.ProjectsDir
	}

	wizReq := wizard.WizardRequest{
		ProjectName:     body.ProjectName,
		Language:        body.Language,
		Framework:       body.Framework,
		OutputDir:       body.OutputDir,
		Description:     body.Description,
		DocPRD:          body.DocPRD,
		DocUIUX:         body.DocUIUX,
		DocArchitecture: body.DocArchitecture,
		DocOther:        body.DocOther,
	}

	field := r.URL.Query().Get("field")
	result := wizard.Validate(wizReq)

	w.Header().Set("Content-Type", "application/json")

	errMap := result.ErrorMap()
	if field != "" {
		// Return error only for the requested field
		if msg, ok := errMap[field]; ok {
			json.NewEncoder(w).Encode(map[string]any{
				"valid": false,
				"field": field,
				"error": msg,
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"valid": true,
			"field": field,
		})
		return
	}

	// Return all errors
	json.NewEncoder(w).Encode(map[string]any{
		"valid":  result.IsValid(),
		"errors": errMap,
	})
}

// ScanProjectsDir returns subdirectories of the configured projects directory.
func (h *Handlers) ScanProjectsDir(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir = h.cfg.ProjectsDir
	}

	// Expand ~ in path
	if len(dir) > 0 && dir[0] == '~' {
		home, _ := filepath.Abs(dir)
		dir = home
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"projects_dir": h.cfg.ProjectsDir,
	})
}
