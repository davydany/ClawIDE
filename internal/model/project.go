package model

import "time"

type Project struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Starred      bool      `json:"starred"`
	Color        string    `json:"color"`
	ActiveBranch string    `json:"active_branch,omitempty"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
