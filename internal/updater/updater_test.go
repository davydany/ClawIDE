package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/sse"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/davydany/ClawIDE/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeRelease(tag string) githubRelease {
	return githubRelease{
		TagName: tag,
		HTMLURL: "https://github.com/davydany/ClawIDE/releases/tag/" + tag,
		Assets: []githubAsset{
			{
				Name:               platformAssetName(tag),
				BrowserDownloadURL: "https://example.com/" + platformAssetName(tag),
				Size:               15000000,
			},
		},
	}
}

func setupTestUpdater(t *testing.T, handler http.Handler) (*Updater, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)

	dataDir := t.TempDir()
	cfg := &config.Config{
		DataDir:         dataDir,
		AutoUpdateCheck: true,
	}

	notifPath := filepath.Join(dataDir, "notifications.json")
	notifStore, err := store.NewNotificationStore(notifPath, 200)
	require.NoError(t, err)

	hub := sse.NewHub()
	u := NewWithBaseURL(cfg, notifStore, hub, ts.URL)
	return u, ts
}

func TestCheck_DevVersion(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "dev"

	u, ts := setupTestUpdater(t, http.NotFoundHandler())
	defer ts.Close()

	state := u.Check()
	assert.True(t, state.IsDev)
	assert.Equal(t, "dev", state.CurrentVersion)
	assert.False(t, state.UpdateAvailable)
}

func TestCheck_NoUpdate(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.0.0"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(fakeRelease("v1.0.0"))
	})

	u, ts := setupTestUpdater(t, handler)
	defer ts.Close()

	state := u.Check()
	assert.False(t, state.IsDev)
	assert.False(t, state.UpdateAvailable)
	assert.Equal(t, "v1.0.0", state.CurrentVersion)
	assert.Equal(t, "v1.0.0", state.LatestVersion)
}

func TestCheck_UpdateAvailable(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.0.0"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(fakeRelease("v1.1.0"))
	})

	u, ts := setupTestUpdater(t, handler)
	defer ts.Close()

	state := u.Check()
	assert.True(t, state.UpdateAvailable)
	assert.Equal(t, "v1.1.0", state.LatestVersion)
	assert.NotEmpty(t, state.AssetURL)
	assert.Equal(t, int64(15000000), state.AssetSize)
	assert.Empty(t, state.Error)
}

func TestCheck_RateLimit(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.0.0"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	u, ts := setupTestUpdater(t, handler)
	defer ts.Close()

	state := u.Check()
	assert.Contains(t, state.Error, "rate limit")
}

func TestCheck_NetworkError(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.0.0"

	dataDir := t.TempDir()
	cfg := &config.Config{
		DataDir:         dataDir,
		AutoUpdateCheck: true,
	}

	notifPath := filepath.Join(dataDir, "notifications.json")
	notifStore, err := store.NewNotificationStore(notifPath, 200)
	require.NoError(t, err)

	u := NewWithBaseURL(cfg, notifStore, sse.NewHub(), "http://127.0.0.1:1") // dead port
	state := u.Check()
	assert.Contains(t, state.Error, "network error")
}

func TestCheck_NoPlatformAsset(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.0.0"

	// Release with no matching asset
	release := githubRelease{
		TagName: "v2.0.0",
		HTMLURL: "https://github.com/davydany/ClawIDE/releases/tag/v2.0.0",
		Assets: []githubAsset{
			{
				Name:               "clawide-v2.0.0-windows-amd64.tar.gz",
				BrowserDownloadURL: "https://example.com/clawide-v2.0.0-windows-amd64.tar.gz",
				Size:               10000000,
			},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(release)
	})

	u, ts := setupTestUpdater(t, handler)
	defer ts.Close()

	state := u.Check()
	assert.True(t, state.UpdateAvailable)
	assert.Contains(t, state.Error, "no build for")
}

func TestCheck_Disabled(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.0.0"

	u, ts := setupTestUpdater(t, http.NotFoundHandler())
	defer ts.Close()
	u.cfg.AutoUpdateCheck = false

	state := u.Check()
	assert.Contains(t, state.Error, "disabled")
}

func TestStatePersistence(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.0.0"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(fakeRelease("v1.1.0"))
	})

	u, ts := setupTestUpdater(t, handler)
	defer ts.Close()

	u.Check()

	// Verify state file was written
	data, err := os.ReadFile(u.cfg.UpdateStatePath())
	require.NoError(t, err)

	var persisted State
	require.NoError(t, json.Unmarshal(data, &persisted))
	assert.True(t, persisted.UpdateAvailable)
	assert.Equal(t, "v1.1.0", persisted.LatestVersion)
}

func TestStateLoadOnInit(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{
		DataDir:         dataDir,
		AutoUpdateCheck: true,
	}

	// Write a state file before creating the updater
	state := State{
		CurrentVersion:  "v1.0.0",
		UpdateAvailable: true,
		LatestVersion:   "v1.2.0",
	}
	data, _ := json.Marshal(state)
	os.WriteFile(cfg.UpdateStatePath(), data, 0644)

	notifPath := filepath.Join(dataDir, "notifications.json")
	notifStore, err := store.NewNotificationStore(notifPath, 200)
	require.NoError(t, err)

	u := NewWithBaseURL(cfg, notifStore, sse.NewHub(), "http://unused")

	got := u.State()
	assert.True(t, got.UpdateAvailable)
	assert.Equal(t, "v1.2.0", got.LatestVersion)
}

func TestNotificationSent(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.0.0"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(fakeRelease("v1.1.0"))
	})

	u, ts := setupTestUpdater(t, handler)
	defer ts.Close()

	u.Check()

	notifs := u.notificationStore.GetAll()
	require.Len(t, notifs, 1)
	assert.Contains(t, notifs[0].Title, "Update Available")
	assert.Contains(t, notifs[0].Body, "v1.1.0")
}

func TestState_IsDockerField(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.0.0"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(fakeRelease("v1.0.0"))
	})

	u, ts := setupTestUpdater(t, handler)
	defer ts.Close()

	state := u.State()
	// IsDocker should match the standalone function
	assert.Equal(t, IsDocker(), state.IsDocker)
}

func TestNotificationIdempotent(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.0.0"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(fakeRelease("v1.1.0"))
	})

	u, ts := setupTestUpdater(t, handler)
	defer ts.Close()

	u.Check()
	u.Check() // second check should not duplicate notification

	notifs := u.notificationStore.GetAll()
	assert.Len(t, notifs, 1)
}
