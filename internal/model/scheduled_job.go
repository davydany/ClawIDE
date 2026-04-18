package model

import "time"

// ScheduledJob represents a managed scheduled job attached to a project.
// Supported job types:
//   - "loop":  sends /loop <interval> <prompt> to a tmux pane (in-session)
//   - "cron":  installs a system crontab entry that runs the agent CLI headlessly
type ScheduledJob struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"project_id"`
	Name           string     `json:"name"`
	JobType        string     `json:"job_type"`                  // "loop" | "cron"
	Agent          string     `json:"agent"`                     // "claude" | "codex" | "gemini"
	Interval       string     `json:"interval"`                  // loop: e.g. "5m", "" = dynamic/self-paced
	CronExpression string     `json:"cron_expression,omitempty"` // cron: 5-field cron expression
	Prompt         string     `json:"prompt"`                    // slash command or plain prompt text
	TargetPaneID   string     `json:"target_pane_id,omitempty"`  // loop: pane UUID to inject into
	Status         string     `json:"status"`                    // "idle" | "running"
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
