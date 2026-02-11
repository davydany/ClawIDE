package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSettingsTest(t *testing.T) *Handlers {
	t.Helper()
	h, _ := setupHandlerWithRenderer(t)
	return h
}

func TestSettingsPage(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()

	h.SettingsPage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "settings")
}

func TestUpdateSettings(t *testing.T) {
	t.Run("updates allowed fields", func(t *testing.T) {
		h := setupSettingsTest(t)

		body := `{"host": "127.0.0.1", "port": 3000, "log_level": "debug"}`
		req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
		w := httptest.NewRecorder()

		h.UpdateSettings(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]string
		require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
		assert.Equal(t, "ok", result["status"])

		// Verify config file was written
		configPath := filepath.Join(h.cfg.DataDir, "config.json")
		data, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var saved map[string]any
		require.NoError(t, json.Unmarshal(data, &saved))
		assert.Equal(t, "127.0.0.1", saved["host"])
		assert.Equal(t, "debug", saved["log_level"])
	})

	t.Run("rejects disallowed fields", func(t *testing.T) {
		h := setupSettingsTest(t)

		body := `{"host": "10.0.0.1", "secret_field": "should-not-persist"}`
		req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
		w := httptest.NewRecorder()

		h.UpdateSettings(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		configPath := filepath.Join(h.cfg.DataDir, "config.json")
		data, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var saved map[string]any
		require.NoError(t, json.Unmarshal(data, &saved))
		assert.Equal(t, "10.0.0.1", saved["host"])
		_, hasSecret := saved["secret_field"]
		assert.False(t, hasSecret, "disallowed field should not be persisted")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		h := setupSettingsTest(t)

		req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader("not json"))
		w := httptest.NewRecorder()

		h.UpdateSettings(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
