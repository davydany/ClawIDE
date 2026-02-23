package wizard

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generatorTestFS creates a minimal template FS suitable for generator tests.
func generatorTestFS() fstest.MapFS {
	return fstest.MapFS{
		"templates/common/CLAUDE.md.tmpl": &fstest.MapFile{
			Data: []byte("# {{.ProjectName}}\n\nLanguage: {{.Language}}\nFramework: {{.Framework}}\n{{if .HasPRD}}See docs/supporting/prd.md{{end}}\n"),
		},
		"templates/common/README.md.tmpl": &fstest.MapFile{
			Data: []byte("# {{.ProjectName}}\n\n{{.Description}}\n"),
		},
		"templates/common/.gitignore": &fstest.MapFile{
			Data: []byte("*.log\n.env\n"),
		},
		"templates/python/django/requirements.in": &fstest.MapFile{
			Data: []byte("django>=5.1\n"),
		},
		"templates/python/django/Dockerfile.tmpl": &fstest.MapFile{
			Data: []byte("FROM python:3.13\nWORKDIR /app\n"),
		},
	}
}

func setupGenerator(t *testing.T) (*Generator, *JobTracker) {
	t.Helper()
	reg, err := NewTemplateRegistry(generatorTestFS())
	require.NoError(t, err)

	tracker := NewJobTracker()
	gen := NewGenerator(reg, tracker)
	return gen, tracker
}

func TestGenerator_Generate_Success(t *testing.T) {
	gen, tracker := setupGenerator(t)
	outDir := t.TempDir()

	req := WizardRequest{
		ProjectName: "myapp",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outDir,
		Description: "A test application",
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.NoError(t, err)

	snap := job.Snapshot()
	assert.Equal(t, JobStatusCompleted, snap.Status)
	assert.Equal(t, filepath.Join(outDir, "myapp"), snap.OutputDir)

	// Verify generated files exist
	projectDir := filepath.Join(outDir, "myapp")
	assertFileExists(t, filepath.Join(projectDir, "CLAUDE.md"))
	assertFileExists(t, filepath.Join(projectDir, "README.md"))
	assertFileExists(t, filepath.Join(projectDir, ".gitignore"))
	assertFileExists(t, filepath.Join(projectDir, "requirements.in"))
	assertFileExists(t, filepath.Join(projectDir, "Dockerfile"))

	// Verify CLAUDE.md content
	content, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "myapp")
	assert.Contains(t, string(content), "Python")
	assert.Contains(t, string(content), "Django")

	// Verify README content
	content, err = os.ReadFile(filepath.Join(projectDir, "README.md"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "myapp")
	assert.Contains(t, string(content), "A test application")

	// Verify git was initialized
	assertFileExists(t, filepath.Join(projectDir, ".git"))
}

func TestGenerator_Generate_WithDocs(t *testing.T) {
	gen, tracker := setupGenerator(t)
	outDir := t.TempDir()

	// Create supporting doc files
	docDir := t.TempDir()
	prdFile := filepath.Join(docDir, "prd.md")
	require.NoError(t, os.WriteFile(prdFile, []byte("# Product Requirements\nThis is the PRD."), 0644))

	req := WizardRequest{
		ProjectName: "myapp",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outDir,
		DocPRD:      prdFile,
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.NoError(t, err)

	projectDir := filepath.Join(outDir, "myapp")

	// Verify docs were copied
	copiedPRD := filepath.Join(projectDir, "docs", "supporting", "prd.md")
	assertFileExists(t, copiedPRD)

	content, err := os.ReadFile(copiedPRD)
	require.NoError(t, err)
	assert.Equal(t, "# Product Requirements\nThis is the PRD.", string(content))

	// Verify CLAUDE.md references the PRD
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(claudeContent), "prd.md")
}

func TestGenerator_Generate_ValidationFailure(t *testing.T) {
	gen, tracker := setupGenerator(t)

	req := WizardRequest{
		// Missing required fields
		ProjectName: "",
		Language:    "",
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	assert.Error(t, err)

	snap := job.Snapshot()
	assert.Equal(t, JobStatusFailed, snap.Status)
	assert.Contains(t, snap.Error, "validation failed")
}

func TestGenerator_Generate_Rollback(t *testing.T) {
	// Use a template registry that will fail during rendering
	fsys := fstest.MapFS{
		"templates/common/CLAUDE.md.tmpl": &fstest.MapFile{
			Data: []byte("# {{.ProjectName}}"),
		},
		"templates/python/django/bad.tmpl": &fstest.MapFile{
			// Invalid template that will fail to render
			Data: []byte("{{.NonexistentMethod}}"),
		},
	}

	reg, err := NewTemplateRegistry(fsys)
	require.NoError(t, err)

	tracker := NewJobTracker()
	gen := NewGenerator(reg, tracker)

	outDir := t.TempDir()
	req := WizardRequest{
		ProjectName: "failapp",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outDir,
	}

	job := tracker.Add(req)
	err = gen.Generate(context.Background(), job)
	assert.Error(t, err)

	snap := job.Snapshot()
	assert.Equal(t, JobStatusRolledBack, snap.Status)

	// Verify directory was cleaned up
	_, statErr := os.Stat(filepath.Join(outDir, "failapp"))
	assert.True(t, os.IsNotExist(statErr), "project directory should be removed after rollback")
}

func TestGenerator_CopyDocs_AllTypes(t *testing.T) {
	gen, tracker := setupGenerator(t)
	outDir := t.TempDir()

	// Create all doc types
	docDir := t.TempDir()
	docs := map[string]string{
		"prd.md":  "PRD content",
		"uiux.md": "UI/UX content",
		"arch.md": "Architecture content",
		"other.md": "Other content",
	}
	for name, content := range docs {
		require.NoError(t, os.WriteFile(filepath.Join(docDir, name), []byte(content), 0644))
	}

	req := WizardRequest{
		ProjectName:     "docapp",
		Language:        "python",
		Framework:       "django",
		OutputDir:       outDir,
		DocPRD:          filepath.Join(docDir, "prd.md"),
		DocUIUX:         filepath.Join(docDir, "uiux.md"),
		DocArchitecture: filepath.Join(docDir, "arch.md"),
		DocOther:        filepath.Join(docDir, "other.md"),
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.NoError(t, err)

	supportingDir := filepath.Join(outDir, "docapp", "docs", "supporting")
	for _, expected := range []string{"prd.md", "uiux.md", "architecture.md", "other.md"} {
		assertFileExists(t, filepath.Join(supportingDir, expected))
	}
}

func TestGenerator_Generate_DocsDirectoryCreated(t *testing.T) {
	gen, tracker := setupGenerator(t)
	outDir := t.TempDir()

	req := WizardRequest{
		ProjectName: "nodocs",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outDir,
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.NoError(t, err)

	// docs/supporting/ should still be created even without docs
	docsDir := filepath.Join(outDir, "nodocs", "docs", "supporting")
	info, err := os.Stat(docsDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestGenerator_StepsTracking(t *testing.T) {
	gen, tracker := setupGenerator(t)
	outDir := t.TempDir()

	req := WizardRequest{
		ProjectName: "stepapp",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outDir,
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.NoError(t, err)

	snap := job.Snapshot()

	// Verify each step was completed
	for _, step := range snap.Steps {
		assert.Equal(t, JobStatusCompleted, step.Status,
			"step %s should be completed, got %s (msg: %s)", step.Name, step.Status, step.Message)
	}
}

// ---------------------------------------------------------------------------
// copyFile
// ---------------------------------------------------------------------------

func TestCopyFile_Success(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcPath := filepath.Join(srcDir, "source.txt")
	dstPath := filepath.Join(dstDir, "dest.txt")
	require.NoError(t, os.WriteFile(srcPath, []byte("hello world"), 0644))

	err := copyFile(srcPath, dstPath)
	require.NoError(t, err)

	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(content))
}

func TestCopyFile_SourceNotFound(t *testing.T) {
	dstPath := filepath.Join(t.TempDir(), "dest.txt")
	err := copyFile("/nonexistent/file.txt", dstPath)
	assert.Error(t, err)
}

func TestCopyFile_DestDirNotWritable(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "source.txt")
	require.NoError(t, os.WriteFile(srcPath, []byte("content"), 0644))

	err := copyFile(srcPath, "/nonexistent/dir/dest.txt")
	assert.Error(t, err)
}

func TestCopyFile_LargeFile(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	srcPath := filepath.Join(srcDir, "large.bin")
	dstPath := filepath.Join(dstDir, "large.bin")
	require.NoError(t, os.WriteFile(srcPath, data, 0644))

	err := copyFile(srcPath, dstPath)
	require.NoError(t, err)

	copied, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, data, copied)
}

// ---------------------------------------------------------------------------
// rollback
// ---------------------------------------------------------------------------

func TestRollback_RemovesDirectory(t *testing.T) {
	gen, _ := setupGenerator(t)
	dir := filepath.Join(t.TempDir(), "project-to-remove")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644))

	job := NewJob(WizardRequest{ProjectName: "test"})
	gen.rollback(dir, job)

	_, err := os.Stat(dir)
	assert.True(t, os.IsNotExist(err))

	snap := job.Snapshot()
	assert.Equal(t, JobStatusRolledBack, snap.Status)
}

func TestRollback_NonexistentDirectory(t *testing.T) {
	gen, _ := setupGenerator(t)
	job := NewJob(WizardRequest{ProjectName: "test"})

	gen.rollback("/nonexistent/path", job)

	snap := job.Snapshot()
	assert.Equal(t, JobStatusRolledBack, snap.Status)
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestGenerator_Generate_InvalidProjectName(t *testing.T) {
	gen, tracker := setupGenerator(t)
	req := WizardRequest{
		ProjectName: "-invalid",
		Language:    "python",
		Framework:   "django",
		OutputDir:   t.TempDir(),
	}
	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	assert.Error(t, err)
	snap := job.Snapshot()
	assert.Equal(t, JobStatusFailed, snap.Status)
}

func TestGenerator_Generate_UnsupportedFrameworkTemplate(t *testing.T) {
	gen, tracker := setupGenerator(t)
	req := WizardRequest{
		ProjectName: "test",
		Language:    "python",
		Framework:   "flask",
		OutputDir:   t.TempDir(),
	}
	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	assert.Error(t, err)
}

func TestGenerator_Generate_GitInitCreatesRepo(t *testing.T) {
	gen, tracker := setupGenerator(t)
	outDir := t.TempDir()
	req := WizardRequest{
		ProjectName: "gitapp",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outDir,
	}
	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.NoError(t, err)

	gitDir := filepath.Join(outDir, "gitapp", ".git")
	info, err := os.Stat(gitDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestNewGenerator_SetsDefaultTimeout(t *testing.T) {
	reg, err := NewTemplateRegistry(generatorTestFS())
	require.NoError(t, err)
	tracker := NewJobTracker()
	gen := NewGenerator(reg, tracker)
	require.NotNil(t, gen)
	assert.NotNil(t, gen.executor)
}

// ---------------------------------------------------------------------------
// Empty project generation
// ---------------------------------------------------------------------------

func TestGenerator_GenerateEmpty_Success(t *testing.T) {
	gen, tracker := setupGenerator(t)
	outDir := t.TempDir()

	req := WizardRequest{
		ProjectName:  "empty-app",
		EmptyProject: true,
		OutputDir:    outDir,
		Description:  "An empty project",
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.NoError(t, err)

	snap := job.Snapshot()
	assert.Equal(t, JobStatusCompleted, snap.Status)
	assert.Equal(t, filepath.Join(outDir, "empty-app"), snap.OutputDir)

	projectDir := filepath.Join(outDir, "empty-app")

	// Verify CLAUDE.md exists and has correct content
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(claudeContent), "# empty-app")
	assert.Contains(t, string(claudeContent), "An empty project")

	// Verify NO template files (no README.md, no .gitignore from templates, no requirements.in, no Dockerfile)
	_, err = os.Stat(filepath.Join(projectDir, "README.md"))
	assert.True(t, os.IsNotExist(err), "empty project should NOT have README.md from templates")

	_, err = os.Stat(filepath.Join(projectDir, "requirements.in"))
	assert.True(t, os.IsNotExist(err), "empty project should NOT have requirements.in")

	_, err = os.Stat(filepath.Join(projectDir, "Dockerfile"))
	assert.True(t, os.IsNotExist(err), "empty project should NOT have Dockerfile")

	// Verify git was initialized
	assertFileExists(t, filepath.Join(projectDir, ".git"))

	// Verify docs/supporting/ directory exists
	info, err := os.Stat(filepath.Join(projectDir, "docs", "supporting"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestGenerator_GenerateEmpty_WithDocs(t *testing.T) {
	gen, tracker := setupGenerator(t)
	outDir := t.TempDir()

	// Create a doc file
	docDir := t.TempDir()
	prdFile := filepath.Join(docDir, "prd.md")
	require.NoError(t, os.WriteFile(prdFile, []byte("# Product Requirements\nDetails here."), 0644))

	req := WizardRequest{
		ProjectName:  "empty-with-docs",
		EmptyProject: true,
		OutputDir:    outDir,
		Description:  "Empty project with docs",
		DocPRD:       prdFile,
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.NoError(t, err)

	projectDir := filepath.Join(outDir, "empty-with-docs")

	// Verify docs were copied
	copiedPRD := filepath.Join(projectDir, "docs", "supporting", "prd.md")
	assertFileExists(t, copiedPRD)
	content, err := os.ReadFile(copiedPRD)
	require.NoError(t, err)
	assert.Equal(t, "# Product Requirements\nDetails here.", string(content))

	// Verify CLAUDE.md references the doc
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(claudeContent), "prd.md")
	assert.Contains(t, string(claudeContent), "docs/supporting/")
}

func TestGenerator_GenerateEmpty_StepsTracking(t *testing.T) {
	gen, tracker := setupGenerator(t)
	outDir := t.TempDir()

	req := WizardRequest{
		ProjectName:  "step-track-empty",
		EmptyProject: true,
		OutputDir:    outDir,
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.NoError(t, err)

	snap := job.Snapshot()
	expectedSteps := []string{"validate", "create_directory", "copy_docs", "generate_claude_md", "init_git"}
	require.Len(t, snap.Steps, len(expectedSteps))

	for i, expected := range expectedSteps {
		assert.Equal(t, expected, snap.Steps[i].Name)
		assert.Equal(t, JobStatusCompleted, snap.Steps[i].Status,
			"step %s should be completed", expected)
	}
}

func TestGenerator_GenerateEmpty_ValidationFailure(t *testing.T) {
	gen, tracker := setupGenerator(t)

	req := WizardRequest{
		ProjectName:  "",
		EmptyProject: true,
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")

	snap := job.Snapshot()
	assert.Equal(t, JobStatusFailed, snap.Status)
}

func TestGenerator_GenerateEmpty_ClaudeMdNoDocs(t *testing.T) {
	gen, tracker := setupGenerator(t)
	outDir := t.TempDir()

	req := WizardRequest{
		ProjectName:  "no-docs-empty",
		EmptyProject: true,
		OutputDir:    outDir,
		Description:  "Project without docs",
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.NoError(t, err)

	projectDir := filepath.Join(outDir, "no-docs-empty")
	claudeContent, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	require.NoError(t, err)

	content := string(claudeContent)
	assert.Contains(t, content, "# no-docs-empty")
	assert.Contains(t, content, "Project without docs")
	assert.NotContains(t, content, "Supporting Documentation", "should not have doc section when no docs provided")
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	assert.NoError(t, err, "expected file to exist: %s", path)
}
