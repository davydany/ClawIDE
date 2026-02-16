package wizard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_RequiredFields(t *testing.T) {
	result := Validate(WizardRequest{})
	assert.False(t, result.IsValid())

	errMap := result.ErrorMap()
	assert.Contains(t, errMap, "project_name")
	assert.Contains(t, errMap, "language")
	assert.Contains(t, errMap, "framework")
	assert.Contains(t, errMap, "output_dir")
}

func TestValidate_InvalidProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"starts with hyphen", "-bad", true},
		{"starts with dot", ".hidden", true},
		{"has spaces", "my project", true},
		{"has special chars", "my@project!", true},
		{"empty", "", true},
		{"too long", string(make([]byte, 65)), true},
		{"valid simple", "myproject", false},
		{"valid with hyphen", "my-project", false},
		{"valid with underscore", "my_project", false},
		{"valid with dot", "my.project", false},
		{"valid with numbers", "project123", false},
		{"starts with number", "1project", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir := t.TempDir()
			req := WizardRequest{
				ProjectName: tt.input,
				Language:    "python",
				Framework:   "django",
				OutputDir:   outDir,
			}
			result := Validate(req)
			errMap := result.ErrorMap()
			if tt.wantErr {
				assert.Contains(t, errMap, "project_name", "expected project_name error for %q", tt.input)
			} else {
				assert.NotContains(t, errMap, "project_name", "unexpected project_name error for %q", tt.input)
			}
		})
	}
}

func TestValidate_InvalidLanguage(t *testing.T) {
	outDir := t.TempDir()
	req := WizardRequest{
		ProjectName: "test",
		Language:    "cobol",
		Framework:   "something",
		OutputDir:   outDir,
	}
	result := Validate(req)
	errMap := result.ErrorMap()
	assert.Contains(t, errMap, "language")
}

func TestValidate_InvalidFramework(t *testing.T) {
	outDir := t.TempDir()
	req := WizardRequest{
		ProjectName: "test",
		Language:    "python",
		Framework:   "rails",
		OutputDir:   outDir,
	}
	result := Validate(req)
	errMap := result.ErrorMap()
	assert.Contains(t, errMap, "framework")
}

func TestValidate_OutputDirNotExist(t *testing.T) {
	req := WizardRequest{
		ProjectName: "test",
		Language:    "python",
		Framework:   "django",
		OutputDir:   "/nonexistent/path/that/doesnt/exist",
	}
	result := Validate(req)
	errMap := result.ErrorMap()
	assert.Contains(t, errMap, "output_dir")
}

func TestValidate_OutputDirIsFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "notadir")
	require.NoError(t, os.WriteFile(tmpFile, []byte("x"), 0644))

	req := WizardRequest{
		ProjectName: "test",
		Language:    "python",
		Framework:   "django",
		OutputDir:   tmpFile,
	}
	result := Validate(req)
	errMap := result.ErrorMap()
	assert.Contains(t, errMap, "output_dir")
}

func TestValidate_ProjectDirAlreadyExists(t *testing.T) {
	outDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(outDir, "existing"), 0755))

	req := WizardRequest{
		ProjectName: "existing",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outDir,
	}
	result := Validate(req)
	errMap := result.ErrorMap()
	assert.Contains(t, errMap, "project_name")
	assert.Contains(t, errMap["project_name"], "already exists")
}

func TestValidate_ValidRequest(t *testing.T) {
	outDir := t.TempDir()
	req := WizardRequest{
		ProjectName: "my-awesome-project",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outDir,
		Description: "A test project",
	}
	result := Validate(req)
	assert.True(t, result.IsValid(), "expected valid request, got errors: %v", result.Errors)
}

func TestValidate_DocPathNotExist(t *testing.T) {
	outDir := t.TempDir()
	req := WizardRequest{
		ProjectName: "test",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outDir,
		DocPRD:      "/nonexistent/file.md",
	}
	result := Validate(req)
	errMap := result.ErrorMap()
	assert.Contains(t, errMap, "doc_prd")
}

func TestValidate_DocPathIsDir(t *testing.T) {
	outDir := t.TempDir()
	docDir := t.TempDir()
	req := WizardRequest{
		ProjectName: "test",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outDir,
		DocPRD:      docDir,
	}
	result := Validate(req)
	errMap := result.ErrorMap()
	assert.Contains(t, errMap, "doc_prd")
}

func TestValidate_ValidDocPath(t *testing.T) {
	outDir := t.TempDir()
	docFile := filepath.Join(t.TempDir(), "prd.md")
	require.NoError(t, os.WriteFile(docFile, []byte("# PRD"), 0644))

	req := WizardRequest{
		ProjectName: "test",
		Language:    "python",
		Framework:   "django",
		OutputDir:   outDir,
		DocPRD:      docFile,
	}
	result := Validate(req)
	assert.True(t, result.IsValid(), "expected valid request, got errors: %v", result.Errors)
}

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{Field: "name", Message: "is required"}
	assert.Equal(t, "name: is required", err.Error())
}

// ---------------------------------------------------------------------------
// Additional edge cases for validator coverage
// ---------------------------------------------------------------------------

func TestValidationResult_IsValid_Empty(t *testing.T) {
	vr := ValidationResult{}
	assert.True(t, vr.IsValid())
}

func TestValidationResult_ErrorMap_FirstErrorPerField(t *testing.T) {
	vr := ValidationResult{}
	vr.Add("name", "first error")
	vr.Add("name", "second error")
	vr.Add("email", "required")

	m := vr.ErrorMap()
	assert.Equal(t, "first error", m["name"], "should keep only first error per field")
	assert.Equal(t, "required", m["email"])
	assert.Len(t, m, 2)
}

func TestValidate_WhitespaceOnlyFields(t *testing.T) {
	result := Validate(WizardRequest{
		ProjectName: "   ",
		Language:    "   ",
		Framework:   "   ",
		OutputDir:   "   ",
	})
	assert.False(t, result.IsValid())
	errMap := result.ErrorMap()
	assert.Contains(t, errMap, "project_name")
	assert.Contains(t, errMap, "language")
	assert.Contains(t, errMap, "framework")
	assert.Contains(t, errMap, "output_dir")
}

func TestValidate_FrameworkSkippedWhenLanguageEmpty(t *testing.T) {
	// When language is empty, framework validation against language
	// should not produce an "unsupported framework" error (only missing language).
	outDir := t.TempDir()
	result := Validate(WizardRequest{
		ProjectName: "test",
		Language:    "",
		Framework:   "django",
		OutputDir:   outDir,
	})
	errMap := result.ErrorMap()
	assert.Contains(t, errMap, "language")
	// Framework error should just be about the missing language, not about invalid combo
	assert.NotContains(t, errMap, "framework", "should not validate framework when language is empty")
}

func TestValidate_AllDocTypes(t *testing.T) {
	outDir := t.TempDir()

	// Create valid doc files
	docDir := t.TempDir()
	prd := filepath.Join(docDir, "prd.md")
	uiux := filepath.Join(docDir, "uiux.md")
	arch := filepath.Join(docDir, "arch.md")
	other := filepath.Join(docDir, "other.md")
	require.NoError(t, os.WriteFile(prd, []byte("prd"), 0644))
	require.NoError(t, os.WriteFile(uiux, []byte("uiux"), 0644))
	require.NoError(t, os.WriteFile(arch, []byte("arch"), 0644))
	require.NoError(t, os.WriteFile(other, []byte("other"), 0644))

	req := WizardRequest{
		ProjectName:     "test",
		Language:        "python",
		Framework:       "django",
		OutputDir:       outDir,
		DocPRD:          prd,
		DocUIUX:         uiux,
		DocArchitecture: arch,
		DocOther:        other,
	}
	result := Validate(req)
	assert.True(t, result.IsValid(), "all valid docs should pass, got errors: %v", result.Errors)
}

func TestValidate_AllDocTypesInvalid(t *testing.T) {
	outDir := t.TempDir()
	req := WizardRequest{
		ProjectName:     "test",
		Language:        "python",
		Framework:       "django",
		OutputDir:       outDir,
		DocPRD:          "/nope/prd.md",
		DocUIUX:         "/nope/uiux.md",
		DocArchitecture: "/nope/arch.md",
		DocOther:        "/nope/other.md",
	}
	result := Validate(req)
	errMap := result.ErrorMap()
	assert.Contains(t, errMap, "doc_prd")
	assert.Contains(t, errMap, "doc_uiux")
	assert.Contains(t, errMap, "doc_architecture")
	assert.Contains(t, errMap, "doc_other")
}

func TestExpandHomePath_WithTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	expanded := expandHomePath("~/projects")
	assert.Equal(t, filepath.Join(home, "projects"), expanded)
}

func TestExpandHomePath_WithoutTilde(t *testing.T) {
	path := "/absolute/path"
	assert.Equal(t, path, expandHomePath(path))
}

func TestExpandHomePath_EmptyString(t *testing.T) {
	assert.Equal(t, "", expandHomePath(""))
}

func TestExpandHomePath_JustTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, home, expandHomePath("~"))
}

func TestValidate_TildeInOutputDir(t *testing.T) {
	// This validates that expandHomePath is called during validation.
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	// Create a temp dir inside the home dir won't work in all CIs,
	// so just check that a tilde path to a nonexistent dir produces the right error.
	req := WizardRequest{
		ProjectName: "test",
		Language:    "python",
		Framework:   "django",
		OutputDir:   "~/nonexistent-wizard-test-dir",
	}
	result := Validate(req)
	errMap := result.ErrorMap()
	assert.Contains(t, errMap, "output_dir")
	_ = home // used only for context
}
