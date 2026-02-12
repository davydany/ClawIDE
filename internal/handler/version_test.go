package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/davydany/ClawIDE/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionHandler(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	w := httptest.NewRecorder()

	h.Version(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var info version.Info
	require.NoError(t, json.NewDecoder(w.Body).Decode(&info))
	assert.Equal(t, "dev", info.Version)
	assert.Equal(t, "none", info.Commit)
	assert.Equal(t, "unknown", info.Date)
}
