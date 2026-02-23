package migration

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/store"
)

// oldNote matches the structure stored in the legacy global notes.json.
type oldNote struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// oldBookmark matches the structure stored in the legacy global bookmarks.json.
type oldBookmark struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Emoji     string    `json:"emoji,omitempty"`
	Starred   bool      `json:"starred"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MigrateNotesForProject reads notes from the global notes.json that belong
// to projectID and migrates them into the project-specific note store.
// The global file is renamed to notes.json.migrated after successful migration.
func MigrateNotesForProject(globalNotesPath string, projectID string, projectStore *store.ProjectNoteStore) error {
	data, err := os.ReadFile(globalNotesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to migrate
		}
		return fmt.Errorf("reading global notes: %w", err)
	}

	var notes []oldNote
	if err := json.Unmarshal(data, &notes); err != nil {
		return fmt.Errorf("parsing global notes: %w", err)
	}

	migrated := 0
	for i, n := range notes {
		if n.ProjectID != projectID {
			continue
		}

		// Check if already migrated (note with same ID exists)
		if _, exists := projectStore.Get(n.ID); exists {
			continue
		}

		newNote := model.Note{
			ID:        n.ID,
			ProjectID: n.ProjectID,
			Title:     n.Title,
			Content:   n.Content,
			Order:     i,
			CreatedAt: n.CreatedAt,
			UpdatedAt: n.UpdatedAt,
		}
		if err := projectStore.Add(newNote); err != nil {
			return fmt.Errorf("migrating note %s: %w", n.ID, err)
		}
		migrated++
	}

	if migrated > 0 {
		log.Printf("Migrated %d notes for project %s", migrated, projectID)
	}
	return nil
}

// MigrateBookmarksForProject reads bookmarks from the global bookmarks.json that
// belong to projectID and migrates them into the project-specific bookmark store.
// Converts the Starred field to InBar.
func MigrateBookmarksForProject(globalBookmarksPath string, projectID string, projectStore *store.ProjectBookmarkStore) error {
	data, err := os.ReadFile(globalBookmarksPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to migrate
		}
		return fmt.Errorf("reading global bookmarks: %w", err)
	}

	var bookmarks []oldBookmark
	if err := json.Unmarshal(data, &bookmarks); err != nil {
		return fmt.Errorf("parsing global bookmarks: %w", err)
	}

	migrated := 0
	for i, b := range bookmarks {
		if b.ProjectID != projectID {
			continue
		}

		// Check if already migrated
		if _, exists := projectStore.Get(b.ID); exists {
			continue
		}

		newBookmark := model.Bookmark{
			ID:        b.ID,
			ProjectID: b.ProjectID,
			Name:      b.Name,
			URL:       b.URL,
			Emoji:     b.Emoji,
			InBar:     b.Starred, // starred → in_bar
			Order:     i,
			CreatedAt: b.CreatedAt,
			UpdatedAt: b.UpdatedAt,
		}
		if err := projectStore.Add(newBookmark); err != nil {
			return fmt.Errorf("migrating bookmark %s: %w", b.ID, err)
		}
		migrated++
	}

	if migrated > 0 {
		log.Printf("Migrated %d bookmarks for project %s", migrated, projectID)
	}
	return nil
}

// MarkGlobalFilesMigrated renames the global JSON files to *.json.migrated
// if all items have been processed. Call this after all projects have been migrated.
func MarkGlobalFilesMigrated(globalNotesPath, globalBookmarksPath string) {
	for _, path := range []string{globalNotesPath, globalBookmarksPath} {
		if _, err := os.Stat(path); err == nil {
			dest := path + ".migrated"
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				if err := os.Rename(path, dest); err != nil {
					log.Printf("Warning: failed to rename %s to %s: %v", path, dest, err)
				} else {
					log.Printf("Migrated global file: %s → %s", filepath.Base(path), filepath.Base(dest))
				}
			}
		}
	}
}

// EnsureProjectStores creates and returns project-scoped stores for a given
// project path, running migration from global stores if needed.
func EnsureProjectStores(projectPath, globalNotesPath, globalBookmarksPath, projectID string) (*store.ProjectNoteStore, *store.ProjectBookmarkStore, error) {
	notesDir := filepath.Join(projectPath, ".clawide", "notes")
	bookmarksDir := filepath.Join(projectPath, ".clawide", "bookmarks")

	noteStore, err := store.NewProjectNoteStore(notesDir)
	if err != nil {
		return nil, nil, fmt.Errorf("initializing project note store: %w", err)
	}

	bookmarkStore, err := store.NewProjectBookmarkStore(bookmarksDir)
	if err != nil {
		return nil, nil, fmt.Errorf("initializing project bookmark store: %w", err)
	}

	// Migrate from global storage if global files exist
	if err := MigrateNotesForProject(globalNotesPath, projectID, noteStore); err != nil {
		log.Printf("Warning: note migration failed for project %s: %v", projectID, err)
	}
	if err := MigrateBookmarksForProject(globalBookmarksPath, projectID, bookmarkStore); err != nil {
		log.Printf("Warning: bookmark migration failed for project %s: %v", projectID, err)
	}

	return noteStore, bookmarkStore, nil
}
