package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/davydany/ClawIDE/internal/wizard"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// GET /projects/wizard - ShowWizard
// ---------------------------------------------------------------------------

func TestShowWizard_JSON(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	req := httptest.NewRequest(http.MethodGet, "/projects/wizard", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	h.ShowWizard(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	languages, ok := resp["languages"].([]any)
	require.True(t, ok, "response should contain languages array")
	assert.GreaterOrEqual(t, len(languages), 8, "should have at least 8 languages")

	_, ok = resp["projects_dir"]
	assert.True(t, ok, "response should contain projects_dir")
}

// ---------------------------------------------------------------------------
// GET /api/wizard/languages - GetWizardLanguages
// ---------------------------------------------------------------------------

func TestGetWizardLanguages(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/wizard/languages", nil)
	w := httptest.NewRecorder()

	h.GetWizardLanguages(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	languages, ok := resp["languages"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(languages), 8)
}

// ---------------------------------------------------------------------------
// POST /projects/wizard/create - CreateProjectFromWizard
// ---------------------------------------------------------------------------

func TestCreateProjectFromWizard_JSON_Valid(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{
		"project_name": "test-wizard-project",
		"language": "python",
		"framework": "django",
		"description": "A test project from the wizard"
	}`
	req := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateProjectFromWizard(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp["job_id"], "response should contain job_id")

	// Allow async goroutine to finish before TempDir cleanup
	time.Sleep(500 * time.Millisecond)
}

func TestCreateProjectFromWizard_Form_Valid(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	form := url.Values{
		"project_name": {"test-form-project"},
		"language":     {"python"},
		"framework":    {"django"},
		"description":  {"A test project via form"},
	}
	req := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.CreateProjectFromWizard(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp["job_id"])

	// Allow async goroutine to finish before TempDir cleanup
	time.Sleep(500 * time.Millisecond)
}

func TestCreateProjectFromWizard_ValidationFailure_MissingName(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{
		"language": "python",
		"framework": "django"
	}`
	req := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateProjectFromWizard(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	errors, ok := resp["errors"].(map[string]any)
	require.True(t, ok, "response should contain errors map")
	assert.Contains(t, errors, "project_name")
}

func TestCreateProjectFromWizard_ValidationFailure_InvalidName(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{
		"project_name": "-invalid-name",
		"language": "python",
		"framework": "django"
	}`
	req := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateProjectFromWizard(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateProjectFromWizard_ValidationFailure_InvalidLanguage(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{
		"project_name": "test-project",
		"language": "cobol",
		"framework": "whatever"
	}`
	req := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateProjectFromWizard(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	errors := resp["errors"].(map[string]any)
	assert.Contains(t, errors, "language")
}

func TestCreateProjectFromWizard_ValidationFailure_InvalidFramework(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{
		"project_name": "test-project",
		"language": "python",
		"framework": "rails"
	}`
	req := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateProjectFromWizard(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	errors := resp["errors"].(map[string]any)
	assert.Contains(t, errors, "framework")
}

func TestCreateProjectFromWizard_InvalidJSON(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	req := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateProjectFromWizard(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateProjectFromWizard_DefaultsOutputDir(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{
		"project_name": "default-dir-project",
		"language": "python",
		"framework": "django"
	}`
	req := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateProjectFromWizard(w, req)

	// Should succeed because output_dir defaults to h.cfg.ProjectsDir
	assert.Equal(t, http.StatusAccepted, w.Code)

	// Allow async goroutine to finish before TempDir cleanup
	time.Sleep(500 * time.Millisecond)
}

func TestCreateProjectFromWizard_EmptyBody(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	req := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateProjectFromWizard(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GET /projects/wizard/status/{jobID} - GetWizardStatus
// ---------------------------------------------------------------------------

func TestGetWizardStatus_Found(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	// Create a job via the tracker
	wizReq := wizard.WizardRequest{
		ProjectName: "status-test",
		Language:    "python",
		Framework:   "django",
	}
	job := h.wizardJobs.Add(wizReq)

	req := httptest.NewRequest(http.MethodGet, "/projects/wizard/status/"+job.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", job.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.GetWizardStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp wizardStatusResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, job.ID, resp.JobID)
	assert.Equal(t, wizard.JobStatusPending, resp.Status)
	assert.Len(t, resp.Steps, 7) // default steps
}

func TestGetWizardStatus_NotFound(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	req := httptest.NewRequest(http.MethodGet, "/projects/wizard/status/nonexistent-id", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", "nonexistent-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.GetWizardStatus(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetWizardStatus_EmptyJobID(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	req := httptest.NewRequest(http.MethodGet, "/projects/wizard/status/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.GetWizardStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetWizardStatus_CompletedJob(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	wizReq := wizard.WizardRequest{
		ProjectName: "completed-test",
		Language:    "python",
		Framework:   "django",
	}
	job := h.wizardJobs.Add(wizReq)
	job.Complete("/path/to/project")

	req := httptest.NewRequest(http.MethodGet, "/projects/wizard/status/"+job.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", job.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.GetWizardStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp wizardStatusResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, wizard.JobStatusCompleted, resp.Status)
	assert.Equal(t, "/path/to/project", resp.OutputDir)
}

func TestGetWizardStatus_FailedJob(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	wizReq := wizard.WizardRequest{
		ProjectName: "failed-test",
		Language:    "python",
		Framework:   "django",
	}
	job := h.wizardJobs.Add(wizReq)
	job.Fail(assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/projects/wizard/status/"+job.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", job.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.GetWizardStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp wizardStatusResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, wizard.JobStatusFailed, resp.Status)
	assert.NotEmpty(t, resp.Error)
}

// ---------------------------------------------------------------------------
// POST /api/wizard/validate - ValidateWizardField
// ---------------------------------------------------------------------------

func TestValidateWizardField_SingleField_Valid(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{
		"project_name": "valid-name",
		"language": "python",
		"framework": "django"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/wizard/validate?field=project_name", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ValidateWizardField(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp["valid"].(bool))
	assert.Equal(t, "project_name", resp["field"])
}

func TestValidateWizardField_SingleField_Invalid(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{
		"project_name": "-bad-name",
		"language": "python",
		"framework": "django"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/wizard/validate?field=project_name", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ValidateWizardField(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.False(t, resp["valid"].(bool))
	assert.Equal(t, "project_name", resp["field"])
	assert.NotEmpty(t, resp["error"])
}

func TestValidateWizardField_AllFields(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{
		"project_name": "good-name",
		"language": "python",
		"framework": "django"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/wizard/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ValidateWizardField(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp["valid"].(bool))
}

func TestValidateWizardField_AllFields_WithErrors(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/wizard/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ValidateWizardField(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.False(t, resp["valid"].(bool))
	errors := resp["errors"].(map[string]any)
	assert.Contains(t, errors, "project_name")
	assert.Contains(t, errors, "language")
}

func TestValidateWizardField_InvalidJSON(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/wizard/validate", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ValidateWizardField(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestValidateWizardField_DefaultsOutputDir(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{
		"project_name": "test",
		"language": "python",
		"framework": "django"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/wizard/validate?field=output_dir", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ValidateWizardField(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	// Should be valid because output_dir defaults to cfg.ProjectsDir
	assert.True(t, resp["valid"].(bool))
}

// ---------------------------------------------------------------------------
// GET /projects/wizard (HTML) - ShowWizard HTML fallback
// ---------------------------------------------------------------------------

func TestShowWizard_HTML_RendersErrorOnMissingTemplate(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	// No Accept: application/json → takes the HTML branch
	req := httptest.NewRequest(http.MethodGet, "/projects/wizard", nil)
	w := httptest.NewRecorder()

	h.ShowWizard(w, req)

	// The minimal test template FS doesn't have a "wizard" template,
	// so RenderHTMX returns an error and the handler responds 500.
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// GET /api/wizard/scan-dir - ScanProjectsDir
// ---------------------------------------------------------------------------

func TestScanProjectsDir_DefaultDir(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/wizard/scan-dir", nil)
	w := httptest.NewRecorder()

	h.ScanProjectsDir(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp["projects_dir"])
}

func TestScanProjectsDir_WithQueryParam(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/wizard/scan-dir?dir=/tmp", nil)
	w := httptest.NewRecorder()

	h.ScanProjectsDir(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp["projects_dir"])
}

func TestScanProjectsDir_TildeExpansion(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/wizard/scan-dir?dir=~/projects", nil)
	w := httptest.NewRecorder()

	h.ScanProjectsDir(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp["projects_dir"])
}

// ---------------------------------------------------------------------------
// Integration: Create → Status polling
// ---------------------------------------------------------------------------

func TestWizardFlow_CreateThenPollStatus(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	// Step 1: Create a project
	body := `{
		"project_name": "flow-test",
		"language": "python",
		"framework": "django",
		"description": "Integration flow test"
	}`
	createReq := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()

	h.CreateProjectFromWizard(createW, createReq)

	require.Equal(t, http.StatusAccepted, createW.Code)

	var createResp map[string]string
	require.NoError(t, json.NewDecoder(createW.Body).Decode(&createResp))
	jobID := createResp["job_id"]
	require.NotEmpty(t, jobID)

	// Step 2: Poll status (immediate — should be pending or running)
	statusReq := httptest.NewRequest(http.MethodGet, "/projects/wizard/status/"+jobID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", jobID)
	statusReq = statusReq.WithContext(context.WithValue(statusReq.Context(), chi.RouteCtxKey, rctx))
	statusW := httptest.NewRecorder()

	h.GetWizardStatus(statusW, statusReq)

	assert.Equal(t, http.StatusOK, statusW.Code)

	var statusResp wizardStatusResponse
	require.NoError(t, json.NewDecoder(statusW.Body).Decode(&statusResp))
	assert.Equal(t, jobID, statusResp.JobID)
	// Status could be pending, running, or completed depending on timing
	assert.Contains(t, []wizard.JobStatus{
		wizard.JobStatusPending,
		wizard.JobStatusRunning,
		wizard.JobStatusCompleted,
		wizard.JobStatusFailed,
	}, statusResp.Status)

	// Step 3: Wait briefly and poll again for eventual completion
	time.Sleep(3 * time.Second)

	statusReq2 := httptest.NewRequest(http.MethodGet, "/projects/wizard/status/"+jobID, nil)
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("jobID", jobID)
	statusReq2 = statusReq2.WithContext(context.WithValue(statusReq2.Context(), chi.RouteCtxKey, rctx2))
	statusW2 := httptest.NewRecorder()

	h.GetWizardStatus(statusW2, statusReq2)

	var statusResp2 wizardStatusResponse
	require.NoError(t, json.NewDecoder(statusW2.Body).Decode(&statusResp2))
	// After waiting, job should have progressed
	assert.NotEqual(t, wizard.JobStatusPending, statusResp2.Status,
		"job should have started after 3 seconds")
}

// ---------------------------------------------------------------------------
// Empty Project tests
// ---------------------------------------------------------------------------

func TestCreateProjectFromWizard_EmptyProject_Valid(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{
		"project_name": "empty-test-project",
		"empty_project": true,
		"description": "An empty project"
	}`
	req := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateProjectFromWizard(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp["job_id"], "response should contain job_id")

	// Allow async goroutine to finish before TempDir cleanup
	time.Sleep(500 * time.Millisecond)
}

func TestCreateProjectFromWizard_EmptyProject_MissingName(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	body := `{
		"empty_project": true
	}`
	req := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateProjectFromWizard(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	errors, ok := resp["errors"].(map[string]any)
	require.True(t, ok, "response should contain errors map")
	assert.Contains(t, errors, "project_name")
	// Should NOT contain language/framework errors for empty projects
	assert.NotContains(t, errors, "language")
	assert.NotContains(t, errors, "framework")
}

func TestCreateProjectFromWizard_EmptyProject_NoLanguageRequired(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	// Empty project should not require language/framework
	body := `{
		"project_name": "bare-project",
		"empty_project": true
	}`
	req := httptest.NewRequest(http.MethodPost, "/projects/wizard/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateProjectFromWizard(w, req)

	// Should be accepted since empty projects don't need language/framework
	assert.Equal(t, http.StatusAccepted, w.Code)

	time.Sleep(500 * time.Millisecond)
}

func TestGetWizardStatus_EmptyProject_Steps(t *testing.T) {
	h, _ := setupHandlerWithRenderer(t)

	wizReq := wizard.WizardRequest{
		ProjectName:  "empty-status-test",
		EmptyProject: true,
	}
	job := h.wizardJobs.Add(wizReq)

	req := httptest.NewRequest(http.MethodGet, "/projects/wizard/status/"+job.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", job.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.GetWizardStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp wizardStatusResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp.Steps, 5, "empty project should have 5 steps")
}
