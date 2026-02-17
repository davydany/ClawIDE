package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestProjectNoteStore(t *testing.T) *ProjectNoteStore {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "notes")
	s, err := NewProjectNoteStore(dir)
	require.NoError(t, err)
	return s
}

func TestProjectNoteStore_CRUD(t *testing.T) {
	s := newTestProjectNoteStore(t)

	now := time.Now()
	note := model.Note{
		ID:        "note-1",
		ProjectID: "proj-1",
		Title:     "test-note",
		Content:   "# Hello\n\nThis is test content.",
		Order:     0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Add
	require.NoError(t, s.Add(note))

	// Get
	got, ok := s.Get("note-1")
	require.True(t, ok)
	assert.Equal(t, "test-note", got.Title)
	assert.Equal(t, "# Hello\n\nThis is test content.", got.Content)

	// GetAll
	all := s.GetAll()
	assert.Len(t, all, 1)

	// Update
	note.Title = "updated-title"
	note.Content = "Updated content"
	require.NoError(t, s.Update(note))

	got, _ = s.Get("note-1")
	assert.Equal(t, "updated-title", got.Title)
	assert.Equal(t, "Updated content", got.Content)

	// Delete
	require.NoError(t, s.Delete("note-1"))
	_, ok = s.Get("note-1")
	assert.False(t, ok)
}

func TestProjectNoteStore_FolderCRUD(t *testing.T) {
	s := newTestProjectNoteStore(t)

	now := time.Now()
	folder := model.Folder{
		ID:        "folder-1",
		Name:      "test-folder",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create
	require.NoError(t, s.CreateFolder(folder))
	folders := s.GetFolders()
	assert.Len(t, folders, 1)

	// Get
	got, ok := s.GetFolder("folder-1")
	require.True(t, ok)
	assert.Equal(t, "test-folder", got.Name)

	// Update
	folder.Name = "renamed-folder"
	require.NoError(t, s.UpdateFolder(folder))
	got, _ = s.GetFolder("folder-1")
	assert.Equal(t, "renamed-folder", got.Name)

	// Delete
	require.NoError(t, s.DeleteFolder("folder-1"))
	folders = s.GetFolders()
	assert.Empty(t, folders)
}

func TestProjectNoteStore_GetByFolder(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	require.NoError(t, s.CreateFolder(model.Folder{ID: "f1", Name: "f1", CreatedAt: now, UpdatedAt: now}))

	require.NoError(t, s.Add(model.Note{ID: "n1", Title: "in-folder", FolderID: "f1", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Add(model.Note{ID: "n2", Title: "no-folder", CreatedAt: now, UpdatedAt: now}))

	inFolder := s.GetByFolder("f1")
	assert.Len(t, inFolder, 1)
	assert.Equal(t, "n1", inFolder[0].ID)

	root := s.GetByFolder("")
	assert.Len(t, root, 1)
	assert.Equal(t, "n2", root[0].ID)
}

func TestProjectNoteStore_Search(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	require.NoError(t, s.Add(model.Note{ID: "n1", Title: "architecture-decision", Content: "We chose Go.", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Add(model.Note{ID: "n2", Title: "meeting-notes", Content: "Discussed frontend.", CreatedAt: now, UpdatedAt: now}))

	results := s.Search("architecture")
	assert.Len(t, results, 1)
	assert.Equal(t, "n1", results[0].ID)

	results = s.Search("frontend")
	assert.Len(t, results, 1)
	assert.Equal(t, "n2", results[0].ID)
}

func TestProjectNoteStore_FolderDepthValidation(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	// Create 5 nested folders (max depth)
	for i := 1; i <= 5; i++ {
		parentID := ""
		if i > 1 {
			parentID = "f" + string(rune('0'+i-1))
		}
		f := model.Folder{
			ID:        "f" + string(rune('0'+i)),
			Name:      "level-" + string(rune('0'+i)),
			ParentID:  parentID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, s.CreateFolder(f), "creating folder level %d", i)
	}

	// 6th level should fail
	err := s.CreateFolder(model.Folder{
		ID:        "f6",
		Name:      "level-6",
		ParentID:  "f5",
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum depth")
}

func TestProjectNoteStore_DeleteFolder_MovesNotes(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	require.NoError(t, s.CreateFolder(model.Folder{ID: "f1", Name: "folder-one", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Add(model.Note{ID: "n1", Title: "note-in-folder", FolderID: "f1", CreatedAt: now, UpdatedAt: now}))

	require.NoError(t, s.DeleteFolder("f1"))

	// Note should now be in root
	n, ok := s.Get("n1")
	require.True(t, ok)
	assert.Empty(t, n.FolderID)
}

func TestProjectNoteStore_Reorder(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	require.NoError(t, s.Add(model.Note{ID: "n1", Title: "first", Order: 0, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Add(model.Note{ID: "n2", Title: "second", Order: 1, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Add(model.Note{ID: "n3", Title: "third", Order: 2, CreatedAt: now, UpdatedAt: now}))

	// Reorder: n3, n1, n2
	require.NoError(t, s.Reorder([]string{"n3", "n1", "n2"}))

	all := s.GetAll()
	assert.Equal(t, "n3", all[0].ID) // order 0
	assert.Equal(t, "n1", all[1].ID) // order 1
	assert.Equal(t, "n2", all[2].ID) // order 2
}

func TestProjectNoteStore_Persistence(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "notes")

	// Create store and add a note
	s1, err := NewProjectNoteStore(dir)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, s1.Add(model.Note{
		ID:        "n1",
		ProjectID: "p1",
		Title:     "persisted-note",
		Content:   "# Content\n\nWith markdown.",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	// Verify <title>.md file exists (not <uuid>.md)
	_, err = os.Stat(filepath.Join(dir, "persisted-note.md"))
	require.NoError(t, err, "expected persisted-note.md to exist")

	// UUID-named file should NOT exist
	_, err = os.Stat(filepath.Join(dir, "n1.md"))
	assert.True(t, os.IsNotExist(err), "UUID-named file n1.md should not exist")

	// Reload store from disk
	s2, err := NewProjectNoteStore(dir)
	require.NoError(t, err)

	got, ok := s2.Get("n1")
	require.True(t, ok)
	assert.Equal(t, "persisted-note", got.Title)
	assert.Equal(t, "# Content\n\nWith markdown.", got.Content)
}

// --------------- New tests for directory-based storage ---------------

func TestProjectNoteStore_TitleAsFilename(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	require.NoError(t, s.Add(model.Note{
		ID:        "abc-123",
		Title:     "my-note",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	// File should be <title>.md, not <uuid>.md
	_, err := os.Stat(filepath.Join(s.baseDir, "my-note.md"))
	require.NoError(t, err, "expected my-note.md to exist")

	_, err = os.Stat(filepath.Join(s.baseDir, "abc-123.md"))
	assert.True(t, os.IsNotExist(err), "UUID-named file should not exist")
}

func TestProjectNoteStore_FolderDirectory(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	// Create a folder
	require.NoError(t, s.CreateFolder(model.Folder{
		ID:        "f1",
		Name:      "docs",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	// Verify directory was created
	info, err := os.Stat(filepath.Join(s.baseDir, "docs"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Create a note in the folder
	require.NoError(t, s.Add(model.Note{
		ID:        "n1",
		Title:     "readme",
		FolderID:  "f1",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	// Note file should be in the folder directory
	_, err = os.Stat(filepath.Join(s.baseDir, "docs", "readme.md"))
	require.NoError(t, err, "expected docs/readme.md to exist")
}

func TestProjectNoteStore_MoveNote(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	require.NoError(t, s.CreateFolder(model.Folder{ID: "f1", Name: "folder-a", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.CreateFolder(model.Folder{ID: "f2", Name: "folder-b", CreatedAt: now, UpdatedAt: now}))

	require.NoError(t, s.Add(model.Note{
		ID:        "n1",
		Title:     "movable-note",
		FolderID:  "f1",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	// Verify note is in folder-a
	_, err := os.Stat(filepath.Join(s.baseDir, "folder-a", "movable-note.md"))
	require.NoError(t, err)

	// Move note to folder-b
	note, _ := s.Get("n1")
	note.FolderID = "f2"
	note.UpdatedAt = time.Now()
	require.NoError(t, s.Update(note))

	// Old file should be gone
	_, err = os.Stat(filepath.Join(s.baseDir, "folder-a", "movable-note.md"))
	assert.True(t, os.IsNotExist(err), "old file should be removed")

	// New file should exist
	_, err = os.Stat(filepath.Join(s.baseDir, "folder-b", "movable-note.md"))
	require.NoError(t, err, "expected file in folder-b")
}

func TestProjectNoteStore_RenameNote(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	require.NoError(t, s.Add(model.Note{
		ID:        "n1",
		Title:     "old-name",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	_, err := os.Stat(filepath.Join(s.baseDir, "old-name.md"))
	require.NoError(t, err)

	// Rename the note
	note, _ := s.Get("n1")
	note.Title = "new-name"
	note.UpdatedAt = time.Now()
	require.NoError(t, s.Update(note))

	// Old file should be gone
	_, err = os.Stat(filepath.Join(s.baseDir, "old-name.md"))
	assert.True(t, os.IsNotExist(err), "old file should be removed after rename")

	// New file should exist
	_, err = os.Stat(filepath.Join(s.baseDir, "new-name.md"))
	require.NoError(t, err, "expected new-name.md to exist")
}

func TestProjectNoteStore_MoveFolder(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	require.NoError(t, s.CreateFolder(model.Folder{ID: "parent", Name: "parent-dir", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.CreateFolder(model.Folder{ID: "child", Name: "child-dir", CreatedAt: now, UpdatedAt: now}))

	// Add a note in the child folder
	require.NoError(t, s.Add(model.Note{
		ID:        "n1",
		Title:     "nested-note",
		FolderID:  "child",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	// Verify initial state
	_, err := os.Stat(filepath.Join(s.baseDir, "child-dir", "nested-note.md"))
	require.NoError(t, err)

	// Move child folder under parent
	folder, ok := s.GetFolder("child")
	require.True(t, ok)
	folder.ParentID = "parent"
	folder.UpdatedAt = time.Now()
	require.NoError(t, s.UpdateFolder(folder))

	// Old directory should be gone (or empty)
	_, err = os.Stat(filepath.Join(s.baseDir, "child-dir", "nested-note.md"))
	assert.True(t, os.IsNotExist(err), "old location should not have the note file")

	// New directory should exist under parent
	_, err = os.Stat(filepath.Join(s.baseDir, "parent-dir", "child-dir"))
	require.NoError(t, err, "child-dir should exist under parent-dir")

	// Note file should be in the moved directory
	_, err = os.Stat(filepath.Join(s.baseDir, "parent-dir", "child-dir", "nested-note.md"))
	require.NoError(t, err, "note should be in parent-dir/child-dir/")
}

func TestProjectNoteStore_RenameFolder(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	require.NoError(t, s.CreateFolder(model.Folder{ID: "f1", Name: "old-folder-name", CreatedAt: now, UpdatedAt: now}))

	// Add a note in the folder
	require.NoError(t, s.Add(model.Note{
		ID:        "n1",
		Title:     "some-note",
		FolderID:  "f1",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	// Verify initial state
	_, err := os.Stat(filepath.Join(s.baseDir, "old-folder-name", "some-note.md"))
	require.NoError(t, err)

	// Rename the folder
	folder, ok := s.GetFolder("f1")
	require.True(t, ok)
	folder.Name = "new-folder-name"
	folder.UpdatedAt = time.Now()
	require.NoError(t, s.UpdateFolder(folder))

	// Old directory should be gone
	_, err = os.Stat(filepath.Join(s.baseDir, "old-folder-name"))
	assert.True(t, os.IsNotExist(err), "old folder directory should be removed")

	// New directory should exist with the note inside
	_, err = os.Stat(filepath.Join(s.baseDir, "new-folder-name", "some-note.md"))
	require.NoError(t, err, "note should be in renamed folder directory")
}

func TestProjectNoteStore_DuplicateTitle(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	require.NoError(t, s.Add(model.Note{
		ID:        "n1",
		Title:     "duplicate-me",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	// Adding another note with the same title in the same folder should fail
	err := s.Add(model.Note{
		ID:        "n2",
		Title:     "duplicate-me",
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestProjectNoteStore_SameTitleDifferentFolders(t *testing.T) {
	s := newTestProjectNoteStore(t)
	now := time.Now()

	require.NoError(t, s.CreateFolder(model.Folder{ID: "f1", Name: "folder-one", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.CreateFolder(model.Folder{ID: "f2", Name: "folder-two", CreatedAt: now, UpdatedAt: now}))

	// Add a note in folder-one
	require.NoError(t, s.Add(model.Note{
		ID:        "n1",
		Title:     "same-title",
		FolderID:  "f1",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	// Add a note with the same title in folder-two -- should succeed
	require.NoError(t, s.Add(model.Note{
		ID:        "n2",
		Title:     "same-title",
		FolderID:  "f2",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	// Also add one in root -- should succeed
	require.NoError(t, s.Add(model.Note{
		ID:        "n3",
		Title:     "same-title",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	assert.Len(t, s.GetAll(), 3)
}

func TestProjectNoteStore_Migration(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "notes")
	require.NoError(t, os.MkdirAll(dir, 0755))

	// Create a UUID-named file manually (simulating old format)
	noteID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	frontmatter := "---\nid: " + noteID + "\ntitle: migrated-note\norder: 0\ncreated_at: 2024-01-01T00:00:00Z\nupdated_at: 2024-01-01T00:00:00Z\n---\n# Migrated content"
	require.NoError(t, os.WriteFile(filepath.Join(dir, noteID+".md"), []byte(frontmatter), 0644))

	// Init store -- should trigger migration
	s, err := NewProjectNoteStore(dir)
	require.NoError(t, err)

	// UUID-named file should be gone
	_, err = os.Stat(filepath.Join(dir, noteID+".md"))
	assert.True(t, os.IsNotExist(err), "UUID file should be renamed")

	// Title-named file should exist
	_, err = os.Stat(filepath.Join(dir, "migrated-note.md"))
	require.NoError(t, err, "expected migrated-note.md to exist")

	// Note should be loadable
	got, ok := s.Get(noteID)
	require.True(t, ok)
	assert.Equal(t, "migrated-note", got.Title)
}

func TestProjectNoteStore_InvalidTitle(t *testing.T) {
	tests := []struct {
		title string
		valid bool
	}{
		{"valid-title", true},
		{"valid.title", true},
		{"valid_title", true},
		{"ValidTitle123", true},
		{"title with spaces", false},
		{"title/slash", false},
		{"title@special!", false},
		{"..", false},
		{".", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			err := model.ValidateNoteTitle(tt.title)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
