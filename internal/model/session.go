package model

import "time"

type Session struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	Branch    string    `json:"branch"`
	WorkDir   string    `json:"work_dir"`
	CreatedAt time.Time `json:"created_at"`
}
