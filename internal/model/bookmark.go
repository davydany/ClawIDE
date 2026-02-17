package model

import "time"

type Bookmark struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	FolderID  string    `json:"folder_id,omitempty"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Emoji     string    `json:"emoji,omitempty"`
	InBar     bool      `json:"in_bar,omitempty"`
	Order     int       `json:"order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Starred is kept for backward compatibility during migration / JSON deserialization
	// from the old global store. New code should use InBar instead.
	Starred bool `json:"starred,omitempty"`
}
