package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/cron"
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
		Name           string `json:"name"`
		JobType        string `json:"job_type"`
		Agent          string `json:"agent"`
		Interval       string `json:"interval"`
		CronExpression string `json:"cron_expression"`
		Prompt         string `json:"prompt"`
		TargetPaneID   string `json:"target_pane_id"`
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
	if body.JobType == "cron" && !cron.IsSupported() {
		http.Error(w, "cron is not supported on this system", http.StatusBadRequest)
		return
	}
	if body.JobType == "cron" && strings.TrimSpace(body.CronExpression) == "" {
		http.Error(w, "cron_expression is required for cron jobs", http.StatusBadRequest)
		return
	}

	now := time.Now()
	job := model.ScheduledJob{
		ID:             uuid.New().String(),
		ProjectID:      project.ID,
		Name:           body.Name,
		JobType:        body.JobType,
		Agent:          body.Agent,
		Interval:       body.Interval,
		CronExpression: body.CronExpression,
		Prompt:         body.Prompt,
		TargetPaneID:   body.TargetPaneID,
		Status:         "idle",
		CreatedAt:      now,
		UpdatedAt:      now,
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
		Name           *string `json:"name"`
		JobType        *string `json:"job_type"`
		Agent          *string `json:"agent"`
		Interval       *string `json:"interval"`
		CronExpression *string `json:"cron_expression"`
		Prompt         *string `json:"prompt"`
		TargetPaneID   *string `json:"target_pane_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// If the job is currently running as a cron and the type or expression is
	// changing, remove the old crontab entry first.
	wasRunningCron := job.Status == "running" && job.JobType == "cron"

	if body.Name != nil {
		job.Name = *body.Name
	}
	if body.JobType != nil {
		job.JobType = *body.JobType
	}
	if body.Agent != nil {
		job.Agent = *body.Agent
	}
	if body.Interval != nil {
		job.Interval = *body.Interval
	}
	if body.CronExpression != nil {
		job.CronExpression = *body.CronExpression
	}
	if body.Prompt != nil {
		job.Prompt = *body.Prompt
	}
	if body.TargetPaneID != nil {
		job.TargetPaneID = *body.TargetPaneID
	}
	job.UpdatedAt = time.Now()

	// If we changed a running cron job, uninstall the old entry and mark idle.
	if wasRunningCron {
		if err := cron.Remove(job.ID); err != nil {
			log.Printf("cron remove on update error: %v", err)
		}
		job.Status = "idle"
	}

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

	// If it's a running cron job, clean up the crontab entry first.
	job, ok := h.store.GetScheduledJob(jid)
	if ok && job.JobType == "cron" && job.Status == "running" {
		if err := cron.Remove(jid); err != nil {
			log.Printf("cron remove on delete error: %v", err)
		}
	}

	if err := h.store.DeleteScheduledJob(jid); err != nil {
		http.Error(w, "scheduled job not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// StartScheduledJob starts the job. For loop jobs it sends the /loop command to
// the target pane; for cron jobs it installs a system crontab entry.
func (h *Handlers) StartScheduledJob(w http.ResponseWriter, r *http.Request) {
	jid := chi.URLParam(r, "jid")
	project := middleware.GetProject(r)
	job, ok := h.store.GetScheduledJob(jid)
	if !ok {
		http.Error(w, "scheduled job not found", http.StatusNotFound)
		return
	}

	switch job.JobType {
	case "cron":
		h.startCronJob(w, project, job)
	default: // "loop"
		h.startLoopJob(w, job)
	}
}

func (h *Handlers) startLoopJob(w http.ResponseWriter, job model.ScheduledJob) {
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
	if err := h.store.SetScheduledJobStatus(job.ID, "running", &now); err != nil {
		log.Printf("scheduled job status update error: %v", err)
	}

	job.Status = "running"
	job.LastRunAt = &now
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (h *Handlers) startCronJob(w http.ResponseWriter, project model.Project, job model.ScheduledJob) {
	if !cron.IsSupported() {
		http.Error(w, "cron is not supported on this system", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(job.CronExpression) == "" {
		http.Error(w, "cron expression is required", http.StatusBadRequest)
		return
	}

	logDir := filepath.Join(h.cfg.DataDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("failed to create cron log directory: %v", err)
	}
	logPath := filepath.Join(logDir, "cron-"+job.ID+".log")
	command := cron.BuildCommand(job.Agent, job.Prompt, project.Path, logPath)

	if err := cron.Install(job.ID, job.CronExpression, command); err != nil {
		log.Printf("cron install error: %v", err)
		http.Error(w, "failed to install crontab entry: "+err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	if err := h.store.SetScheduledJobStatus(job.ID, "running", &now); err != nil {
		log.Printf("scheduled job status update error: %v", err)
	}

	job.Status = "running"
	job.LastRunAt = &now
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// StopScheduledJob stops the job. For loop jobs it sends Ctrl+C to the target
// pane; for cron jobs it removes the crontab entry.
func (h *Handlers) StopScheduledJob(w http.ResponseWriter, r *http.Request) {
	jid := chi.URLParam(r, "jid")
	job, ok := h.store.GetScheduledJob(jid)
	if !ok {
		http.Error(w, "scheduled job not found", http.StatusNotFound)
		return
	}

	switch job.JobType {
	case "cron":
		if err := cron.Remove(job.ID); err != nil {
			log.Printf("cron remove error: %v", err)
		}
	default: // "loop"
		if job.TargetPaneID != "" {
			tmuxSession := tmux.TmuxName(job.TargetPaneID)
			if tmux.HasSession(tmuxSession) {
				if err := tmux.SendControl(tmuxSession, "C-c"); err != nil {
					log.Printf("scheduled job stop error (tmux C-c): %v", err)
				}
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

// CronSupported returns whether the system supports cron jobs. Called by the
// frontend to conditionally show/hide the cron option.
func (h *Handlers) CronSupported(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"supported": cron.IsSupported()})
}
