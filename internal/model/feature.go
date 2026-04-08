package model

import "time"

const (
	// FeatureTypeFeature is the default workspace type backed by a git worktree.
	FeatureTypeFeature = "feature"
	// FeatureTypeBranch is a workspace type backed by a full git clone.
	FeatureTypeBranch = "branch"
)

// Feature represents an isolated development workspace backed by a git
// worktree or a full clone. Each feature owns a branch and a working
// directory where sessions run independently from the main project checkout.
type Feature struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Type         string    `json:"type"`
	Name         string    `json:"name"`
	BranchName   string    `json:"branch_name"`
	BaseBranch   string    `json:"base_branch"`
	WorktreePath string    `json:"worktree_path"`
	Color        string    `json:"color"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsClone returns true if the workspace is backed by a full git clone
// rather than a worktree.
func (f Feature) IsClone() bool {
	return f.Type == FeatureTypeBranch
}

// EffectiveType returns the workspace type, defaulting to "feature" for
// backward compatibility with records that have an empty Type field.
func (f Feature) EffectiveType() string {
	if f.Type == "" {
		return FeatureTypeFeature
	}
	return f.Type
}
