package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/sse"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHandler(t *testing.T) (*Handlers, *store.Store, *store.NotificationStore) {
	t.Helper()
	dir := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.DataDir = dir

	st, err := store.New(filepath.Join(dir, "state.json"))
	require.NoError(t, err)

	notifStore, err := store.NewNotificationStore(filepath.Join(dir, "notifications.json"), 200)
	require.NoError(t, err)

	hub := sse.NewHub()

	h := &Handlers{
		cfg:               cfg,
		store:             st,
		notificationStore: notifStore,
		sseHub:            hub,
	}

	return h, st, notifStore
}

func TestCreateNotification(t *testing.T) {
	h, _, _ := setupTestHandler(t)

	body := `{"title":"Test Notification","source":"test","level":"info"}`
	req := httptest.NewRequest("POST", "/api/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateNotification(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var n model.Notification
	err := json.Unmarshal(w.Body.Bytes(), &n)
	require.NoError(t, err)
	assert.Equal(t, "Test Notification", n.Title)
	assert.Equal(t, "test", n.Source)
	assert.Equal(t, "info", n.Level)
	assert.NotEmpty(t, n.ID)
}

func TestCreateNotification_MissingTitle(t *testing.T) {
	h, _, _ := setupTestHandler(t)

	body := `{"source":"test"}`
	req := httptest.NewRequest("POST", "/api/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateNotification(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateNotification_DefaultSourceLevel(t *testing.T) {
	h, _, _ := setupTestHandler(t)

	body := `{"title":"Minimal"}`
	req := httptest.NewRequest("POST", "/api/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateNotification(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var n model.Notification
	json.Unmarshal(w.Body.Bytes(), &n)
	assert.Equal(t, "system", n.Source)
	assert.Equal(t, "info", n.Level)
}

func TestListNotifications(t *testing.T) {
	h, _, notifStore := setupTestHandler(t)

	notifStore.Add(model.Notification{ID: "n1", Title: "First", Source: "test", Level: "info"})
	notifStore.Add(model.Notification{ID: "n2", Title: "Second", Source: "test", Level: "info"})

	req := httptest.NewRequest("GET", "/api/notifications", nil)
	w := httptest.NewRecorder()

	h.ListNotifications(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var notifications []model.Notification
	json.Unmarshal(w.Body.Bytes(), &notifications)
	assert.Len(t, notifications, 2)
}

func TestListNotifications_UnreadOnly(t *testing.T) {
	h, _, notifStore := setupTestHandler(t)

	notifStore.Add(model.Notification{ID: "n1", Title: "First", Source: "test", Level: "info"})
	notifStore.Add(model.Notification{ID: "n2", Title: "Second", Source: "test", Level: "info"})
	notifStore.MarkRead("n1")

	req := httptest.NewRequest("GET", "/api/notifications?unread_only=true", nil)
	w := httptest.NewRecorder()

	h.ListNotifications(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var notifications []model.Notification
	json.Unmarshal(w.Body.Bytes(), &notifications)
	assert.Len(t, notifications, 1)
	assert.Equal(t, "n2", notifications[0].ID)
}

func TestUnreadNotificationCount(t *testing.T) {
	h, _, notifStore := setupTestHandler(t)

	notifStore.Add(model.Notification{ID: "n1", Title: "First", Source: "test", Level: "info"})
	notifStore.Add(model.Notification{ID: "n2", Title: "Second", Source: "test", Level: "info"})

	req := httptest.NewRequest("GET", "/api/notifications/unread-count", nil)
	w := httptest.NewRecorder()

	h.UnreadNotificationCount(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]int
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 2, resp["count"])
}

func TestMarkNotificationRead(t *testing.T) {
	h, _, notifStore := setupTestHandler(t)

	notifStore.Add(model.Notification{ID: "n1", Title: "First", Source: "test", Level: "info"})

	// Create chi context for URL params
	r := chi.NewRouter()
	r.Patch("/api/notifications/{notifID}/read", h.MarkNotificationRead)

	req := httptest.NewRequest("PATCH", "/api/notifications/n1/read", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, 0, notifStore.UnreadCount())
}

func TestMarkAllNotificationsRead(t *testing.T) {
	h, _, notifStore := setupTestHandler(t)

	notifStore.Add(model.Notification{ID: "n1", Title: "First", Source: "test", Level: "info"})
	notifStore.Add(model.Notification{ID: "n2", Title: "Second", Source: "test", Level: "info"})

	req := httptest.NewRequest("POST", "/api/notifications/read-all", nil)
	w := httptest.NewRecorder()

	h.MarkAllNotificationsRead(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, 0, notifStore.UnreadCount())
}

func TestDeleteNotification(t *testing.T) {
	h, _, notifStore := setupTestHandler(t)

	notifStore.Add(model.Notification{ID: "n1", Title: "First", Source: "test", Level: "info"})

	r := chi.NewRouter()
	r.Delete("/api/notifications/{notifID}", h.DeleteNotification)

	req := httptest.NewRequest("DELETE", "/api/notifications/n1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Len(t, notifStore.GetAll(), 0)
}

func TestResolveProjectFromCWD(t *testing.T) {
	h, st, _ := setupTestHandler(t)

	// Add a project
	st.AddProject(model.Project{
		ID:   "proj-1",
		Name: "My Project",
		Path: "/Users/test/projects/myproject",
	})

	// Add a feature with worktree
	st.AddFeature(model.Feature{
		ID:           "feat-1",
		ProjectID:    "proj-1",
		Name:         "feature-x",
		WorktreePath: "/Users/test/projects/myproject-worktrees/feature-x",
	})

	// Test feature worktree resolution (more specific)
	pid, fid := h.resolveProjectFromCWD("/Users/test/projects/myproject-worktrees/feature-x/src")
	assert.Equal(t, "proj-1", pid)
	assert.Equal(t, "feat-1", fid)

	// Test project root resolution
	pid, fid = h.resolveProjectFromCWD("/Users/test/projects/myproject/src/main.go")
	assert.Equal(t, "proj-1", pid)
	assert.Empty(t, fid)

	// Test no match
	pid, fid = h.resolveProjectFromCWD("/Users/test/other/project")
	assert.Empty(t, pid)
	assert.Empty(t, fid)
}

func TestCreateNotification_CWDResolution(t *testing.T) {
	h, st, _ := setupTestHandler(t)

	st.AddProject(model.Project{
		ID:   "proj-1",
		Name: "My Project",
		Path: "/Users/test/projects/myproject",
	})

	body := `{"title":"Claude done","source":"claude","level":"success","cwd":"/Users/test/projects/myproject"}`
	req := httptest.NewRequest("POST", "/api/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateNotification(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var n model.Notification
	json.Unmarshal(w.Body.Bytes(), &n)
	assert.Equal(t, "proj-1", n.ProjectID)
}
