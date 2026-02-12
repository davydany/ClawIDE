package model

import "time"

type Note struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"` // empty = global note
	Title     string    `json:"title"`
	Content   string    `json:"content"` // markdown
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
