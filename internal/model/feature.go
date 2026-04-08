package model

import "time"

// Feature represents an isolated development workspace backed by a git
// worktree. Each feature owns a branch and a worktree directory where
// sessions run independently from the main project checkout.
type Feature struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Name         string    `json:"name"`
	BranchName   string    `json:"branch_name"`
	BaseBranch   string    `json:"base_branch"`
	WorktreePath string    `json:"worktree_path"`
	Color        string    `json:"color"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TrashedFeature holds a soft-deleted feature. The git branch is preserved
// so the feature can be restored by recreating its worktree. Items are
// automatically purged after 30 days.
type TrashedFeature struct {
	ID          string    `json:"id"`           // unique trash item ID
	Feature     Feature   `json:"feature"`      // snapshot of the original feature
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"` // snapshot — project may be deleted later
	ProjectPath string    `json:"project_path"` // needed for worktree recreation
	TrashedAt   time.Time `json:"trashed_at"`
}
