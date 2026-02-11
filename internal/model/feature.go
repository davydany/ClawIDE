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
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
