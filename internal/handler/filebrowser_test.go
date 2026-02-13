package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAndValidatePath(t *testing.T) {
	root := "/projects/myapp"

	tests := []struct {
		name      string
		root      string
		path      string
		wantOK    bool
		wantPath  string
	}{
		{
			name:     "normal path",
			root:     root,
			path:     "src/main.go",
			wantOK:   true,
			wantPath: "/projects/myapp/src/main.go",
		},
		{
			name:     "traversal attack normalized safely",
			root:     root,
			path:     "../../etc/passwd",
			wantOK:   true,
			wantPath: "/projects/myapp/etc/passwd",
		},
		{
			name:     "root path",
			root:     root,
			path:     ".",
			wantOK:   true,
			wantPath: "/projects/myapp",
		},
		{
			name:     "nested path",
			root:     root,
			path:     "a/b/c/d.txt",
			wantOK:   true,
			wantPath: "/projects/myapp/a/b/c/d.txt",
		},
		{
			name:     "empty path",
			root:     root,
			path:     "",
			wantOK:   true,
			wantPath: "/projects/myapp",
		},
		{
			name:     "single dot-dot resolves to root",
			root:     root,
			path:     "..",
			wantOK:   true,
			wantPath: "/projects/myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveAndValidatePath(tt.root, tt.path)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantPath, got)
			}
		})
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "text content",
			data: []byte("hello world\nthis is text"),
			want: "text/plain; charset=utf-8",
		},
		{
			name: "binary content with null bytes",
			data: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00},
			want: "application/octet-stream",
		},
		{
			name: "empty content",
			data: []byte{},
			want: "text/plain; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectContentType(tt.data)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Helper to create a handler instance with a project in context
func setupFileBrowserTest(t *testing.T) (*Handlers, string) {
	t.Helper()
	projectDir := t.TempDir()

	storeDir := t.TempDir()
	st, err := store.New(filepath.Join(storeDir, "state.json"))
	require.NoError(t, err)

	cfg := &config.Config{}
	h := New(cfg, st, nil, nil, nil, nil, nil, nil, nil, nil)

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# Test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, ".hidden"), []byte("secret"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "src", "main.go"), []byte("package main"), 0644))

	return h, projectDir
}

// withProjectContext injects a project into the request context using the same
// middleware flow. We create a chi router with the ProjectLoader middleware to
// set the context correctly, then extract the enriched request.
func withProjectContext(req *http.Request, project model.Project) *http.Request {
	// Use the store+middleware approach: add project to a temp store,
	// set up chi route context, and run through the middleware.
	// But that's heavyweight. Instead, we can use the same technique the
	// middleware uses: go through a handler that captures the request after
	// the middleware sets the context.
	//
	// Simpler approach: since we control the store, just add the project
	// and use the middleware directly in test helpers.
	// For unit tests of pure functions, we don't need context at all.
	// For handler tests, we use the middleware inline.

	// For tests that need the project in context, we replicate the middleware's
	// behavior. The middleware uses an unexported contextKey type, so we run
	// the request through a minimal middleware chain.
	storeDir, _ := os.MkdirTemp("", "handler-test-store-*")
	st, _ := store.New(filepath.Join(storeDir, "state.json"))
	_ = st.AddProject(project)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", project.ID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	// Run through the middleware to set the project context
	var enrichedReq *http.Request
	mw := middleware.ProjectLoader(st)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enrichedReq = r
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	return enrichedReq
}

func TestListFiles(t *testing.T) {
	h, projectDir := setupFileBrowserTest(t)

	t.Run("returns dir listing JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/files?path=.", nil)
		req = withProjectContext(req, model.Project{ID: "p1", Path: projectDir})
		w := httptest.NewRecorder()

		h.ListFiles(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var entries []FileEntry
		require.NoError(t, json.NewDecoder(w.Body).Decode(&entries))
		// Should have "src" dir and "README.md" (hidden files filtered)
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name
		}
		assert.Contains(t, names, "src")
		assert.Contains(t, names, "README.md")
	})

	t.Run("hidden files filtered by default", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/files?path=.", nil)
		req = withProjectContext(req, model.Project{ID: "p1", Path: projectDir})
		w := httptest.NewRecorder()

		h.ListFiles(w, req)

		var entries []FileEntry
		require.NoError(t, json.NewDecoder(w.Body).Decode(&entries))
		for _, e := range entries {
			assert.False(t, strings.HasPrefix(e.Name, "."), "hidden file should be filtered: %s", e.Name)
		}
	})

	t.Run("hidden files included when requested", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/files?path=.&hidden=true", nil)
		req = withProjectContext(req, model.Project{ID: "p1", Path: projectDir})
		w := httptest.NewRecorder()

		h.ListFiles(w, req)

		var entries []FileEntry
		require.NoError(t, json.NewDecoder(w.Body).Decode(&entries))
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name
		}
		assert.Contains(t, names, ".hidden")
	})

	t.Run("path not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/files?path=nonexistent", nil)
		req = withProjectContext(req, model.Project{ID: "p1", Path: projectDir})
		w := httptest.NewRecorder()

		h.ListFiles(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestReadFile(t *testing.T) {
	h, projectDir := setupFileBrowserTest(t)

	t.Run("reads file content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/file?path=README.md", nil)
		req = withProjectContext(req, model.Project{ID: "p1", Path: projectDir})
		w := httptest.NewRecorder()

		h.ReadFile(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "# Test", w.Body.String())
		assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	})

	t.Run("file too large", func(t *testing.T) {
		// Create a file larger than 1MB
		largeData := make([]byte, maxFileReadSize+1)
		largePath := filepath.Join(projectDir, "large.bin")
		require.NoError(t, os.WriteFile(largePath, largeData, 0644))

		req := httptest.NewRequest(http.MethodGet, "/file?path=large.bin", nil)
		req = withProjectContext(req, model.Project{ID: "p1", Path: projectDir})
		w := httptest.NewRecorder()

		h.ReadFile(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("file not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/file?path=nonexistent.txt", nil)
		req = withProjectContext(req, model.Project{ID: "p1", Path: projectDir})
		w := httptest.NewRecorder()

		h.ReadFile(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("missing path parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/file", nil)
		req = withProjectContext(req, model.Project{ID: "p1", Path: projectDir})
		w := httptest.NewRecorder()

		h.ReadFile(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestWriteFile(t *testing.T) {
	h, projectDir := setupFileBrowserTest(t)

	t.Run("writes file and reads it back", func(t *testing.T) {
		body := strings.NewReader("new file content")
		req := httptest.NewRequest(http.MethodPut, "/file?path=new.txt", body)
		req = withProjectContext(req, model.Project{ID: "p1", Path: projectDir})
		w := httptest.NewRecorder()

		h.WriteFile(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify the file was written
		data, err := os.ReadFile(filepath.Join(projectDir, "new.txt"))
		require.NoError(t, err)
		assert.Equal(t, "new file content", string(data))
	})

	t.Run("creates parent directories", func(t *testing.T) {
		body := strings.NewReader("deep file")
		req := httptest.NewRequest(http.MethodPut, "/file?path=a/b/c/deep.txt", body)
		req = withProjectContext(req, model.Project{ID: "p1", Path: projectDir})
		w := httptest.NewRecorder()

		h.WriteFile(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		data, err := os.ReadFile(filepath.Join(projectDir, "a", "b", "c", "deep.txt"))
		require.NoError(t, err)
		assert.Equal(t, "deep file", string(data))
	})

	t.Run("path traversal normalized safely", func(t *testing.T) {
		// The path ../../evil.txt gets normalized to evil.txt within the project root
		body := strings.NewReader("safe content")
		req := httptest.NewRequest(http.MethodPut, "/file?path=../../evil.txt", body)
		req = withProjectContext(req, model.Project{ID: "p1", Path: projectDir})
		w := httptest.NewRecorder()

		h.WriteFile(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify it was written inside the project dir, not outside
		data, err := os.ReadFile(filepath.Join(projectDir, "evil.txt"))
		require.NoError(t, err)
		assert.Equal(t, "safe content", string(data))
	})
}
