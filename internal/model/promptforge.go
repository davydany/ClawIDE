package model

import (
	"fmt"
	"regexp"
	"time"
)

// PromptType enumerates how a prompt body is interpreted.
// "plain" = literal markdown; "jinja" = Nunjucks/Jinja template compiled in the browser.
type PromptType string

const (
	PromptTypePlain PromptType = "plain"
	PromptTypeJinja PromptType = "jinja"
)

// VariableType is the registered set of variable types supported by PromptForge v1.
type VariableType string

const (
	VariableTypeString  VariableType = "string"
	VariableTypeText    VariableType = "text"
	VariableTypeNumber  VariableType = "number"
	VariableTypeBoolean VariableType = "boolean"
	VariableTypeSelect  VariableType = "select"
	VariableTypeDate    VariableType = "date"
)

// Variable declares one template input for a Jinja-type prompt.
type Variable struct {
	Name     string       `json:"name" yaml:"name"`
	Type     VariableType `json:"type" yaml:"type"`
	Label    string       `json:"label,omitempty" yaml:"label,omitempty"`
	Default  string       `json:"default,omitempty" yaml:"default,omitempty"`
	Options  []string     `json:"options,omitempty" yaml:"options,omitempty"`
	Required bool         `json:"required,omitempty" yaml:"required,omitempty"`
}

// Prompt is a single markdown/Jinja template stored on disk as
// <Title>.md with YAML frontmatter.
type Prompt struct {
	ID        string     `json:"id" yaml:"id"`
	FolderID  string     `json:"folder_id,omitempty" yaml:"folder_id,omitempty"`
	Title     string     `json:"title" yaml:"title"`
	Type      PromptType `json:"type" yaml:"type"`
	Variables []Variable `json:"variables,omitempty" yaml:"variables,omitempty"`
	Content   string     `json:"content" yaml:"-"` // stored as markdown body, not in frontmatter
	CreatedAt time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"updated_at"`
}

// CompiledVersion is a single historical compilation of a prompt, stored at
// <Title>.compiled/<version-id>.md with YAML frontmatter.
type CompiledVersion struct {
	ID             string                 `json:"id" yaml:"id"`
	PromptID       string                 `json:"prompt_id" yaml:"prompt_id"`
	Title          string                 `json:"title" yaml:"title"`
	VariableValues map[string]interface{} `json:"variable_values,omitempty" yaml:"variable_values,omitempty"`
	Content        string                 `json:"content" yaml:"-"`
	CompiledAt     time.Time              `json:"compiled_at" yaml:"compiled_at"`
}

// PromptFilenameRegex constrains what can appear in prompt filenames on disk.
// Matches the existing note/folder convention.
var PromptFilenameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// ValidatePromptTitle checks that a prompt title is safe to use as a filename.
func ValidatePromptTitle(title string) error {
	if title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if title == "." || title == ".." {
		return fmt.Errorf("title cannot be '.' or '..'")
	}
	if len(title) > 255 {
		return fmt.Errorf("title exceeds maximum length of 255 characters")
	}
	if !PromptFilenameRegex.MatchString(title) {
		return fmt.Errorf("title contains invalid characters; only letters, numbers, dots, hyphens, and underscores are allowed")
	}
	return nil
}

// ValidateVersionTitle is looser than prompt titles because version titles are
// user-facing and not filenames (versions are stored under UUIDs).
func ValidateVersionTitle(title string) error {
	if title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if len(title) > 255 {
		return fmt.Errorf("title exceeds maximum length of 255 characters")
	}
	return nil
}

var validVariableNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidateVariableName ensures a variable name is a valid identifier (safe
// for use in Jinja/Nunjucks template variable expressions).
func ValidateVariableName(name string) error {
	if name == "" {
		return fmt.Errorf("variable name cannot be empty")
	}
	if !validVariableNameRegex.MatchString(name) {
		return fmt.Errorf("variable name must start with a letter or underscore and contain only letters, numbers, and underscores")
	}
	return nil
}

// ValidateVariableType rejects types outside the registered set.
func ValidateVariableType(t VariableType) error {
	switch t {
	case VariableTypeString, VariableTypeText, VariableTypeNumber,
		VariableTypeBoolean, VariableTypeSelect, VariableTypeDate:
		return nil
	}
	return fmt.Errorf("invalid variable type: %q", t)
}

// ValidatePromptType rejects prompt types outside {plain, jinja}.
func ValidatePromptType(t PromptType) error {
	switch t {
	case PromptTypePlain, PromptTypeJinja:
		return nil
	}
	return fmt.Errorf("invalid prompt type: %q", t)
}

// ValidateVariables validates an entire variable list: unique names, valid
// names, valid types, and options required for type=select.
func ValidateVariables(vars []Variable) error {
	seen := make(map[string]bool, len(vars))
	for i, v := range vars {
		if err := ValidateVariableName(v.Name); err != nil {
			return fmt.Errorf("variable[%d]: %w", i, err)
		}
		if seen[v.Name] {
			return fmt.Errorf("variable[%d]: duplicate name %q", i, v.Name)
		}
		seen[v.Name] = true
		if err := ValidateVariableType(v.Type); err != nil {
			return fmt.Errorf("variable[%d]: %w", i, err)
		}
		if v.Type == VariableTypeSelect && len(v.Options) == 0 {
			return fmt.Errorf("variable[%d] %q: select variables require at least one option", i, v.Name)
		}
	}
	return nil
}
