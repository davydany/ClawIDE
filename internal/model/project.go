package model

import (
	"path/filepath"
	"time"
)

// TaskStorageMode controls where a project's tasks.md file is stored.
type TaskStorageMode string

const (
	// TaskStorageInProject stores tasks.md at <project>/.clawide/tasks.md (tracked in git).
	// This is the default when the field is empty.
	TaskStorageInProject TaskStorageMode = ""

	// TaskStorageGlobal stores tasks.md at ~/.clawide/projects/<project-slug>/tasks.md
	// (NOT tracked in the project's git repo).
	TaskStorageGlobal TaskStorageMode = "global"
)

type Project struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Path            string          `json:"path"`
	Starred         bool            `json:"starred"`
	Color           string          `json:"color"`
	ActiveBranch    string          `json:"active_branch,omitempty"`
	SortOrder       int             `json:"sort_order"`
	TaskStorage     TaskStorageMode `json:"task_storage,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// TaskStorageDir returns the directory to pass to NewProjectTaskStore based on the project's
// storage mode. globalDataDir is typically ~/.clawide (from config.DataDir).
func (p Project) TaskStorageDir(globalDataDir string) string {
	if p.TaskStorage == TaskStorageGlobal {
		return filepath.Join(globalDataDir, "projects", p.ID)
	}
	// Default: in-project
	return p.Path
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
