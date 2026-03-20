package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidationError holds a structured validation failure with the field name
// and a human-readable message suitable for UI display.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationResult aggregates all validation errors for a wizard request.
type ValidationResult struct {
	Errors []ValidationError `json:"errors"`
}

// IsValid returns true if no validation errors were found.
func (vr *ValidationResult) IsValid() bool {
	return len(vr.Errors) == 0
}

// Add appends a validation error.
func (vr *ValidationResult) Add(field, message string) {
	vr.Errors = append(vr.Errors, ValidationError{Field: field, Message: message})
}

// ErrorMap returns errors grouped by field name for easy frontend consumption.
func (vr *ValidationResult) ErrorMap() map[string]string {
	m := make(map[string]string, len(vr.Errors))
	for _, e := range vr.Errors {
		// Keep only the first error per field
		if _, exists := m[e.Field]; !exists {
			m[e.Field] = e.Message
		}
	}
	return m
}

// validProjectName matches alphanumeric, hyphens, underscores, and dots.
// Must start with a letter or number, 1-64 characters.
var validProjectName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

// Validate checks a WizardRequest and returns all validation errors.
// It checks required fields, format constraints, language/framework validity,
// and filesystem conditions (output directory existence, no conflicts).
func Validate(req WizardRequest) ValidationResult {
	var result ValidationResult

	// Project name
	name := strings.TrimSpace(req.ProjectName)
	if name == "" {
		result.Add("project_name", "Project name is required")
	} else if !validProjectName.MatchString(name) {
		result.Add("project_name", "Project name must start with a letter or number and contain only letters, numbers, hyphens, underscores, or dots (max 64 chars)")
	}

	// Language & Framework (not required for empty projects)
	if !req.EmptyProject {
		lang := strings.TrimSpace(req.Language)
		if lang == "" {
			result.Add("language", "Language is required")
		} else if _, ok := FindLanguage(lang); !ok {
			result.Add("language", fmt.Sprintf("Unsupported language: %s", lang))
		}

		fw := strings.TrimSpace(req.Framework)
		if fw == "" {
			result.Add("framework", "Framework is required")
		} else if lang != "" {
			if _, ok := FindFramework(lang, fw); !ok {
				result.Add("framework", fmt.Sprintf("Unsupported framework %q for language %q", fw, lang))
			}
		}
	}

	// Output directory
	outDir := strings.TrimSpace(req.OutputDir)
	if outDir == "" {
		result.Add("output_dir", "Output directory is required")
	} else {
		// Expand ~ to home directory
		expanded := expandHomePath(outDir)

		info, err := os.Stat(expanded)
		if err != nil {
			if os.IsNotExist(err) {
				result.Add("output_dir", "Output directory does not exist")
			} else {
				result.Add("output_dir", fmt.Sprintf("Cannot access output directory: %v", err))
			}
		} else if !info.IsDir() {
			result.Add("output_dir", "Output path is not a directory")
		} else {
			// Check that the target project directory doesn't already exist
			if name != "" {
				projectPath := filepath.Join(expanded, name)
				if _, err := os.Stat(projectPath); err == nil {
					result.Add("project_name", fmt.Sprintf("Directory %q already exists in the output directory", name))
				}
			}
		}
	}

	// Validate doc paths (if provided, files must exist)
	validateDocPath(&result, "doc_prd", req.DocPRD)
	validateDocPath(&result, "doc_uiux", req.DocUIUX)
	validateDocPath(&result, "doc_architecture", req.DocArchitecture)
	validateDocPath(&result, "doc_other", req.DocOther)

	return result
}

// validateDocPath checks that a supporting document is either:
// 1. A file path pointing to an existing readable file, OR
// 2. Direct document content (multi-line text)
// This allows users to either provide file paths or paste content directly.
func validateDocPath(result *ValidationResult, field, path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}

	// Check if this looks like a file path (starts with / or ~/ or ./ or Windows drive letter)
	isFilePath := strings.HasPrefix(path, "/") || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "./") ||
		(len(path) >= 3 && path[1] == ':' && (path[2] == '/' || path[2] == '\\'))

	if isFilePath {
		// Validate as file path
		expanded := expandHomePath(path)
		info, err := os.Stat(expanded)
		if err != nil {
			if os.IsNotExist(err) {
				result.Add(field, "File does not exist")
			} else {
				result.Add(field, fmt.Sprintf("Cannot access file: %v", err))
			}
			return
		}
		if info.IsDir() {
			result.Add(field, "Path is a directory, expected a file")
		}
	}
	// If not a file path, assume it's direct content (markdown, text, etc.) - no validation needed
}

// expandHomePath expands a leading ~ to the user's home directory.
func expandHomePath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
