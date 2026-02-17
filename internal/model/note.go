package model

import (
	"fmt"
	"regexp"
	"time"
)

type Note struct {
	ID        string    `json:"id" yaml:"id"`
	ProjectID string    `json:"project_id" yaml:"project_id"`
	FolderID  string    `json:"folder_id,omitempty" yaml:"folder_id,omitempty"`
	Title     string    `json:"title" yaml:"title"`
	Content   string    `json:"content" yaml:"-"` // stored as markdown body, not in frontmatter
	Order     int       `json:"order" yaml:"order"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

var validTitleRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// ValidateNoteTitle checks that a note title is valid for use as a filename.
func ValidateNoteTitle(title string) error {
	if title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if title == "." || title == ".." {
		return fmt.Errorf("title cannot be '.' or '..'")
	}
	if len(title) > 255 {
		return fmt.Errorf("title exceeds maximum length of 255 characters")
	}
	if !validTitleRegex.MatchString(title) {
		return fmt.Errorf("title contains invalid characters; only letters, numbers, dots, hyphens, and underscores are allowed")
	}
	return nil
}

// ValidateFolderName checks that a folder name is valid for use as a directory name.
func ValidateFolderName(name string) error {
	if name == "" {
		return fmt.Errorf("folder name cannot be empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("folder name cannot be '.' or '..'")
	}
	if len(name) > 255 {
		return fmt.Errorf("folder name exceeds maximum length of 255 characters")
	}
	if !validTitleRegex.MatchString(name) {
		return fmt.Errorf("folder name contains invalid characters; only letters, numbers, dots, hyphens, and underscores are allowed")
	}
	return nil
}
