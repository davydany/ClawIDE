package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/tmux"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) ListScheduledJobs(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	jobs := h.store.GetScheduledJobs(project.ID)
	if jobs == nil {
		jobs = []model.ScheduledJob{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jobs); err != nil {
		log.Printf("scheduled jobs list JSON encode error: %v", err)
	}
}

func (h *Handlers) CreateScheduledJob(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	var body struct {
		Name         string `json:"name"`
		JobType      string `json:"job_type"`
		Agent        string `json:"agent"`
		Interval     string `json:"interval"`
		Prompt       string `json:"prompt"`
		TargetPaneID string `json:"target_pane_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Prompt) == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}
	if body.JobType == "" {
		body.JobType = "loop"
	}
	if body.Agent == "" {
		body.Agent = "claude"
	}

	now := time.Now()
	job := model.ScheduledJob{
		ID:           uuid.New().String(),
		ProjectID:    project.ID,
		Name:         body.Name,
		JobType:      body.JobType,
		Agent:        body.Agent,
		Interval:     body.Interval,
		Prompt:       body.Prompt,
		TargetPaneID: body.TargetPaneID,
		Status:       "idle",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.store.AddScheduledJob(job); err != nil {
		log.Printf("scheduled job create error: %v", err)
		http.Error(w, "failed to create scheduled job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

func (h *Handlers) GetScheduledJob(w http.ResponseWriter, r *http.Request) {
	jid := chi.URLParam(r, "jid")
	job, ok := h.store.GetScheduledJob(jid)
	if !ok {
		http.Error(w, "scheduled job not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (h *Handlers) UpdateScheduledJob(w http.ResponseWriter, r *http.Request) {
	jid := chi.URLParam(r, "jid")
	job, ok := h.store.GetScheduledJob(jid)
	if !ok {
		http.Error(w, "scheduled job not found", http.StatusNotFound)
		return
	}

	var body struct {
		Name         *string `json:"name"`
		Agent        *string `json:"agent"`
		Interval     *string `json:"interval"`
		Prompt       *string `json:"prompt"`
		TargetPaneID *string `json:"target_pane_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Name != nil {
		job.Name = *body.Name
	}
	if body.Agent != nil {
		job.Agent = *body.Agent
	}
	if body.Interval != nil {
		job.Interval = *body.Interval
	}
	if body.Prompt != nil {
		job.Prompt = *body.Prompt
	}
	if body.TargetPaneID != nil {
		job.TargetPaneID = *body.TargetPaneID
	}
	job.UpdatedAt = time.Now()

	if err := h.store.UpdateScheduledJob(job); err != nil {
		log.Printf("scheduled job update error: %v", err)
		http.Error(w, "failed to update scheduled job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (h *Handlers) DeleteScheduledJob(w http.ResponseWriter, r *http.Request) {
	jid := chi.URLParam(r, "jid")
	if err := h.store.DeleteScheduledJob(jid); err != nil {
		http.Error(w, "scheduled job not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// StartScheduledJob sends the /loop command to the target pane via tmux.
func (h *Handlers) StartScheduledJob(w http.ResponseWriter, r *http.Request) {
	jid := chi.URLParam(r, "jid")
	job, ok := h.store.GetScheduledJob(jid)
	if !ok {
		http.Error(w, "scheduled job not found", http.StatusNotFound)
		return
	}

	if job.TargetPaneID == "" {
		http.Error(w, "no target pane configured", http.StatusBadRequest)
		return
	}

	tmuxSession := tmux.TmuxName(job.TargetPaneID)
	if !tmux.HasSession(tmuxSession) {
		http.Error(w, "target pane session not found — is the pane still open?", http.StatusConflict)
		return
	}

	// Build the /loop command
	cmd := "/loop"
	if job.Interval != "" {
		cmd += " " + job.Interval
	}
	cmd += " " + job.Prompt

	if err := tmux.SendKeys(tmuxSession, cmd); err != nil {
		log.Printf("scheduled job start error (tmux send-keys): %v", err)
		http.Error(w, "failed to send command to pane", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	if err := h.store.SetScheduledJobStatus(jid, "running", &now); err != nil {
		log.Printf("scheduled job status update error: %v", err)
	}

	job.Status = "running"
	job.LastRunAt = &now
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// StopScheduledJob sends Ctrl+C to the target pane to interrupt the loop.
func (h *Handlers) StopScheduledJob(w http.ResponseWriter, r *http.Request) {
	jid := chi.URLParam(r, "jid")
	job, ok := h.store.GetScheduledJob(jid)
	if !ok {
		http.Error(w, "scheduled job not found", http.StatusNotFound)
		return
	}

	if job.TargetPaneID != "" {
		tmuxSession := tmux.TmuxName(job.TargetPaneID)
		if tmux.HasSession(tmuxSession) {
			if err := tmux.SendControl(tmuxSession, "C-c"); err != nil {
				log.Printf("scheduled job stop error (tmux C-c): %v", err)
			}
		}
	}

	if err := h.store.SetScheduledJobStatus(jid, "idle", nil); err != nil {
		log.Printf("scheduled job status update error: %v", err)
	}

	job.Status = "idle"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}
