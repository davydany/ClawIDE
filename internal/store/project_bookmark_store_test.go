package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestProjectBookmarkStore(t *testing.T) *ProjectBookmarkStore {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "bookmarks")
	s, err := NewProjectBookmarkStore(dir)
	require.NoError(t, err)
	return s
}

func TestProjectBookmarkStore_CRUD(t *testing.T) {
	s := newTestProjectBookmarkStore(t)

	now := time.Now()
	bm := model.Bookmark{
		ID:        "bm-1",
		ProjectID: "proj-1",
		Name:      "Google",
		URL:       "https://google.com",
		InBar:     true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Add
	require.NoError(t, s.Add(bm))

	// Get
	got, ok := s.Get("bm-1")
	require.True(t, ok)
	assert.Equal(t, "Google", got.Name)
	assert.True(t, got.InBar)

	// GetAll
	all := s.GetAll()
	assert.Len(t, all, 1)

	// GetInBar
	bar := s.GetInBar()
	assert.Len(t, bar, 1)

	// CountInBar
	assert.Equal(t, 1, s.CountInBar())

	// Update
	bm.Name = "Google Search"
	require.NoError(t, s.Update(bm))
	got, _ = s.Get("bm-1")
	assert.Equal(t, "Google Search", got.Name)

	// Delete
	require.NoError(t, s.Delete("bm-1"))
	_, ok = s.Get("bm-1")
	assert.False(t, ok)
}

func TestProjectBookmarkStore_FolderCRUD(t *testing.T) {
	s := newTestProjectBookmarkStore(t)

	now := time.Now()
	folder := model.Folder{
		ID:        "folder-1",
		Name:      "Work",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create
	require.NoError(t, s.CreateFolder(folder))
	assert.Len(t, s.GetFolders(), 1)

	// Get
	got, ok := s.GetFolder("folder-1")
	require.True(t, ok)
	assert.Equal(t, "Work", got.Name)

	// Update
	folder.Name = "Work Links"
	require.NoError(t, s.UpdateFolder(folder))
	got, _ = s.GetFolder("folder-1")
	assert.Equal(t, "Work Links", got.Name)

	// Delete
	require.NoError(t, s.DeleteFolder("folder-1"))
	assert.Empty(t, s.GetFolders())
}

func TestProjectBookmarkStore_GetByFolder(t *testing.T) {
	s := newTestProjectBookmarkStore(t)
	now := time.Now()

	require.NoError(t, s.CreateFolder(model.Folder{ID: "f1", Name: "F1", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Add(model.Bookmark{ID: "b1", Name: "In folder", URL: "https://a.com", FolderID: "f1", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Add(model.Bookmark{ID: "b2", Name: "No folder", URL: "https://b.com", CreatedAt: now, UpdatedAt: now}))

	inFolder := s.GetByFolder("f1")
	assert.Len(t, inFolder, 1)
	assert.Equal(t, "b1", inFolder[0].ID)

	root := s.GetByFolder("")
	assert.Len(t, root, 1)
	assert.Equal(t, "b2", root[0].ID)
}

func TestProjectBookmarkStore_Search(t *testing.T) {
	s := newTestProjectBookmarkStore(t)
	now := time.Now()

	require.NoError(t, s.Add(model.Bookmark{ID: "b1", Name: "Google", URL: "https://google.com", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Add(model.Bookmark{ID: "b2", Name: "GitHub", URL: "https://github.com", CreatedAt: now, UpdatedAt: now}))

	results := s.Search("google")
	assert.Len(t, results, 1)
	assert.Equal(t, "b1", results[0].ID)

	results = s.Search("github")
	assert.Len(t, results, 1)
	assert.Equal(t, "b2", results[0].ID)
}

func TestProjectBookmarkStore_FolderDepthValidation(t *testing.T) {
	s := newTestProjectBookmarkStore(t)
	now := time.Now()

	for i := 1; i <= 5; i++ {
		parentID := ""
		if i > 1 {
			parentID = "f" + string(rune('0'+i-1))
		}
		require.NoError(t, s.CreateFolder(model.Folder{
			ID:        "f" + string(rune('0'+i)),
			Name:      "Level",
			ParentID:  parentID,
			CreatedAt: now,
			UpdatedAt: now,
		}))
	}

	err := s.CreateFolder(model.Folder{
		ID:        "f6",
		Name:      "Level 6",
		ParentID:  "f5",
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum depth")
}

func TestProjectBookmarkStore_DeleteFolder_MovesBookmarks(t *testing.T) {
	s := newTestProjectBookmarkStore(t)
	now := time.Now()

	require.NoError(t, s.CreateFolder(model.Folder{ID: "f1", Name: "Folder", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Add(model.Bookmark{ID: "b1", Name: "BM", URL: "https://x.com", FolderID: "f1", CreatedAt: now, UpdatedAt: now}))

	require.NoError(t, s.DeleteFolder("f1"))

	bm, ok := s.Get("b1")
	require.True(t, ok)
	assert.Empty(t, bm.FolderID)
}

func TestProjectBookmarkStore_Reorder(t *testing.T) {
	s := newTestProjectBookmarkStore(t)
	now := time.Now()

	require.NoError(t, s.Add(model.Bookmark{ID: "b1", Name: "A", URL: "https://a.com", Order: 0, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Add(model.Bookmark{ID: "b2", Name: "B", URL: "https://b.com", Order: 1, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Add(model.Bookmark{ID: "b3", Name: "C", URL: "https://c.com", Order: 2, CreatedAt: now, UpdatedAt: now}))

	require.NoError(t, s.Reorder([]string{"b3", "b1", "b2"}))

	all := s.GetAll()
	// After reorder: b3=0, b1=1, b2=2 — but sorted by InBar then Order
	assert.Equal(t, "b3", all[0].ID)
	assert.Equal(t, "b1", all[1].ID)
	assert.Equal(t, "b2", all[2].ID)
}

func TestProjectBookmarkStore_Persistence(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bookmarks")

	s1, err := NewProjectBookmarkStore(dir)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, s1.Add(model.Bookmark{
		ID:        "b1",
		ProjectID: "p1",
		Name:      "Persisted",
		URL:       "https://persisted.com",
		InBar:     true,
		CreatedAt: now,
		UpdatedAt: now,
	}))
	require.NoError(t, s1.CreateFolder(model.Folder{
		ID:        "f1",
		Name:      "Saved Folder",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	// Reload
	s2, err := NewProjectBookmarkStore(dir)
	require.NoError(t, err)

	got, ok := s2.Get("b1")
	require.True(t, ok)
	assert.Equal(t, "Persisted", got.Name)
	assert.True(t, got.InBar)

	folders := s2.GetFolders()
	assert.Len(t, folders, 1)
	assert.Equal(t, "Saved Folder", folders[0].Name)
}
