package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/sse"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/davydany/ClawIDE/internal/updater"
	"github.com/davydany/ClawIDE/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUpdateHandler(t *testing.T, githubHandler http.Handler) *Handlers {
	t.Helper()

	dataDir := t.TempDir()
	cfg := &config.Config{
		DataDir:         dataDir,
		AutoUpdateCheck: true,
	}

	notifPath := filepath.Join(dataDir, "notifications.json")
	notifStore, err := store.NewNotificationStore(notifPath, 200)
	require.NoError(t, err)

	hub := sse.NewHub()

	var ts *httptest.Server
	if githubHandler != nil {
		ts = httptest.NewServer(githubHandler)
		t.Cleanup(ts.Close)
	}

	baseURL := "http://127.0.0.1:1"
	if ts != nil {
		baseURL = ts.URL
	}

	upd := updater.NewWithBaseURL(cfg, notifStore, hub, baseURL)

	return &Handlers{
		cfg:     cfg,
		updater: upd,
		sseHub:  hub,
	}
}

func TestCheckForUpdate_DevMode(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "dev"

	h := setupUpdateHandler(t, nil)

	req := httptest.NewRequest("GET", "/api/update/check", nil)
	w := httptest.NewRecorder()
	h.CheckForUpdate(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var state updater.State
	require.NoError(t, json.NewDecoder(w.Body).Decode(&state))
	assert.True(t, state.IsDev)
	assert.False(t, state.UpdateAvailable)
}

func TestUpdateStatus_Cached(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.0.0"

	h := setupUpdateHandler(t, nil)

	req := httptest.NewRequest("GET", "/api/update/status", nil)
	w := httptest.NewRecorder()
	h.UpdateStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var state updater.State
	require.NoError(t, json.NewDecoder(w.Body).Decode(&state))
	assert.Equal(t, "v1.0.0", state.CurrentVersion)
}

func TestCheckForUpdate_NilUpdater(t *testing.T) {
	h := &Handlers{}

	req := httptest.NewRequest("GET", "/api/update/check", nil)
	w := httptest.NewRecorder()
	h.CheckForUpdate(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUpdateStatus_NilUpdater(t *testing.T) {
	h := &Handlers{}

	req := httptest.NewRequest("GET", "/api/update/status", nil)
	w := httptest.NewRecorder()
	h.UpdateStatus(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestApplyUpdate_NilUpdater(t *testing.T) {
	h := &Handlers{}

	req := httptest.NewRequest("POST", "/api/update/apply", nil)
	w := httptest.NewRecorder()
	h.ApplyUpdate(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestApplyUpdate_NoUpdateAvailable(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.0.0"

	h := setupUpdateHandler(t, nil)

	req := httptest.NewRequest("POST", "/api/update/apply", nil)
	w := httptest.NewRecorder()
	h.ApplyUpdate(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "error", resp["status"])
}
