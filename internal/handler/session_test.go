package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSessionTest(t *testing.T) (*Handlers, *store.Store, model.Project) {
	t.Helper()
	h, st := setupHandlerWithRenderer(t)

	project := model.Project{ID: "proj-1", Name: "Test", Path: t.TempDir()}
	require.NoError(t, st.AddProject(project))

	return h, st, project
}

func withProjectMiddleware(req *http.Request, st *store.Store, projectID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", projectID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	var enrichedReq *http.Request
	mw := middleware.ProjectLoader(st)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enrichedReq = r
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	return enrichedReq
}

func TestCreateSession(t *testing.T) {
	t.Run("creates session and redirects non-HTMX", func(t *testing.T) {
		h, st, project := setupSessionTest(t)

		form := url.Values{"name": {"My Session"}, "branch": {"main"}}
		req := httptest.NewRequest(http.MethodPost, "/projects/proj-1/sessions", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = withProjectMiddleware(req, st, project.ID)
		w := httptest.NewRecorder()

		h.CreateSession(w, req)

		assert.Equal(t, http.StatusSeeOther, w.Code)

		sessions := st.GetSessions(project.ID)
		require.Len(t, sessions, 1)
		assert.Equal(t, "My Session", sessions[0].Name)
		assert.Equal(t, "main", sessions[0].Branch)
		assert.NotNil(t, sessions[0].Layout)
	})

	t.Run("default name when empty", func(t *testing.T) {
		h, st, project := setupSessionTest(t)

		form := url.Values{}
		req := httptest.NewRequest(http.MethodPost, "/projects/proj-1/sessions", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = withProjectMiddleware(req, st, project.ID)
		w := httptest.NewRecorder()

		h.CreateSession(w, req)

		sessions := st.GetSessions(project.ID)
		require.Len(t, sessions, 1)
		assert.Contains(t, sessions[0].Name, "Session ")
	})

	t.Run("default workDir to project path", func(t *testing.T) {
		h, st, project := setupSessionTest(t)

		form := url.Values{"name": {"Test"}}
		req := httptest.NewRequest(http.MethodPost, "/projects/proj-1/sessions", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = withProjectMiddleware(req, st, project.ID)
		w := httptest.NewRecorder()

		h.CreateSession(w, req)

		sessions := st.GetSessions(project.ID)
		require.Len(t, sessions, 1)
		assert.Equal(t, project.Path, sessions[0].WorkDir)
	})
}

func TestRenameSession(t *testing.T) {
	t.Run("renames session via HTMX", func(t *testing.T) {
		h, st, project := setupSessionTest(t)
		sess := model.Session{ID: "s1", ProjectID: project.ID, Name: "Old Name", Layout: model.NewLeafPane("s1")}
		require.NoError(t, st.AddSession(sess))

		form := url.Values{"name": {"New Name"}}
		req := httptest.NewRequest(http.MethodPut, "/sessions/s1", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req = withProjectMiddleware(req, st, project.ID)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", project.ID)
		rctx.URLParams.Add("sid", "s1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Re-run through project middleware to get project context with chi params
		var enrichedReq *http.Request
		mw := middleware.ProjectLoader(st)
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			enrichedReq = r
		}))
		handler.ServeHTTP(httptest.NewRecorder(), req)

		w := httptest.NewRecorder()
		h.RenameSession(w, enrichedReq)

		assert.Equal(t, http.StatusOK, w.Code)

		updated, ok := st.GetSession("s1")
		require.True(t, ok)
		assert.Equal(t, "New Name", updated.Name)
	})

	t.Run("session not found", func(t *testing.T) {
		h, st, project := setupSessionTest(t)

		form := url.Values{"name": {"New Name"}}
		req := httptest.NewRequest(http.MethodPut, "/sessions/missing", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = withProjectMiddleware(req, st, project.ID)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", project.ID)
		rctx.URLParams.Add("sid", "missing")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		var enrichedReq *http.Request
		mw := middleware.ProjectLoader(st)
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			enrichedReq = r
		}))
		handler.ServeHTTP(httptest.NewRecorder(), req)

		w := httptest.NewRecorder()
		h.RenameSession(w, enrichedReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("empty name", func(t *testing.T) {
		h, st, project := setupSessionTest(t)
		sess := model.Session{ID: "s2", ProjectID: project.ID, Name: "Test", Layout: model.NewLeafPane("s2")}
		require.NoError(t, st.AddSession(sess))

		form := url.Values{"name": {""}}
		req := httptest.NewRequest(http.MethodPut, "/sessions/s2", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = withProjectMiddleware(req, st, project.ID)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", project.ID)
		rctx.URLParams.Add("sid", "s2")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		var enrichedReq *http.Request
		mw := middleware.ProjectLoader(st)
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			enrichedReq = r
		}))
		handler.ServeHTTP(httptest.NewRecorder(), req)

		w := httptest.NewRecorder()
		h.RenameSession(w, enrichedReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
