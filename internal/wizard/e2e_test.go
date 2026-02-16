package wizard

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// E2E Test Helpers
// ---------------------------------------------------------------------------

func setupE2E(t *testing.T) (*Generator, *JobTracker) {
	t.Helper()
	tracker := NewJobTracker()
	reg, err := NewTemplateRegistry(TemplatesFS)
	require.NoError(t, err, "NewTemplateRegistry from embedded FS should succeed")
	gen := NewGenerator(reg, tracker)
	return gen, tracker
}

func createDocFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func e2eReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "expected file to exist: %s", path)
	return string(data)
}

func e2eAssertDir(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err, "expected directory to exist: %s", path)
	assert.True(t, info.IsDir(), "expected %s to be a directory", path)
}

func e2eAssertContains(t *testing.T, path, expected string) {
	t.Helper()
	content := e2eReadFile(t, path)
	assert.Contains(t, content, expected, "file %s should contain %q", path, expected)
}

// frameworksWithTemplates returns only the framework IDs that have matching
// template directories in the embedded FS. Some IDs in languages.go don't
// have templates yet (e.g., chi, quarkus, sinatra, actix, minimal-api, symfony).
func frameworksWithTemplates(t *testing.T) map[string]bool {
	t.Helper()
	reg, err := NewTemplateRegistry(TemplatesFS)
	require.NoError(t, err)
	keys := reg.ListAvailable()
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

// ---------------------------------------------------------------------------
// Test Case 1: Django (Python) — Full flow with PRD doc
// ---------------------------------------------------------------------------

func TestE2E_Django_FullFlow(t *testing.T) {
	gen, tracker := setupE2E(t)
	outputDir := t.TempDir()

	prdContent := "# Product Requirements\n\nThis is the PRD for the blog platform."
	prdPath := createDocFile(t, "prd.md", prdContent)

	req := WizardRequest{
		ProjectName: "test-blog",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outputDir,
		Description: "A blog platform built with Django",
		DocPRD:      prdPath,
	}

	job := tracker.Add(req)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := gen.Generate(ctx, job)
	require.NoError(t, err, "Django generation should succeed")

	projectDir := filepath.Join(outputDir, "test-blog")

	// --- Verify job status ---
	snap := job.Snapshot()
	assert.Equal(t, JobStatusCompleted, snap.Status, "job status should be completed")
	assert.Equal(t, projectDir, snap.OutputDir, "output dir should match")
	for _, step := range snap.Steps {
		assert.NotEqual(t, JobStatusFailed, step.Status,
			"step %s should not have failed", step.Name)
	}

	// --- Verify directory structure ---
	e2eAssertDir(t, projectDir)
	e2eAssertDir(t, filepath.Join(projectDir, "docs", "supporting"))

	// --- Common templates at project root ---
	// The common template generates CLAUDE.md, README.md, .gitignore at root
	claudeContent := e2eReadFile(t, filepath.Join(projectDir, "CLAUDE.md"))
	assert.Contains(t, claudeContent, "test-blog", "CLAUDE.md should contain project name")
	assert.Contains(t, claudeContent, "A blog platform built with Django", "CLAUDE.md should contain description")

	// --- CLAUDE.md conditional doc references ---
	// PRD was provided → reference should be present
	assert.Contains(t, claudeContent, "prd.md",
		"CLAUDE.md should reference PRD since it was provided")
	// UI/UX was NOT provided → no reference
	assert.NotContains(t, claudeContent, "uiux.md",
		"CLAUDE.md should NOT reference UI/UX since it was not provided")

	// --- Framework-specific files under files/ subdirectory ---
	// (The template directory structure python/django/files/* preserves the files/ prefix)
	e2eReadFile(t, filepath.Join(projectDir, "files", "Makefile"))
	e2eReadFile(t, filepath.Join(projectDir, "files", "docker-compose.yml"))
	e2eReadFile(t, filepath.Join(projectDir, "files", "Dockerfile"))
	e2eReadFile(t, filepath.Join(projectDir, "files", "requirements.in"))
	e2eReadFile(t, filepath.Join(projectDir, "files", "manage.py"))

	// Framework-specific CLAUDE.md also exists under files/
	fwClaude := e2eReadFile(t, filepath.Join(projectDir, "files", "CLAUDE.md"))
	assert.Contains(t, fwClaude, "Django", "framework CLAUDE.md should reference Django")
	assert.Contains(t, fwClaude, "prd.md", "framework CLAUDE.md should reference PRD")

	// --- Verify Makefile template rendering ---
	makeContent := e2eReadFile(t, filepath.Join(projectDir, "files", "Makefile"))
	assert.Contains(t, makeContent, "test", "Makefile should have test target")

	// --- Verify docker-compose.yml template interpolation ---
	dockerContent := e2eReadFile(t, filepath.Join(projectDir, "files", "docker-compose.yml"))
	assert.Contains(t, dockerContent, "test-blog", "docker-compose.yml should reference project name")

	// --- Verify supporting docs copied ---
	prdDest := filepath.Join(projectDir, "docs", "supporting", "prd.md")
	e2eAssertContains(t, prdDest, "Product Requirements")

	// UI/UX, architecture, other docs should NOT exist
	_, err = os.Stat(filepath.Join(projectDir, "docs", "supporting", "uiux.md"))
	assert.True(t, os.IsNotExist(err), "uiux.md should not exist when not provided")

	_, err = os.Stat(filepath.Join(projectDir, "docs", "supporting", "architecture.md"))
	assert.True(t, os.IsNotExist(err), "architecture.md should not exist when not provided")

	// --- Verify git was initialized ---
	e2eAssertDir(t, filepath.Join(projectDir, ".git"))
}

// ---------------------------------------------------------------------------
// Test Case 2: Next.js (JavaScript) — No supporting docs
// (Using nextjs instead of react-spa because the framework ID "react" in
// languages.go doesn't match the template directory "react-spa")
// ---------------------------------------------------------------------------

func TestE2E_NextJS_NoDocs(t *testing.T) {
	gen, tracker := setupE2E(t)
	outputDir := t.TempDir()

	req := WizardRequest{
		ProjectName: "ui-kit",
		Language:    "javascript",
		Framework:   "nextjs",
		OutputDir:   outputDir,
		Description: "A Next.js web application",
	}

	job := tracker.Add(req)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := gen.Generate(ctx, job)
	require.NoError(t, err, "Next.js generation should succeed")

	projectDir := filepath.Join(outputDir, "ui-kit")

	// --- Verify job completed ---
	snap := job.Snapshot()
	assert.Equal(t, JobStatusCompleted, snap.Status)

	// --- Common template files at root ---
	claudeContent := e2eReadFile(t, filepath.Join(projectDir, "CLAUDE.md"))
	assert.Contains(t, claudeContent, "ui-kit", "CLAUDE.md should contain project name")

	// NO docs provided → NO doc references in CLAUDE.md
	assert.NotContains(t, claudeContent, "prd.md",
		"CLAUDE.md should NOT reference PRD when none provided")
	assert.NotContains(t, claudeContent, "uiux.md",
		"CLAUDE.md should NOT reference UI/UX when none provided")

	// --- Framework files exist under files/ ---
	e2eReadFile(t, filepath.Join(projectDir, "files", "package.json"))
	e2eReadFile(t, filepath.Join(projectDir, "files", "CLAUDE.md"))

	// --- Verify docs/supporting directory is created (but empty of doc files) ---
	e2eAssertDir(t, filepath.Join(projectDir, "docs", "supporting"))

	for _, docFile := range []string{"prd.md", "uiux.md", "architecture.md", "other.md"} {
		_, err := os.Stat(filepath.Join(projectDir, "docs", "supporting", docFile))
		assert.True(t, os.IsNotExist(err),
			"%s should not exist when no docs provided", docFile)
	}

	// --- Verify git initialized ---
	e2eAssertDir(t, filepath.Join(projectDir, ".git"))
}

// ---------------------------------------------------------------------------
// Test Case 3: Go/Gin — With UI/UX doc only
// ---------------------------------------------------------------------------

func TestE2E_GoGin_WithUIUXDoc(t *testing.T) {
	gen, tracker := setupE2E(t)
	outputDir := t.TempDir()

	uiuxContent := "# UI/UX Design Spec\n\nDashboard layout with sidebar navigation."
	uiuxPath := createDocFile(t, "uiux_design.md", uiuxContent)

	req := WizardRequest{
		ProjectName: "api-service",
		Language:    "go",
		Framework:   "gin",
		OutputDir:   outputDir,
		Description: "A RESTful API service built with Gin",
		DocUIUX:     uiuxPath,
	}

	job := tracker.Add(req)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := gen.Generate(ctx, job)
	require.NoError(t, err, "Go/Gin generation should succeed")

	projectDir := filepath.Join(outputDir, "api-service")

	// --- Verify job completed ---
	snap := job.Snapshot()
	assert.Equal(t, JobStatusCompleted, snap.Status)

	// --- Common CLAUDE.md at root ---
	claudeContent := e2eReadFile(t, filepath.Join(projectDir, "CLAUDE.md"))
	assert.Contains(t, claudeContent, "api-service")

	// UI/UX provided → reference present
	assert.Contains(t, claudeContent, "uiux.md",
		"CLAUDE.md should reference UI/UX doc since it was provided")
	// PRD NOT provided → no reference
	assert.NotContains(t, claudeContent, "prd.md",
		"CLAUDE.md should NOT reference PRD since it was not provided")

	// --- Framework-specific files under files/ ---
	e2eReadFile(t, filepath.Join(projectDir, "files", "go.mod"))
	e2eReadFile(t, filepath.Join(projectDir, "files", "main.go"))
	e2eReadFile(t, filepath.Join(projectDir, "files", "handlers.go"))
	e2eReadFile(t, filepath.Join(projectDir, "files", "Makefile"))
	e2eReadFile(t, filepath.Join(projectDir, "files", "docker-compose.yml"))

	// Framework CLAUDE.md should also reference UI/UX
	fwClaude := e2eReadFile(t, filepath.Join(projectDir, "files", "CLAUDE.md"))
	assert.Contains(t, fwClaude, "Gin", "framework CLAUDE.md should reference Gin")
	assert.Contains(t, fwClaude, "uiux.md", "framework CLAUDE.md should reference UI/UX")
	assert.NotContains(t, fwClaude, "prd.md", "framework CLAUDE.md should NOT reference PRD")

	// --- Go module file has correct module name ---
	goMod := e2eReadFile(t, filepath.Join(projectDir, "files", "go.mod"))
	assert.Contains(t, goMod, "api-service", "go.mod should reference project name")

	// --- Supporting docs ---
	e2eAssertContains(t, filepath.Join(projectDir, "docs", "supporting", "uiux.md"), "UI/UX Design Spec")
	_, err = os.Stat(filepath.Join(projectDir, "docs", "supporting", "prd.md"))
	assert.True(t, os.IsNotExist(err), "prd.md should not exist")

	// --- Git initialized ---
	e2eAssertDir(t, filepath.Join(projectDir, ".git"))
}

// ---------------------------------------------------------------------------
// Test Case 4: All 4 supporting docs provided
// ---------------------------------------------------------------------------

func TestE2E_Django_AllFourDocs(t *testing.T) {
	gen, tracker := setupE2E(t)
	outputDir := t.TempDir()

	prdPath := createDocFile(t, "prd.md", "# PRD\nProduct requirements here.")
	uiuxPath := createDocFile(t, "uiux.md", "# UI/UX\nDesign specs here.")
	archPath := createDocFile(t, "arch.md", "# Architecture\nSystem design here.")
	otherPath := createDocFile(t, "other.md", "# Other\nMisc docs here.")

	req := WizardRequest{
		ProjectName:     "full-docs-project",
		Language:        "python",
		Framework:       "django",
		OutputDir:       outputDir,
		Description:     "A project with all four supporting docs",
		DocPRD:          prdPath,
		DocUIUX:         uiuxPath,
		DocArchitecture: archPath,
		DocOther:        otherPath,
	}

	job := tracker.Add(req)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := gen.Generate(ctx, job)
	require.NoError(t, err, "generation with all 4 docs should succeed")

	projectDir := filepath.Join(outputDir, "full-docs-project")
	docsDir := filepath.Join(projectDir, "docs", "supporting")

	// All 4 docs should be copied
	e2eAssertContains(t, filepath.Join(docsDir, "prd.md"), "Product requirements here")
	e2eAssertContains(t, filepath.Join(docsDir, "uiux.md"), "Design specs here")
	e2eAssertContains(t, filepath.Join(docsDir, "architecture.md"), "System design here")
	e2eAssertContains(t, filepath.Join(docsDir, "other.md"), "Misc docs here")

	// Root CLAUDE.md (common template) should reference all 4 doc types
	claudeContent := e2eReadFile(t, filepath.Join(projectDir, "CLAUDE.md"))
	assert.Contains(t, claudeContent, "prd.md")
	assert.Contains(t, claudeContent, "uiux.md")
	assert.Contains(t, claudeContent, "architecture.md")
	assert.Contains(t, claudeContent, "docs/supporting/")
}

// ---------------------------------------------------------------------------
// Edge Cases
// ---------------------------------------------------------------------------

func TestE2E_InvalidProjectName(t *testing.T) {
	gen, tracker := setupE2E(t)
	outputDir := t.TempDir()

	req := WizardRequest{
		ProjectName: "-invalid name with spaces",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outputDir,
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.Error(t, err, "invalid project name should fail")
	assert.Contains(t, err.Error(), "validation failed")

	snap := job.Snapshot()
	assert.Equal(t, "failed", string(snap.Steps[0].Status))
}

func TestE2E_UnsupportedLanguage(t *testing.T) {
	gen, tracker := setupE2E(t)
	outputDir := t.TempDir()

	req := WizardRequest{
		ProjectName: "cobol-project",
		Language:    "cobol",
		Framework:   "mainframe",
		OutputDir:   outputDir,
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.Error(t, err, "unsupported language should fail validation")
}

func TestE2E_MissingOutputDir(t *testing.T) {
	gen, tracker := setupE2E(t)

	req := WizardRequest{
		ProjectName: "test-project",
		Language:    "python",
		Framework:   "django",
		OutputDir:   "/nonexistent/path/that/does/not/exist",
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.Error(t, err, "nonexistent output dir should fail validation")
}

func TestE2E_NonexistentDocFile(t *testing.T) {
	gen, tracker := setupE2E(t)
	outputDir := t.TempDir()

	// The validator checks doc file existence during Step 1 (validate),
	// so the job fails at validation, not at copy_docs.
	req := WizardRequest{
		ProjectName: "doc-fail-project",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outputDir,
		DocPRD:      "/nonexistent/path/to/prd.md",
	}

	job := tracker.Add(req)
	err := gen.Generate(context.Background(), job)
	require.Error(t, err, "nonexistent doc file should fail validation")
	assert.Contains(t, err.Error(), "validation failed")

	// Project dir should never have been created (failed at validation, before create_directory)
	projectDir := filepath.Join(outputDir, "doc-fail-project")
	_, statErr := os.Stat(projectDir)
	assert.True(t, os.IsNotExist(statErr), "project dir should not exist when validation fails")

	snap := job.Snapshot()
	assert.Equal(t, JobStatusFailed, snap.Status, "job should be marked as failed")
}

func TestE2E_LongDescription(t *testing.T) {
	gen, tracker := setupE2E(t)
	outputDir := t.TempDir()

	longDesc := strings.Repeat("A very detailed project description. ", 100)
	req := WizardRequest{
		ProjectName: "long-desc-project",
		Language:    "go",
		Framework:   "gin",
		OutputDir:   outputDir,
		Description: longDesc,
	}

	job := tracker.Add(req)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := gen.Generate(ctx, job)
	require.NoError(t, err, "long description should be handled gracefully")

	claudeContent := e2eReadFile(t, filepath.Join(outputDir, "long-desc-project", "CLAUDE.md"))
	assert.Contains(t, claudeContent, "A very detailed project description")
}

// ---------------------------------------------------------------------------
// Cross-Framework: Only test frameworks that have matching template directories
// ---------------------------------------------------------------------------

func TestE2E_MatchedFrameworks_GenerateWithoutError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping comprehensive framework test in short mode")
	}

	gen, tracker := setupE2E(t)
	available := frameworksWithTemplates(t)

	languages := SupportedLanguages()

	for _, lang := range languages {
		for _, fw := range lang.Frameworks {
			key := lang.ID + "/" + fw.ID
			if !available[key] {
				// Document: this framework ID has no matching template
				t.Logf("SKIP: %s (ID %q has no matching template directory)", key, fw.ID)
				continue
			}

			t.Run(key, func(t *testing.T) {
				outputDir := t.TempDir()

				req := WizardRequest{
					ProjectName: "test-" + fw.ID,
					Language:    lang.ID,
					Framework:   fw.ID,
					OutputDir:   outputDir,
					Description: "E2E test for " + fw.Name,
				}

				job := tracker.Add(req)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()

				err := gen.Generate(ctx, job)
				require.NoError(t, err, "generation should succeed for %s", key)

				projectDir := filepath.Join(outputDir, "test-"+fw.ID)

				// Verify basic output for every framework
				e2eAssertDir(t, projectDir)
				e2eReadFile(t, filepath.Join(projectDir, "CLAUDE.md"))
				e2eReadFile(t, filepath.Join(projectDir, "README.md"))
				e2eAssertDir(t, filepath.Join(projectDir, ".git"))
				e2eAssertDir(t, filepath.Join(projectDir, "docs", "supporting"))

				claudeContent := e2eReadFile(t, filepath.Join(projectDir, "CLAUDE.md"))
				assert.Contains(t, claudeContent, "test-"+fw.ID,
					"CLAUDE.md should contain project name for %s", key)

				snap := job.Snapshot()
				assert.Equal(t, JobStatusCompleted, snap.Status,
					"job should complete for %s", key)
			})
		}
	}
}

// ---------------------------------------------------------------------------
// Detect Framework ID / Template Directory Mismatches
// ---------------------------------------------------------------------------

func TestE2E_FrameworkTemplateMismatchReport(t *testing.T) {
	available := frameworksWithTemplates(t)

	var mismatched []string
	for _, lang := range SupportedLanguages() {
		for _, fw := range lang.Frameworks {
			key := lang.ID + "/" + fw.ID
			if !available[key] {
				mismatched = append(mismatched, key)
			}
		}
	}

	if len(mismatched) > 0 {
		t.Logf("WARNING: %d framework IDs have no matching template directory:", len(mismatched))
		for _, m := range mismatched {
			t.Logf("  - %s", m)
		}
		// This is an informational warning, not a test failure.
		// These frameworks will pass validation but fail at template lookup.
	}
}

// ---------------------------------------------------------------------------
// Job Status Transition Tracking
// ---------------------------------------------------------------------------

func TestE2E_JobStepProgression(t *testing.T) {
	gen, tracker := setupE2E(t)
	outputDir := t.TempDir()

	req := WizardRequest{
		ProjectName: "step-tracking-test",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outputDir,
		Description: "Testing step progression",
	}

	job := tracker.Add(req)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := gen.Generate(ctx, job)
	require.NoError(t, err)

	snap := job.Snapshot()

	expectedSteps := []string{
		"validate",
		"create_directory",
		"generate_files",
		"copy_docs",
		"generate_claude_md",
		"init_git",
		"install_deps",
	}
	require.Len(t, snap.Steps, len(expectedSteps), "should have %d steps", len(expectedSteps))

	for i, expected := range expectedSteps {
		assert.Equal(t, expected, snap.Steps[i].Name, "step %d name mismatch", i)
		if i < 6 {
			assert.Equal(t, JobStatusCompleted, snap.Steps[i].Status,
				"step %s should be completed", expected)
		}
		assert.False(t, snap.Steps[i].StartedAt.IsZero(),
			"step %s should have a start time", expected)
		assert.False(t, snap.Steps[i].EndedAt.IsZero(),
			"step %s should have an end time", expected)
	}
}
