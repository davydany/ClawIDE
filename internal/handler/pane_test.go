package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenamePane(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		h, st := setupHandlerWithRenderer(t)
		project := model.Project{ID: "proj-1", Name: "Test", Path: t.TempDir()}
		require.NoError(t, st.AddProject(project))

		sess := model.Session{
			ID:        "s1",
			ProjectID: project.ID,
			Name:      "Session",
			Layout:    model.NewLeafPane("p1"),
		}
		require.NoError(t, st.AddSession(sess))

		form := url.Values{"name": {"My Server"}}
		req := httptest.NewRequest(http.MethodPatch, "/panes/p1/rename", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("sid", "s1")
		rctx.URLParams.Add("pid", "p1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.RenamePane(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"ok":true`)

		updated, ok := st.GetSession("s1")
		require.True(t, ok)
		assert.Equal(t, "My Server", updated.Layout.Name)
	})

	t.Run("pane not found", func(t *testing.T) {
		h, st := setupHandlerWithRenderer(t)
		project := model.Project{ID: "proj-1", Name: "Test", Path: t.TempDir()}
		require.NoError(t, st.AddProject(project))

		sess := model.Session{
			ID:        "s1",
			ProjectID: project.ID,
			Name:      "Session",
			Layout:    model.NewLeafPane("p1"),
		}
		require.NoError(t, st.AddSession(sess))

		form := url.Values{"name": {"My Server"}}
		req := httptest.NewRequest(http.MethodPatch, "/panes/missing/rename", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("sid", "s1")
		rctx.URLParams.Add("pid", "missing")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.RenamePane(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("session not found", func(t *testing.T) {
		h, _ := setupHandlerWithRenderer(t)

		form := url.Values{"name": {"My Server"}}
		req := httptest.NewRequest(http.MethodPatch, "/panes/p1/rename", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("sid", "nonexistent")
		rctx.URLParams.Add("pid", "p1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.RenamePane(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("empty name clears pane name", func(t *testing.T) {
		h, st := setupHandlerWithRenderer(t)
		project := model.Project{ID: "proj-1", Name: "Test", Path: t.TempDir()}
		require.NoError(t, st.AddProject(project))

		pane := model.NewLeafPane("p1")
		pane.Name = "Old Name"
		sess := model.Session{
			ID:        "s1",
			ProjectID: project.ID,
			Name:      "Session",
			Layout:    pane,
		}
		require.NoError(t, st.AddSession(sess))

		form := url.Values{"name": {""}}
		req := httptest.NewRequest(http.MethodPatch, "/panes/p1/rename", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("sid", "s1")
		rctx.URLParams.Add("pid", "p1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.RenamePane(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		updated, ok := st.GetSession("s1")
		require.True(t, ok)
		assert.Equal(t, "", updated.Layout.Name)
	})
}

func TestReplaceNodeInTree(t *testing.T) {
	t.Run("replaces first child", func(t *testing.T) {
		first := model.NewLeafPane("old")
		second := model.NewLeafPane("keep")
		root := &model.PaneNode{
			Type:   "split",
			First:  first,
			Second: second,
		}

		replacement := model.NewLeafPane("new")
		replaceNodeInTree(root, first, replacement)

		assert.Equal(t, replacement, root.First)
		assert.Equal(t, second, root.Second)
	})

	t.Run("replaces second child", func(t *testing.T) {
		first := model.NewLeafPane("keep")
		second := model.NewLeafPane("old")
		root := &model.PaneNode{
			Type:   "split",
			First:  first,
			Second: second,
		}

		replacement := model.NewLeafPane("new")
		replaceNodeInTree(root, second, replacement)

		assert.Equal(t, first, root.First)
		assert.Equal(t, replacement, root.Second)
	})

	t.Run("deep replacement", func(t *testing.T) {
		deepChild := model.NewLeafPane("deep-old")
		innerSplit := &model.PaneNode{
			Type:   "split",
			First:  deepChild,
			Second: model.NewLeafPane("sibling"),
		}
		root := &model.PaneNode{
			Type:   "split",
			First:  innerSplit,
			Second: model.NewLeafPane("other"),
		}

		replacement := model.NewLeafPane("deep-new")
		replaceNodeInTree(root, deepChild, replacement)

		assert.Equal(t, replacement, innerSplit.First)
	})

	t.Run("nil root no-op", func(t *testing.T) {
		// Should not panic
		replaceNodeInTree(nil, model.NewLeafPane("a"), model.NewLeafPane("b"))
	})

	t.Run("leaf root no-op", func(t *testing.T) {
		root := model.NewLeafPane("leaf")
		old := model.NewLeafPane("old")
		replacement := model.NewLeafPane("new")

		replaceNodeInTree(root, old, replacement)
		// leaf root remains unchanged
		assert.Equal(t, "leaf", root.PaneID)
	})
}
