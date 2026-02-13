package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/sse"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/davydany/ClawIDE/internal/version"
	"github.com/google/uuid"
)

const (
	defaultGitHubAPI = "https://api.github.com/repos/davydany/ClawIDE/releases/latest"
	checkInterval    = 24 * time.Hour
)

// State represents the cached update check result.
type State struct {
	IsDev           bool      `json:"is_dev"`
	IsDocker        bool      `json:"is_docker"`
	CurrentVersion  string    `json:"current_version"`
	UpdateAvailable bool      `json:"update_available"`
	LatestVersion   string    `json:"latest_version"`
	ReleaseURL      string    `json:"release_url"`
	AssetURL        string    `json:"asset_url,omitempty"`
	AssetSize       int64     `json:"asset_size"`
	LastCheck       time.Time `json:"last_check"`
	Error           string    `json:"error"`
}

// githubRelease models the relevant fields from the GitHub releases API.
type githubRelease struct {
	TagName string        `json:"tag_name"`
	HTMLURL string        `json:"html_url"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Updater manages checking for and applying updates.
type Updater struct {
	cfg               *config.Config
	notificationStore *store.NotificationStore
	sseHub            *sse.Hub
	client            *http.Client
	baseURL           string

	mu    sync.RWMutex
	state State

	stopCh chan struct{}
	done   chan struct{}
}

// New creates an Updater that checks the official GitHub releases API.
func New(cfg *config.Config, notifStore *store.NotificationStore, hub *sse.Hub) *Updater {
	return NewWithBaseURL(cfg, notifStore, hub, defaultGitHubAPI)
}

// NewWithBaseURL creates an Updater with a custom API URL (for testing).
func NewWithBaseURL(cfg *config.Config, notifStore *store.NotificationStore, hub *sse.Hub, baseURL string) *Updater {
	u := &Updater{
		cfg:               cfg,
		notificationStore: notifStore,
		sseHub:            hub,
		client:            &http.Client{Timeout: 15 * time.Second},
		baseURL:           baseURL,
		stopCh:            make(chan struct{}),
		done:              make(chan struct{}),
	}
	u.loadState()
	return u
}

// Start launches the background goroutine that checks every 24 hours.
func (u *Updater) Start() {
	go u.loop()
}

// Stop signals the background goroutine to exit and waits for it.
func (u *Updater) Stop() {
	close(u.stopCh)
	<-u.done
}

// Check performs a fresh check against the GitHub API and returns the state.
func (u *Updater) Check() State {
	if version.IsDevVersion() {
		u.mu.Lock()
		u.state = State{
			IsDev:          true,
			CurrentVersion: version.Version,
			LastCheck:      time.Now().UTC(),
		}
		u.mu.Unlock()
		return u.State()
	}

	if !u.cfg.AutoUpdateCheck {
		u.mu.Lock()
		u.state.IsDev = false
		u.state.CurrentVersion = version.Version
		u.state.Error = "auto-update checks disabled"
		u.state.LastCheck = time.Now().UTC()
		u.mu.Unlock()
		return u.State()
	}

	state := u.checkGitHub()

	u.mu.Lock()
	u.state = state
	u.mu.Unlock()

	u.saveState()

	if state.UpdateAvailable {
		u.sendNotification(state)
	}

	return state
}

// State returns the cached update state.
func (u *Updater) State() State {
	u.mu.RLock()
	defer u.mu.RUnlock()
	s := u.state
	s.IsDev = version.IsDevVersion()
	s.IsDocker = IsDocker()
	s.CurrentVersion = version.Version
	return s
}

func (u *Updater) loop() {
	defer close(u.done)

	if version.IsDevVersion() {
		log.Println("[updater] dev build detected, skipping update checks")
		return
	}

	// Initial check after a short delay to avoid slowing startup.
	select {
	case <-time.After(30 * time.Second):
		u.Check()
	case <-u.stopCh:
		return
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			u.Check()
		case <-u.stopCh:
			return
		}
	}
}

func (u *Updater) checkGitHub() State {
	state := State{
		CurrentVersion: version.Version,
		LastCheck:      time.Now().UTC(),
	}

	req, err := http.NewRequest("GET", u.baseURL, nil)
	if err != nil {
		state.Error = fmt.Sprintf("bad request: %v", err)
		return state
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ClawIDE/"+version.Version)

	resp, err := u.client.Do(req)
	if err != nil {
		state.Error = fmt.Sprintf("network error: %v", err)
		return state
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		state.Error = "GitHub API rate limit exceeded. Try again later."
		return state
	}

	if resp.StatusCode != http.StatusOK {
		state.Error = fmt.Sprintf("GitHub API returned status %d", resp.StatusCode)
		return state
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		state.Error = fmt.Sprintf("reading response: %v", err)
		return state
	}

	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		state.Error = fmt.Sprintf("parsing release: %v", err)
		return state
	}

	state.LatestVersion = release.TagName
	state.ReleaseURL = release.HTMLURL

	if version.CompareVersions(version.Version, release.TagName) < 0 {
		state.UpdateAvailable = true
	}

	// Find matching platform asset
	assetName := platformAssetName(release.TagName)
	for _, a := range release.Assets {
		if a.Name == assetName {
			state.AssetURL = a.BrowserDownloadURL
			state.AssetSize = a.Size
			break
		}
	}

	if state.UpdateAvailable && state.AssetURL == "" {
		state.Error = fmt.Sprintf("no build for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	return state
}

// platformAssetName returns the expected archive filename for this platform.
func platformAssetName(tag string) string {
	return fmt.Sprintf("clawide-%s-%s-%s.tar.gz", tag, runtime.GOOS, runtime.GOARCH)
}

func (u *Updater) sendNotification(state State) {
	if u.notificationStore == nil || u.sseHub == nil {
		return
	}

	n := model.Notification{
		ID:             uuid.New().String(),
		Title:          "Update Available",
		Body:           fmt.Sprintf("ClawIDE %s is available. Go to Settings to update.", state.LatestVersion),
		Source:         "system",
		Level:          "info",
		IdempotencyKey: "update-available-" + state.LatestVersion,
		CreatedAt:      time.Now().UTC(),
	}

	if err := u.notificationStore.Add(n); err != nil {
		log.Printf("[updater] notification add: %v", err)
		return
	}
	u.sseHub.Broadcast(&n)
}

func (u *Updater) loadState() {
	data, err := os.ReadFile(u.cfg.UpdateStatePath())
	if err != nil {
		return
	}
	var s State
	if json.Unmarshal(data, &s) == nil {
		u.state = s
	}
}

func (u *Updater) saveState() {
	u.mu.RLock()
	data, err := json.MarshalIndent(u.state, "", "  ")
	u.mu.RUnlock()
	if err != nil {
		return
	}
	os.WriteFile(u.cfg.UpdateStatePath(), data, 0644)
}
