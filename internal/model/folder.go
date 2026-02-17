package model

import (
	"fmt"
	"time"
)

const MaxFolderDepth = 5

// Folder represents a generic folder for organizing notes or bookmarks.
type Folder struct {
	ID        string    `json:"id" yaml:"id"`
	Name      string    `json:"name" yaml:"name"`
	ParentID  string    `json:"parent_id,omitempty" yaml:"parent_id,omitempty"`
	Order     int       `json:"order" yaml:"order"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// ValidateFolderDepth checks that adding a child under parentID would not
// exceed MaxFolderDepth levels. folders is the full list of existing folders.
// A root-level folder (parentID == "") counts as depth 1.
func ValidateFolderDepth(parentID string, folders []Folder) error {
	if parentID == "" {
		return nil // root level = depth 1, always valid
	}

	depth := 1 // the new folder itself
	current := parentID
	seen := map[string]bool{}

	for current != "" {
		if seen[current] {
			return fmt.Errorf("circular folder reference detected at %s", current)
		}
		seen[current] = true
		depth++

		found := false
		for _, f := range folders {
			if f.ID == current {
				current = f.ParentID
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("parent folder %s not found", current)
		}
	}

	if depth > MaxFolderDepth {
		return fmt.Errorf("folder nesting exceeds maximum depth of %d (would be %d)", MaxFolderDepth, depth)
	}
	return nil
}
