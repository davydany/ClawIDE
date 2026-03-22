package skills

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents a Claude Code skill parsed from a SKILL.md file.
type Skill struct {
	Name                   string `json:"name" yaml:"name"`
	Description            string `json:"description" yaml:"description"`
	Version                string `json:"version,omitempty" yaml:"version,omitempty"`
	ArgumentHint           string `json:"argument_hint,omitempty" yaml:"argument-hint,omitempty"`
	DisableModelInvocation bool   `json:"disable_model_invocation" yaml:"disable-model-invocation,omitempty"`
	UserInvocable          *bool  `json:"user_invocable" yaml:"user-invocable,omitempty"`
	AllowedTools           string `json:"allowed_tools,omitempty" yaml:"allowed-tools,omitempty"`
	Model                  string `json:"model,omitempty" yaml:"model,omitempty"`
	Effort                 string `json:"effort,omitempty" yaml:"effort,omitempty"`
	Context                string `json:"context,omitempty" yaml:"context,omitempty"`
	Agent                  string `json:"agent,omitempty" yaml:"agent,omitempty"`
	Homepage               string `json:"homepage,omitempty" yaml:"homepage,omitempty"`
	Content                string `json:"content" yaml:"-"`
	Scope                  string `json:"scope" yaml:"-"`
	DirName                string `json:"dir_name" yaml:"-"`
}

// IsUserInvocable returns true if the skill is user-invocable (default: true).
func (s *Skill) IsUserInvocable() bool {
	if s.UserInvocable == nil {
		return true
	}
	return *s.UserInvocable
}

// GlobalSkillsDir returns the path to the global skills directory.
func GlobalSkillsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "skills")
}

// ProjectSkillsDir returns the path to the project-level skills directory.
func ProjectSkillsDir(projectPath string) string {
	return filepath.Join(projectPath, ".claude", "skills")
}

// ListSkills scans both global and project skill directories and returns all skills.
func ListSkills(globalDir, projectDir string) ([]Skill, error) {
	var all []Skill

	if globalDir != "" {
		global, err := scanDir(globalDir, "global")
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("scanning global skills: %w", err)
		}
		all = append(all, global...)
	}

	if projectDir != "" {
		project, err := scanDir(projectDir, "project")
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("scanning project skills: %w", err)
		}
		all = append(all, project...)
	}

	return all, nil
}

// GetSkill reads and parses a single skill from a base directory.
func GetSkill(baseDir, dirName string) (*Skill, error) {
	skillPath := filepath.Join(baseDir, dirName, "SKILL.md")
	return ParseSkillFile(skillPath)
}

// CreateSkill creates a new skill directory and SKILL.md file.
func CreateSkill(baseDir string, skill Skill) error {
	dirName := skill.DirName
	if dirName == "" {
		dirName = sanitizeDirName(skill.Name)
	}

	skillDir := filepath.Join(baseDir, dirName)
	if _, err := os.Stat(skillDir); err == nil {
		return fmt.Errorf("skill directory already exists: %s", dirName)
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("creating skill directory: %w", err)
	}

	content := RenderSkillFile(skill)
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		os.RemoveAll(skillDir)
		return fmt.Errorf("writing SKILL.md: %w", err)
	}

	return nil
}

// UpdateSkill overwrites the SKILL.md file for an existing skill.
func UpdateSkill(baseDir, dirName string, skill Skill) error {
	skillDir := filepath.Join(baseDir, dirName)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill directory not found: %s", dirName)
	}

	// If name changed and dir name should update, handle rename
	newDirName := sanitizeDirName(skill.Name)
	if newDirName != dirName && skill.DirName == "" {
		newDir := filepath.Join(baseDir, newDirName)
		if _, err := os.Stat(newDir); err == nil {
			return fmt.Errorf("skill directory already exists: %s", newDirName)
		}
		if err := os.Rename(skillDir, newDir); err != nil {
			return fmt.Errorf("renaming skill directory: %w", err)
		}
		skillDir = newDir
		dirName = newDirName
	}

	content := RenderSkillFile(skill)
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing SKILL.md: %w", err)
	}

	return nil
}

// DeleteSkill removes a skill directory and all its contents.
func DeleteSkill(baseDir, dirName string) error {
	skillDir := filepath.Join(baseDir, dirName)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill directory not found: %s", dirName)
	}

	// Safety: only delete if it contains a SKILL.md
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		return fmt.Errorf("not a valid skill directory (no SKILL.md): %s", dirName)
	}

	return os.RemoveAll(skillDir)
}

// ParseSkillFile reads a SKILL.md file and parses its YAML frontmatter and markdown content.
func ParseSkillFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	frontmatter, body, err := splitFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter in %s: %w", path, err)
	}

	var skill Skill
	if frontmatter != "" {
		if err := yaml.Unmarshal([]byte(frontmatter), &skill); err != nil {
			return nil, fmt.Errorf("parsing YAML in %s: %w", path, err)
		}
	}

	skill.Content = strings.TrimSpace(body)

	// Default name from directory name if not set in frontmatter
	if skill.Name == "" {
		skill.Name = filepath.Base(filepath.Dir(path))
	}

	skill.DirName = filepath.Base(filepath.Dir(path))

	return &skill, nil
}

// RenderSkillFile serializes a Skill back to YAML frontmatter + markdown format.
func RenderSkillFile(skill Skill) string {
	var b strings.Builder

	b.WriteString("---\n")

	// Write fields in a deliberate order for readability
	b.WriteString(fmt.Sprintf("name: %s\n", skill.Name))
	if skill.Description != "" {
		// Use block scalar for multi-line descriptions
		if strings.Contains(skill.Description, "\n") {
			b.WriteString("description: >\n")
			for _, line := range strings.Split(skill.Description, "\n") {
				b.WriteString("  " + strings.TrimSpace(line) + "\n")
			}
		} else {
			b.WriteString(fmt.Sprintf("description: %q\n", skill.Description))
		}
	}
	if skill.Version != "" {
		b.WriteString(fmt.Sprintf("version: %q\n", skill.Version))
	}
	if skill.ArgumentHint != "" {
		b.WriteString(fmt.Sprintf("argument-hint: %q\n", skill.ArgumentHint))
	}
	if skill.DisableModelInvocation {
		b.WriteString("disable-model-invocation: true\n")
	}
	if skill.UserInvocable != nil && !*skill.UserInvocable {
		b.WriteString("user-invocable: false\n")
	}
	if skill.AllowedTools != "" {
		b.WriteString(fmt.Sprintf("allowed-tools: %s\n", skill.AllowedTools))
	}
	if skill.Model != "" {
		b.WriteString(fmt.Sprintf("model: %s\n", skill.Model))
	}
	if skill.Effort != "" {
		b.WriteString(fmt.Sprintf("effort: %s\n", skill.Effort))
	}
	if skill.Context != "" {
		b.WriteString(fmt.Sprintf("context: %s\n", skill.Context))
	}
	if skill.Agent != "" {
		b.WriteString(fmt.Sprintf("agent: %s\n", skill.Agent))
	}
	if skill.Homepage != "" {
		b.WriteString(fmt.Sprintf("homepage: %s\n", skill.Homepage))
	}

	b.WriteString("---\n\n")

	if skill.Content != "" {
		b.WriteString(skill.Content)
		if !strings.HasSuffix(skill.Content, "\n") {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// scanDir reads all skill subdirectories in a directory.
func scanDir(dir, scope string) ([]Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
		skill, err := ParseSkillFile(skillPath)
		if err != nil {
			// Skip directories without valid SKILL.md
			continue
		}

		skill.Scope = scope
		skills = append(skills, *skill)
	}

	return skills, nil
}

// splitFrontmatter separates YAML frontmatter from markdown content.
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(content))

	// Look for opening ---
	if !scanner.Scan() {
		return "", content, nil
	}
	firstLine := strings.TrimSpace(scanner.Text())
	if firstLine != "---" {
		// No frontmatter, entire content is body
		return "", content, nil
	}

	// Collect frontmatter until closing ---
	var fmLines []string
	foundClose := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundClose = true
			break
		}
		fmLines = append(fmLines, line)
	}

	if !foundClose {
		return "", content, fmt.Errorf("unclosed frontmatter (missing closing ---)")
	}

	frontmatter = strings.Join(fmLines, "\n")

	// Rest is body
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}
	body = strings.Join(bodyLines, "\n")

	return frontmatter, body, nil
}

// sanitizeDirName converts a skill name to a valid directory name.
func sanitizeDirName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")

	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	s := result.String()
	// Collapse multiple hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}
