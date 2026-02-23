package migration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davydany/ClawIDE/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateNotesForProject(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "notes.json")
	notesDir := filepath.Join(dir, "project", "notes")

	now := time.Now()
	notes := []oldNote{
		{ID: "n1", ProjectID: "p1", Title: "P1 Note", Content: "Content 1", CreatedAt: now, UpdatedAt: now},
		{ID: "n2", ProjectID: "p2", Title: "P2 Note", Content: "Content 2", CreatedAt: now, UpdatedAt: now},
		{ID: "n3", ProjectID: "p1", Title: "Another P1", Content: "Content 3", CreatedAt: now, UpdatedAt: now},
	}
	data, _ := json.MarshalIndent(notes, "", "  ")
	require.NoError(t, os.WriteFile(globalPath, data, 0644))

	noteStore, err := store.NewProjectNoteStore(notesDir)
	require.NoError(t, err)

	require.NoError(t, MigrateNotesForProject(globalPath, "p1", noteStore))

	all := noteStore.GetAll()
	assert.Len(t, all, 2)

	got, ok := noteStore.Get("n1")
	require.True(t, ok)
	assert.Equal(t, "P1 Note", got.Title)
	assert.Equal(t, "Content 1", got.Content)

	// p2 notes should not be migrated
	_, ok = noteStore.Get("n2")
	assert.False(t, ok)
}

func TestMigrateBookmarksForProject(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "bookmarks.json")
	bmDir := filepath.Join(dir, "project", "bookmarks")

	now := time.Now()
	bookmarks := []oldBookmark{
		{ID: "b1", ProjectID: "p1", Name: "Google", URL: "https://google.com", Starred: true, CreatedAt: now, UpdatedAt: now},
		{ID: "b2", ProjectID: "p2", Name: "GitHub", URL: "https://github.com", Starred: false, CreatedAt: now, UpdatedAt: now},
		{ID: "b3", ProjectID: "p1", Name: "Docs", URL: "https://docs.com", Starred: false, CreatedAt: now, UpdatedAt: now},
	}
	data, _ := json.MarshalIndent(bookmarks, "", "  ")
	require.NoError(t, os.WriteFile(globalPath, data, 0644))

	bmStore, err := store.NewProjectBookmarkStore(bmDir)
	require.NoError(t, err)

	require.NoError(t, MigrateBookmarksForProject(globalPath, "p1", bmStore))

	all := bmStore.GetAll()
	assert.Len(t, all, 2)

	got, ok := bmStore.Get("b1")
	require.True(t, ok)
	assert.Equal(t, "Google", got.Name)
	assert.True(t, got.InBar) // starred → in_bar

	got, ok = bmStore.Get("b3")
	require.True(t, ok)
	assert.False(t, got.InBar) // was not starred
}

func TestMigrateNotesForProject_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	notesDir := filepath.Join(dir, "notes")
	noteStore, err := store.NewProjectNoteStore(notesDir)
	require.NoError(t, err)

	// Should not error on nonexistent file
	err = MigrateNotesForProject("/nonexistent/notes.json", "p1", noteStore)
	assert.NoError(t, err)
}

func TestMigrateBookmarksForProject_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	bmDir := filepath.Join(dir, "bookmarks")
	bmStore, err := store.NewProjectBookmarkStore(bmDir)
	require.NoError(t, err)

	err = MigrateBookmarksForProject("/nonexistent/bookmarks.json", "p1", bmStore)
	assert.NoError(t, err)
}

func TestMigrateNotesForProject_Idempotent(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "notes.json")
	notesDir := filepath.Join(dir, "project", "notes")

	now := time.Now()
	notes := []oldNote{
		{ID: "n1", ProjectID: "p1", Title: "Note", Content: "Content", CreatedAt: now, UpdatedAt: now},
	}
	data, _ := json.MarshalIndent(notes, "", "  ")
	require.NoError(t, os.WriteFile(globalPath, data, 0644))

	noteStore, err := store.NewProjectNoteStore(notesDir)
	require.NoError(t, err)

	// Run twice
	require.NoError(t, MigrateNotesForProject(globalPath, "p1", noteStore))
	require.NoError(t, MigrateNotesForProject(globalPath, "p1", noteStore))

	all := noteStore.GetAll()
	assert.Len(t, all, 1, "should not duplicate on second migration")
}

func TestMarkGlobalFilesMigrated(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "notes.json")
	bookmarksPath := filepath.Join(dir, "bookmarks.json")

	require.NoError(t, os.WriteFile(notesPath, []byte("[]"), 0644))
	require.NoError(t, os.WriteFile(bookmarksPath, []byte("[]"), 0644))

	MarkGlobalFilesMigrated(notesPath, bookmarksPath)

	// Original files should be gone
	_, err := os.Stat(notesPath)
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(bookmarksPath)
	assert.True(t, os.IsNotExist(err))

	// .migrated files should exist
	_, err = os.Stat(notesPath + ".migrated")
	assert.NoError(t, err)

	_, err = os.Stat(bookmarksPath + ".migrated")
	assert.NoError(t, err)
}
