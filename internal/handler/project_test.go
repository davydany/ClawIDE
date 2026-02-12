package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProjectTest(t *testing.T) (*Handlers, *store.Store) {
	t.Helper()
	return setupHandlerWithRenderer(t)
}

func TestListProjects(t *testing.T) {
	t.Run("renders project list", func(t *testing.T) {
		h, st := setupHandlerWithRenderer(t)
		require.NoError(t, st.AddProject(model.Project{ID: "p1", Name: "Test"}))

		req := httptest.NewRequest(http.MethodGet, "/projects", nil)
		w := httptest.NewRecorder()

		h.ListProjects(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "projects:")
	})
}

func TestProjectWorkspace(t *testing.T) {
	t.Run("renders workspace", func(t *testing.T) {
		h, st := setupHandlerWithRenderer(t)
		project := model.Project{ID: "pw-1", Name: "Workspace Test", Path: t.TempDir()}
		require.NoError(t, st.AddProject(project))

		req := httptest.NewRequest(http.MethodGet, "/projects/pw-1", nil)
		req = withProjectMiddleware(req, st, "pw-1")
		w := httptest.NewRecorder()

		h.ProjectWorkspace(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "workspace:")
	})
}

func TestCreateProject(t *testing.T) {
	t.Run("missing name", func(t *testing.T) {
		h, _ := setupProjectTest(t)
		form := url.Values{"path": {"/tmp"}}
		req := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		h.CreateProject(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing path", func(t *testing.T) {
		h, _ := setupProjectTest(t)
		form := url.Values{"name": {"Test"}}
		req := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		h.CreateProject(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("path not a directory", func(t *testing.T) {
		h, _ := setupProjectTest(t)
		form := url.Values{"name": {"Test"}, "path": {"/nonexistent/path"}}
		req := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		h.CreateProject(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("successful creation via HTMX", func(t *testing.T) {
		h, st := setupProjectTest(t)
		dir := t.TempDir()

		form := url.Values{"name": {"My Project"}, "path": {dir}}
		req := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()

		h.CreateProject(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "/", w.Header().Get("HX-Redirect"))

		projects := st.GetProjects()
		require.Len(t, projects, 1)
		assert.Equal(t, "My Project", projects[0].Name)
		assert.Equal(t, dir, projects[0].Path)
	})

	t.Run("successful creation non-HTMX redirects", func(t *testing.T) {
		h, _ := setupProjectTest(t)
		dir := t.TempDir()

		form := url.Values{"name": {"My Project"}, "path": {dir}}
		req := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		h.CreateProject(w, req)

		assert.Equal(t, http.StatusSeeOther, w.Code)
	})
}

func TestDeleteProject(t *testing.T) {
	t.Run("successful delete via HTMX", func(t *testing.T) {
		h, st := setupProjectTest(t)
		require.NoError(t, st.AddProject(model.Project{ID: "p1", Name: "Test"}))

		req := httptest.NewRequest(http.MethodDelete, "/projects/p1", nil)
		req.Header.Set("HX-Request", "true")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "p1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		h.DeleteProject(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "/", w.Header().Get("HX-Redirect"))

		_, ok := st.GetProject("p1")
		assert.False(t, ok)
	})

	t.Run("delete missing project returns 500", func(t *testing.T) {
		h, _ := setupProjectTest(t)

		req := httptest.NewRequest(http.MethodDelete, "/projects/missing", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "missing")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		h.DeleteProject(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("non-HTMX redirects", func(t *testing.T) {
		h, st := setupProjectTest(t)
		require.NoError(t, st.AddProject(model.Project{ID: "p2", Name: "Test2"}))

		req := httptest.NewRequest(http.MethodDelete, "/projects/p2", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "p2")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		h.DeleteProject(w, req)
		assert.Equal(t, http.StatusSeeOther, w.Code)
	})
}

func TestScanProjects(t *testing.T) {
	t.Run("scans projects directory", func(t *testing.T) {
		projectsDir := t.TempDir()
		os.MkdirAll(filepath.Join(projectsDir, "project-a"), 0755)
		os.MkdirAll(filepath.Join(projectsDir, "project-b"), 0755)
		os.MkdirAll(filepath.Join(projectsDir, ".hidden"), 0755)
		os.WriteFile(filepath.Join(projectsDir, "file.txt"), []byte("not a dir"), 0644)

		storeDir := t.TempDir()
		st, err := store.New(filepath.Join(storeDir, "state.json"))
		require.NoError(t, err)

		cfg := &config.Config{ProjectsDir: projectsDir}
		h := New(cfg, st, nil, nil, nil, nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/scan-projects", nil)
		w := httptest.NewRecorder()

		h.ScanProjects(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result ScanResult
		require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
		assert.Len(t, result.Dirs, 2) // project-a and project-b (not .hidden, not file.txt)

		names := make([]string, len(result.Dirs))
		for i, d := range result.Dirs {
			names[i] = d.Name
		}
		assert.Contains(t, names, "project-a")
		assert.Contains(t, names, "project-b")
	})

	t.Run("invalid projects directory", func(t *testing.T) {
		storeDir := t.TempDir()
		st, _ := store.New(filepath.Join(storeDir, "state.json"))
		cfg := &config.Config{ProjectsDir: "/nonexistent/path"}
		h := New(cfg, st, nil, nil, nil, nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/scan-projects", nil)
		w := httptest.NewRecorder()

		h.ScanProjects(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
