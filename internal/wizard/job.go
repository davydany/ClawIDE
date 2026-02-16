package wizard

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the current state of a wizard job.
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusRunning    JobStatus = "running"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusRolledBack JobStatus = "rolled_back"
)

// JobStep represents an individual step within a wizard job.
type JobStep struct {
	Name      string    `json:"name"`
	Status    JobStatus `json:"status"`
	Message   string    `json:"message,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
}

// Job tracks the progress and state of a single project generation.
type Job struct {
	ID        string        `json:"id"`
	Request   WizardRequest `json:"request"`
	Status    JobStatus     `json:"status"`
	Steps     []JobStep     `json:"steps"`
	Error     string        `json:"error,omitempty"`
	OutputDir string        `json:"output_dir,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`

	mu sync.RWMutex
}

// NewJob creates a new pending job for the given request.
func NewJob(req WizardRequest) *Job {
	now := time.Now()
	return &Job{
		ID:        uuid.New().String(),
		Request:   req,
		Status:    JobStatusPending,
		Steps:     defaultSteps(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// defaultSteps returns the standard steps for project generation.
func defaultSteps() []JobStep {
	return []JobStep{
		{Name: "validate", Status: JobStatusPending},
		{Name: "create_directory", Status: JobStatusPending},
		{Name: "copy_templates", Status: JobStatusPending},
		{Name: "copy_docs", Status: JobStatusPending},
		{Name: "generate_claude_md", Status: JobStatusPending},
		{Name: "init_git", Status: JobStatusPending},
		{Name: "install_deps", Status: JobStatusPending},
	}
}

// StartStep marks a step as running.
func (j *Job) StartStep(name string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for i := range j.Steps {
		if j.Steps[i].Name == name {
			j.Steps[i].Status = JobStatusRunning
			j.Steps[i].StartedAt = time.Now()
			break
		}
	}
	j.Status = JobStatusRunning
	j.UpdatedAt = time.Now()
}

// CompleteStep marks a step as completed with an optional message.
func (j *Job) CompleteStep(name, message string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for i := range j.Steps {
		if j.Steps[i].Name == name {
			j.Steps[i].Status = JobStatusCompleted
			j.Steps[i].Message = message
			j.Steps[i].EndedAt = time.Now()
			break
		}
	}
	j.UpdatedAt = time.Now()
}

// FailStep marks a step as failed and records the error.
func (j *Job) FailStep(name string, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for i := range j.Steps {
		if j.Steps[i].Name == name {
			j.Steps[i].Status = JobStatusFailed
			j.Steps[i].Message = err.Error()
			j.Steps[i].EndedAt = time.Now()
			break
		}
	}
	j.Status = JobStatusFailed
	j.Error = err.Error()
	j.UpdatedAt = time.Now()
}

// Complete marks the entire job as completed.
func (j *Job) Complete(outputDir string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = JobStatusCompleted
	j.OutputDir = outputDir
	j.UpdatedAt = time.Now()
}

// Fail marks the entire job as failed with the given error.
func (j *Job) Fail(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = JobStatusFailed
	j.Error = err.Error()
	j.UpdatedAt = time.Now()
}

// MarkRolledBack marks the job as rolled back after a failure cleanup.
func (j *Job) MarkRolledBack() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = JobStatusRolledBack
	j.UpdatedAt = time.Now()
}

// Snapshot returns a read-safe copy of the job state for serialization.
func (j *Job) Snapshot() Job {
	j.mu.RLock()
	defer j.mu.RUnlock()
	snap := *j
	snap.Steps = make([]JobStep, len(j.Steps))
	copy(snap.Steps, j.Steps)
	return snap
}

// JobTracker provides thread-safe storage and retrieval of wizard jobs.
type JobTracker struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

// NewJobTracker creates a new empty job tracker.
func NewJobTracker() *JobTracker {
	return &JobTracker{
		jobs: make(map[string]*Job),
	}
}

// Add registers a new job and returns it.
func (jt *JobTracker) Add(req WizardRequest) *Job {
	job := NewJob(req)
	jt.mu.Lock()
	defer jt.mu.Unlock()
	jt.jobs[job.ID] = job
	return job
}

// Get retrieves a job by ID.
func (jt *JobTracker) Get(id string) (*Job, error) {
	jt.mu.RLock()
	defer jt.mu.RUnlock()
	job, ok := jt.jobs[id]
	if !ok {
		return nil, fmt.Errorf("job %s not found", id)
	}
	return job, nil
}

// Remove deletes a job from the tracker.
func (jt *JobTracker) Remove(id string) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	delete(jt.jobs, id)
}

// List returns snapshots of all tracked jobs.
func (jt *JobTracker) List() []Job {
	jt.mu.RLock()
	defer jt.mu.RUnlock()
	result := make([]Job, 0, len(jt.jobs))
	for _, job := range jt.jobs {
		result = append(result, job.Snapshot())
	}
	return result
}
