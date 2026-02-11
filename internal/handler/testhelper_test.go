package handler

import (
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/davydany/ClawIDE/internal/tmpl"
	"github.com/stretchr/testify/require"
)

// minimalTemplateFS creates a minimal in-memory filesystem that satisfies
// the template renderer's expectations (base + pages + components).
func minimalTemplateFS() fstest.MapFS {
	return fstest.MapFS{
		"templates/base.html": &fstest.MapFile{
			Data: []byte(`{{define "base.html"}}<!DOCTYPE html><html><body>{{block "body" .}}{{end}}</body></html>{{end}}`),
		},
		"templates/pages/project-list.html": &fstest.MapFile{
			Data: []byte(`{{define "body"}}projects:{{if .Projects}}{{len .Projects}}{{else}}0{{end}}{{end}}`),
		},
		"templates/pages/workspace.html": &fstest.MapFile{
			Data: []byte(`{{define "body"}}workspace:{{.Project.Name}}{{end}}`),
		},
		"templates/pages/settings.html": &fstest.MapFile{
			Data: []byte(`{{define "body"}}settings{{end}}`),
		},
		"templates/partials/project-list.html": &fstest.MapFile{
			Data: []byte(`projects-partial:{{if .Projects}}{{len .Projects}}{{else}}0{{end}}`),
		},
		"templates/partials/session-list.html": &fstest.MapFile{
			Data: []byte(`sessions-partial:{{if .Sessions}}{{len .Sessions}}{{else}}0{{end}}`),
		},
		"templates/partials/workspace.html": &fstest.MapFile{
			Data: []byte(`workspace-partial:{{.Project.Name}}`),
		},
		"templates/partials/settings.html": &fstest.MapFile{
			Data: []byte(`settings-partial`),
		},
		"templates/components/project-card.html": &fstest.MapFile{
			Data: []byte(`{{define "project-card"}}card{{end}}`),
		},
	}
}

// setupHandlerWithRenderer creates a full Handlers instance with a working renderer.
func setupHandlerWithRenderer(t *testing.T) (*Handlers, *store.Store) {
	t.Helper()
	storeDir := t.TempDir()
	st, err := store.New(filepath.Join(storeDir, "state.json"))
	require.NoError(t, err)

	renderer, err := tmpl.New(minimalTemplateFS())
	require.NoError(t, err)

	projectsDir := t.TempDir()
	cfg := &config.Config{
		ProjectsDir: projectsDir,
		DataDir:     t.TempDir(),
		Host:        "0.0.0.0",
		Port:        9800,
	}
	h := New(cfg, st, renderer, nil)
	return h, st
}
