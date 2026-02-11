package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	st, err := store.New(filepath.Join(dir, "state.json"))
	require.NoError(t, err)
	return st
}

func TestProjectLoader(t *testing.T) {
	t.Run("loads project into context", func(t *testing.T) {
		st := setupTestStore(t)
		require.NoError(t, st.AddProject(model.Project{ID: "proj-1", Name: "Test", Path: "/test"}))

		var gotProject model.Project
		mw := ProjectLoader(st)
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotProject = GetProject(r)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/projects/proj-1", nil)
		// Set chi URL param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "proj-1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "proj-1", gotProject.ID)
		assert.Equal(t, "Test", gotProject.Name)
	})

	t.Run("project not found returns 404", func(t *testing.T) {
		st := setupTestStore(t)
		mw := ProjectLoader(st)
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		req := httptest.NewRequest(http.MethodGet, "/projects/missing", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "missing")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("missing ID returns 400", func(t *testing.T) {
		st := setupTestStore(t)
		mw := ProjectLoader(st)
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		req := httptest.NewRequest(http.MethodGet, "/projects/", nil)
		rctx := chi.NewRouteContext()
		// ID param is empty
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetProject(t *testing.T) {
	t.Run("missing from context returns zero value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		p := GetProject(req)
		assert.Equal(t, model.Project{}, p)
	})
}
