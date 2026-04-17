package model

import "time"

// ScheduledJob represents a managed scheduled job (e.g. a loop) attached to a
// project. In v1 only JobType "loop" is supported; future types include "cron"
// and "webhook".
type ScheduledJob struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"project_id"`
	Name         string     `json:"name"`
	JobType      string     `json:"job_type"`       // "loop"
	Agent        string     `json:"agent"`           // "claude" | "codex"
	Interval     string     `json:"interval"`        // e.g. "5m", "" = dynamic/self-paced
	Prompt       string     `json:"prompt"`          // slash command or plain prompt text
	TargetPaneID string     `json:"target_pane_id"`  // pane UUID to inject into
	Status       string     `json:"status"`          // "idle" | "running"
	LastRunAt    *time.Time `json:"last_run_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
