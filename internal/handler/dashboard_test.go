package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboard(t *testing.T) {
	t.Run("renders with no projects", func(t *testing.T) {
		h, _ := setupHandlerWithRenderer(t)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Dashboard(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "projects:")
	})

	t.Run("renders with projects", func(t *testing.T) {
		h, st := setupHandlerWithRenderer(t)
		require.NoError(t, st.AddProject(model.Project{ID: "p1", Name: "Test", Path: "/test"}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Dashboard(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("discovers unregistered project directories", func(t *testing.T) {
		h, st := setupHandlerWithRenderer(t)

		// Create directories in the projects dir
		os.MkdirAll(filepath.Join(h.cfg.ProjectsDir, "unregistered"), 0755)
		os.MkdirAll(filepath.Join(h.cfg.ProjectsDir, "registered"), 0755)

		// Register one
		require.NoError(t, st.AddProject(model.Project{
			ID:   "p1",
			Name: "Registered",
			Path: filepath.Join(h.cfg.ProjectsDir, "registered"),
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Dashboard(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty projects dir handled gracefully", func(t *testing.T) {
		h, _ := setupHandlerWithRenderer(t)
		h.cfg.ProjectsDir = "" // empty projects dir

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		h.Dashboard(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
