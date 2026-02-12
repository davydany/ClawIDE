package model

import "time"

type Notification struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Body           string    `json:"body,omitempty"`
	Source         string    `json:"source"`
	Level          string    `json:"level"`
	ProjectID      string    `json:"project_id,omitempty"`
	SessionID      string    `json:"session_id,omitempty"`
	FeatureID      string    `json:"feature_id,omitempty"`
	CWD            string    `json:"cwd,omitempty"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
	Read           bool      `json:"read"`
	CreatedAt      time.Time `json:"created_at"`
}
