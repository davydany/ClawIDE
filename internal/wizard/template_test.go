package wizard

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// minimalWizardFS creates a minimal in-memory filesystem with the required
// template directory structure: common/ + one language/framework.
func minimalWizardFS() fstest.MapFS {
	return fstest.MapFS{
		"templates/common/.gitignore": &fstest.MapFile{
			Data: []byte("*.log\n*.tmp\n"),
		},
		"templates/common/CLAUDE.md.tmpl": &fstest.MapFile{
			Data: []byte("# {{.ProjectName}}\n\nLanguage: {{.Language}}\nFramework: {{.Framework}}\n"),
		},
		"templates/python/django/README.md.tmpl": &fstest.MapFile{
			Data: []byte("# {{.ProjectName}} - Django\n\n{{.Description}}\n"),
		},
		"templates/python/django/manage.py.tmpl": &fstest.MapFile{
			Data: []byte("#!/usr/bin/env python\nimport os\nos.environ.setdefault('DJANGO_SETTINGS_MODULE', '{{.ProjectName}}.settings')\n"),
		},
		"templates/python/django/requirements.txt": &fstest.MapFile{
			Data: []byte("django>=5.0\npsycopg2-binary\n"),
		},
	}
}

// multiLanguageFS creates a filesystem with common + multiple language/framework combos.
func multiLanguageFS() fstest.MapFS {
	return fstest.MapFS{
		"templates/common/.gitignore": &fstest.MapFile{
			Data: []byte("*.log\n"),
		},
		"templates/python/django/app.py.tmpl": &fstest.MapFile{
			Data: []byte("# Django app for {{.ProjectName}}"),
		},
		"templates/python/flask/app.py.tmpl": &fstest.MapFile{
			Data: []byte("from flask import Flask\napp = Flask('{{.ProjectName}}')"),
		},
		"templates/go/gin/main.go.tmpl": &fstest.MapFile{
			Data: []byte("package main\n// {{.ProjectName}} gin server"),
		},
		// A non-directory file at the templates level should be skipped
		"templates/README.md": &fstest.MapFile{
			Data: []byte("This file should be ignored"),
		},
	}
}

// ---------------------------------------------------------------------------
// NewTemplateRegistry
// ---------------------------------------------------------------------------

func TestNewTemplateRegistry_LoadsFromFS(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)
	require.NotNil(t, reg)

	// Should have loaded python/django
	assert.True(t, reg.HasTemplateSet("python", "django"))
}

func TestNewTemplateRegistry_LoadsMultipleLanguages(t *testing.T) {
	reg, err := NewTemplateRegistry(multiLanguageFS())
	require.NoError(t, err)

	available := reg.ListAvailable()
	assert.Contains(t, available, "python/django")
	assert.Contains(t, available, "python/flask")
	assert.Contains(t, available, "go/gin")
	assert.Len(t, available, 3)
}

func TestNewTemplateRegistry_MissingTemplatesDir(t *testing.T) {
	emptyFS := fstest.MapFS{}
	_, err := NewTemplateRegistry(emptyFS)
	assert.Error(t, err, "should fail when templates directory is missing")
}

func TestNewTemplateRegistry_CommonHasNoFiles(t *testing.T) {
	// common/ dir exists but contains only a subdirectory, no actual files.
	// The framework templates should still load fine.
	fs := fstest.MapFS{
		"templates/common/subdir/nested.txt": &fstest.MapFile{Data: []byte("nested")},
		"templates/python/django/app.py": &fstest.MapFile{
			Data: []byte("# app"),
		},
	}

	reg, err := NewTemplateRegistry(fs)
	require.NoError(t, err)
	assert.True(t, reg.HasTemplateSet("python", "django"))
}

// ---------------------------------------------------------------------------
// HasTemplateSet
// ---------------------------------------------------------------------------

func TestHasTemplateSet_Found(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	assert.True(t, reg.HasTemplateSet("python", "django"))
}

func TestHasTemplateSet_NotFound(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	assert.False(t, reg.HasTemplateSet("python", "flask"))
	assert.False(t, reg.HasTemplateSet("go", "gin"))
	assert.False(t, reg.HasTemplateSet("", ""))
	assert.False(t, reg.HasTemplateSet("nonexistent", "nonexistent"))
}

// ---------------------------------------------------------------------------
// ListAvailable
// ---------------------------------------------------------------------------

func TestListAvailable_ReturnsAllSets(t *testing.T) {
	reg, err := NewTemplateRegistry(multiLanguageFS())
	require.NoError(t, err)

	available := reg.ListAvailable()
	assert.Len(t, available, 3)
	assert.ElementsMatch(t, []string{"python/django", "python/flask", "go/gin"}, available)
}

func TestListAvailable_EmptyWhenNoFrameworkSets(t *testing.T) {
	// Only common templates, no language/framework dirs
	fs := fstest.MapFS{
		"templates/common/.gitignore": &fstest.MapFile{Data: []byte("*.log")},
	}
	reg, err := NewTemplateRegistry(fs)
	require.NoError(t, err)
	assert.Empty(t, reg.ListAvailable())
}

// ---------------------------------------------------------------------------
// Get (merging common + framework)
// ---------------------------------------------------------------------------

func TestGet_MergesCommonAndFramework(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	set, err := reg.Get("python", "django")
	require.NoError(t, err)
	require.NotNil(t, set)

	assert.Equal(t, "python", set.Language)
	assert.Equal(t, "django", set.Framework)

	// Should have: .gitignore (common) + CLAUDE.md.tmpl (common) + README.md.tmpl + manage.py.tmpl + requirements.txt (all framework)
	paths := make(map[string]bool)
	for _, f := range set.Files {
		paths[f.RelPath] = true
	}

	assert.True(t, paths[".gitignore"], "common .gitignore should be merged")
	assert.True(t, paths["CLAUDE.md"], "common CLAUDE.md.tmpl should appear as CLAUDE.md")
	assert.True(t, paths["README.md"], "framework README.md.tmpl should appear as README.md")
	assert.True(t, paths["manage.py"], "framework manage.py.tmpl should appear as manage.py")
	assert.True(t, paths["requirements.txt"], "framework requirements.txt should be included")
}

func TestGet_FrameworkOverridesCommon(t *testing.T) {
	// Both common and framework have .gitignore — framework should win
	fs := fstest.MapFS{
		"templates/common/.gitignore": &fstest.MapFile{
			Data: []byte("common-ignore"),
		},
		"templates/python/django/.gitignore": &fstest.MapFile{
			Data: []byte("django-specific-ignore"),
		},
	}

	reg, err := NewTemplateRegistry(fs)
	require.NoError(t, err)

	set, err := reg.Get("python", "django")
	require.NoError(t, err)

	// Should have only one .gitignore (the framework one)
	var gitignoreCount int
	var gitignoreContent string
	for _, f := range set.Files {
		if f.RelPath == ".gitignore" {
			gitignoreCount++
			gitignoreContent = f.Content
		}
	}
	assert.Equal(t, 1, gitignoreCount, "should have exactly one .gitignore")
	assert.Equal(t, "django-specific-ignore", gitignoreContent, "framework .gitignore should override common")
}

func TestGet_NotFound(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	_, err = reg.Get("nonexistent", "framework")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no templates found")
}

// ---------------------------------------------------------------------------
// RenderFile
// ---------------------------------------------------------------------------

func TestRenderFile_TemplateContent(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	data := TemplateData{
		ProjectName: "my-app",
		Language:    "python",
		Framework:   "django",
		Description: "A test app",
	}

	tf := TemplateFile{
		RelPath:    "README.md",
		Content:    "# {{.ProjectName}}\n\n{{.Description}}\n",
		IsTemplate: true,
	}

	content, outPath, err := reg.RenderFile(tf, data)
	require.NoError(t, err)
	assert.Equal(t, "README.md", outPath)
	assert.Equal(t, "# my-app\n\nA test app\n", content)
}

func TestRenderFile_StaticContent(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	data := TemplateData{ProjectName: "my-app"}

	tf := TemplateFile{
		RelPath:    ".gitignore",
		Content:    "*.log\n*.tmp\n",
		IsTemplate: false,
	}

	content, outPath, err := reg.RenderFile(tf, data)
	require.NoError(t, err)
	assert.Equal(t, ".gitignore", outPath)
	assert.Equal(t, "*.log\n*.tmp\n", content, "static content should be returned as-is")
}

func TestRenderFile_TemplatedPath(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	data := TemplateData{ProjectName: "myapp"}

	tf := TemplateFile{
		RelPath:    "{{.ProjectName}}/settings/__init__.py",
		Content:    "# settings init",
		IsTemplate: false,
	}

	_, outPath, err := reg.RenderFile(tf, data)
	require.NoError(t, err)
	assert.Equal(t, "myapp/settings/__init__.py", outPath)
}

func TestRenderFile_CustomFuncMap(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	data := TemplateData{ProjectName: "MyApp", Language: "python"}

	tf := TemplateFile{
		RelPath:    "config.py",
		Content:    "APP_NAME = '{{lower .ProjectName}}'\nLANG = '{{upper .Language}}'\n",
		IsTemplate: true,
	}

	content, _, err := reg.RenderFile(tf, data)
	require.NoError(t, err)
	assert.Contains(t, content, "APP_NAME = 'myapp'")
	assert.Contains(t, content, "LANG = 'PYTHON'")
}

func TestRenderFile_InvalidPathTemplate(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	tf := TemplateFile{
		RelPath:    "{{.Invalid",
		Content:    "content",
		IsTemplate: false,
	}

	_, _, err = reg.RenderFile(tf, TemplateData{})
	assert.Error(t, err, "invalid path template should return error")
	assert.Contains(t, err.Error(), "parsing path template")
}

func TestRenderFile_InvalidContentTemplate(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	tf := TemplateFile{
		RelPath:    "file.txt",
		Content:    "{{.Missing",
		IsTemplate: true,
	}

	_, _, err = reg.RenderFile(tf, TemplateData{})
	assert.Error(t, err, "invalid content template should return error")
	assert.Contains(t, err.Error(), "parsing content template")
}

func TestRenderFile_MissingFieldInPath(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	tf := TemplateFile{
		RelPath:    "{{.NonexistentField}}/file.txt",
		Content:    "content",
		IsTemplate: false,
	}

	_, _, err = reg.RenderFile(tf, TemplateData{})
	assert.Error(t, err, "nonexistent field in path template should return error")
}

func TestRenderFile_DocFlags(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	data := TemplateData{
		ProjectName:        "test",
		HasPRD:          true,
		HasUIUX:         false,
		HasArchitecture: true,
		HasOther:        false,
	}

	tf := TemplateFile{
		RelPath:    "CLAUDE.md",
		Content:    "{{if .HasPRD}}PRD: yes{{end}}{{if .HasUIUX}}UI: yes{{end}}{{if .HasArchitecture}}ARCH: yes{{end}}",
		IsTemplate: true,
	}

	content, _, err := reg.RenderFile(tf, data)
	require.NoError(t, err)
	assert.Contains(t, content, "PRD: yes")
	assert.NotContains(t, content, "UI: yes")
	assert.Contains(t, content, "ARCH: yes")
}

// ---------------------------------------------------------------------------
// Template loading: .tmpl suffix handling
// ---------------------------------------------------------------------------

func TestLoadSet_TmplSuffixStripped(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	set, err := reg.Get("python", "django")
	require.NoError(t, err)

	for _, f := range set.Files {
		assert.False(t, f.RelPath == "README.md.tmpl", "should strip .tmpl suffix from RelPath")
		assert.False(t, f.RelPath == "CLAUDE.md.tmpl", "should strip .tmpl suffix from RelPath")
		assert.False(t, f.RelPath == "manage.py.tmpl", "should strip .tmpl suffix from RelPath")
	}
}

func TestLoadSet_TmplFilesMarkedAsTemplate(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	set, err := reg.Get("python", "django")
	require.NoError(t, err)

	fileMap := make(map[string]TemplateFile)
	for _, f := range set.Files {
		fileMap[f.RelPath] = f
	}

	// .tmpl files should have IsTemplate = true
	assert.True(t, fileMap["README.md"].IsTemplate, "README.md.tmpl should be marked as template")
	assert.True(t, fileMap["CLAUDE.md"].IsTemplate, "CLAUDE.md.tmpl should be marked as template")
	assert.True(t, fileMap["manage.py"].IsTemplate, "manage.py.tmpl should be marked as template")

	// Non-.tmpl files should have IsTemplate = false
	assert.False(t, fileMap[".gitignore"].IsTemplate, ".gitignore should not be marked as template")
	assert.False(t, fileMap["requirements.txt"].IsTemplate, "requirements.txt should not be marked as template")
}

// ---------------------------------------------------------------------------
// Template loading: non-directory entries skipped at language level
// ---------------------------------------------------------------------------

func TestLoadFromFS_SkipsNonDirectoryEntries(t *testing.T) {
	reg, err := NewTemplateRegistry(multiLanguageFS())
	require.NoError(t, err)

	available := reg.ListAvailable()
	// The README.md file at templates/ root should not create a set
	for _, key := range available {
		assert.NotContains(t, key, "README")
	}
}

// ---------------------------------------------------------------------------
// Integration: full render flow from registry
// ---------------------------------------------------------------------------

func TestIntegration_FullRenderFlow(t *testing.T) {
	reg, err := NewTemplateRegistry(minimalWizardFS())
	require.NoError(t, err)

	set, err := reg.Get("python", "django")
	require.NoError(t, err)

	data := TemplateData{
		ProjectName: "my-project",
		Language:    "python",
		Framework:   "django",
		Description: "Integration test project",
	}

	renderedPaths := make(map[string]string)
	for _, f := range set.Files {
		content, outPath, err := reg.RenderFile(f, data)
		require.NoError(t, err, "rendering %s", f.RelPath)
		renderedPaths[outPath] = content
	}

	// Verify key outputs
	assert.Contains(t, renderedPaths, ".gitignore")
	assert.Contains(t, renderedPaths, "CLAUDE.md")
	assert.Contains(t, renderedPaths, "README.md")
	assert.Contains(t, renderedPaths, "manage.py")
	assert.Contains(t, renderedPaths, "requirements.txt")

	// Verify template rendering in content
	assert.Contains(t, renderedPaths["CLAUDE.md"], "my-project")
	assert.Contains(t, renderedPaths["CLAUDE.md"], "python")
	assert.Contains(t, renderedPaths["README.md"], "my-project")
	assert.Contains(t, renderedPaths["README.md"], "Integration test project")
	assert.Contains(t, renderedPaths["manage.py"], "my-project.settings")
}
