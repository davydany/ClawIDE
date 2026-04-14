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

// TrashedProject holds a soft-deleted project. The project directory has been
// moved into the ClawIDE trash folder on disk; OriginalPath records where to
// put it back on restore. Entries are auto-purged after 30 days.
type TrashedProject struct {
	ID           string    `json:"id"`
	Project      Project   `json:"project"`
	OriginalPath string    `json:"original_path"`
	TrashedPath  string    `json:"trashed_path"`
	TrashedAt    time.Time `json:"trashed_at"`
}
