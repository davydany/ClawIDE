package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	s, err := New(path)
	require.NoError(t, err)
	return s
}

func TestNew(t *testing.T) {
	t.Run("creates from non-existent file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "state.json")
		s, err := New(path)
		require.NoError(t, err)
		assert.NotNil(t, s)
		assert.Empty(t, s.GetProjects())
	})

	t.Run("loads existing valid state", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "state.json")

		state := State{
			Projects: []model.Project{
				{ID: "p1", Name: "Project 1", Path: "/path1"},
			},
			Sessions: []model.Session{
				{
					ID:        "s1",
					ProjectID: "p1",
					Name:      "Session 1",
					Layout:    model.NewLeafPane("s1"),
				},
			},
		}
		data, err := json.MarshalIndent(state, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, data, 0644))

		s, err := New(path)
		require.NoError(t, err)
		assert.Len(t, s.GetProjects(), 1)
		assert.Len(t, s.GetAllSessions(), 1)
	})
}

func TestGetProjects(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s := newTestStore(t)
		assert.Empty(t, s.GetProjects())
	})

	t.Run("single", func(t *testing.T) {
		s := newTestStore(t)
		require.NoError(t, s.AddProject(model.Project{ID: "p1", Name: "Test"}))
		projects := s.GetProjects()
		assert.Len(t, projects, 1)
		assert.Equal(t, "p1", projects[0].ID)
	})

	t.Run("multiple", func(t *testing.T) {
		s := newTestStore(t)
		require.NoError(t, s.AddProject(model.Project{ID: "p1", Name: "First"}))
		require.NoError(t, s.AddProject(model.Project{ID: "p2", Name: "Second"}))
		assert.Len(t, s.GetProjects(), 2)
	})
}

func TestGetProject(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.AddProject(model.Project{ID: "p1", Name: "Test"}))

	t.Run("found", func(t *testing.T) {
		p, ok := s.GetProject("p1")
		assert.True(t, ok)
		assert.Equal(t, "Test", p.Name)
	})

	t.Run("not found", func(t *testing.T) {
		_, ok := s.GetProject("missing")
		assert.False(t, ok)
	})
}

func TestAddProject(t *testing.T) {
	s := newTestStore(t)
	p := model.Project{
		ID:        "p1",
		Name:      "Test Project",
		Path:      "/test/path",
		CreatedAt: time.Now(),
	}

	err := s.AddProject(p)
	require.NoError(t, err)

	// Verify persisted to disk
	data, err := os.ReadFile(s.filePath)
	require.NoError(t, err)

	var state State
	require.NoError(t, json.Unmarshal(data, &state))
	assert.Len(t, state.Projects, 1)
	assert.Equal(t, "p1", state.Projects[0].ID)
}

func TestUpdateProject(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.AddProject(model.Project{ID: "p1", Name: "Original"}))

	t.Run("updates existing", func(t *testing.T) {
		err := s.UpdateProject(model.Project{ID: "p1", Name: "Updated"})
		require.NoError(t, err)

		p, ok := s.GetProject("p1")
		assert.True(t, ok)
		assert.Equal(t, "Updated", p.Name)
	})

	t.Run("error for missing", func(t *testing.T) {
		err := s.UpdateProject(model.Project{ID: "missing", Name: "Nope"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestDeleteProject(t *testing.T) {
	t.Run("removes project and associated sessions", func(t *testing.T) {
		s := newTestStore(t)
		require.NoError(t, s.AddProject(model.Project{ID: "p1", Name: "Test"}))
		require.NoError(t, s.AddSession(model.Session{ID: "s1", ProjectID: "p1", Name: "Session 1"}))
		require.NoError(t, s.AddSession(model.Session{ID: "s2", ProjectID: "p1", Name: "Session 2"}))
		require.NoError(t, s.AddSession(model.Session{ID: "s3", ProjectID: "p2", Name: "Other"}))

		err := s.DeleteProject("p1")
		require.NoError(t, err)

		_, ok := s.GetProject("p1")
		assert.False(t, ok)

		// Sessions for p1 should be gone
		sessions := s.GetSessions("p1")
		assert.Empty(t, sessions)

		// Session for p2 should remain
		all := s.GetAllSessions()
		assert.Len(t, all, 1)
		assert.Equal(t, "s3", all[0].ID)
	})

	t.Run("error on missing", func(t *testing.T) {
		s := newTestStore(t)
		err := s.DeleteProject("missing")
		assert.Error(t, err)
	})
}

func TestSessionCRUD(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.AddProject(model.Project{ID: "p1", Name: "Test"}))

	// AddSession
	sess := model.Session{
		ID:        "s1",
		ProjectID: "p1",
		Name:      "Session 1",
		Layout:    model.NewLeafPane("s1"),
	}
	require.NoError(t, s.AddSession(sess))

	// GetSession
	got, ok := s.GetSession("s1")
	assert.True(t, ok)
	assert.Equal(t, "Session 1", got.Name)

	// GetSession not found
	_, ok = s.GetSession("missing")
	assert.False(t, ok)

	// GetSessions filtered by project
	sessions := s.GetSessions("p1")
	assert.Len(t, sessions, 1)

	sessions = s.GetSessions("other-project")
	assert.Empty(t, sessions)

	// GetAllSessions
	all := s.GetAllSessions()
	assert.Len(t, all, 1)

	// UpdateSession
	sess.Name = "Updated Session"
	require.NoError(t, s.UpdateSession(sess))
	got, _ = s.GetSession("s1")
	assert.Equal(t, "Updated Session", got.Name)

	// UpdateSession not found
	err := s.UpdateSession(model.Session{ID: "missing"})
	assert.Error(t, err)

	// DeleteSession
	require.NoError(t, s.DeleteSession("s1"))
	_, ok = s.GetSession("s1")
	assert.False(t, ok)

	// DeleteSession not found
	err = s.DeleteSession("missing")
	assert.Error(t, err)
}

func TestMigrationBackfillsLayout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Create state with a session that has no layout
	state := State{
		Sessions: []model.Session{
			{ID: "s1", ProjectID: "p1", Name: "No Layout", Layout: nil},
		},
	}
	data, err := json.MarshalIndent(state, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))

	s, err := New(path)
	require.NoError(t, err)

	sess, ok := s.GetSession("s1")
	require.True(t, ok)
	require.NotNil(t, sess.Layout)
	assert.Equal(t, "leaf", sess.Layout.Type)
	assert.Equal(t, "s1", sess.Layout.PaneID)
}

func TestConcurrentAccess(t *testing.T) {
	s := newTestStore(t)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(id string) {
			defer wg.Done()
			_ = s.AddProject(model.Project{ID: id, Name: id})
		}("p" + string(rune('A'+i)))

		go func() {
			defer wg.Done()
			_ = s.GetProjects()
		}()
	}
	wg.Wait()
	// If we get here without panics or race conditions, the test passes
}
