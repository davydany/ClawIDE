package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Agent represents a Claude Code agent definition parsed from a .md file.
type Agent struct {
	Name         string `json:"name" yaml:"name"`
	Description  string `json:"description" yaml:"description"`
	Model        string `json:"model,omitempty" yaml:"model,omitempty"`
	AllowedTools string `json:"allowed_tools,omitempty" yaml:"allowed-tools,omitempty"`
	AgentType    string `json:"agent_type,omitempty" yaml:"agent,omitempty"`
	Content      string `json:"content" yaml:"-"`
	Scope        string `json:"scope" yaml:"-"`
	FileName     string `json:"file_name" yaml:"-"`
}

// GlobalAgentsDir returns the path to the global agents directory.
func GlobalAgentsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "agents")
}

// ProjectAgentsDir returns the path to the project-level agents directory.
func ProjectAgentsDir(projectPath string) string {
	return filepath.Join(projectPath, ".claude", "agents")
}

// ListAgents scans both global and project agent directories and returns all agents.
func ListAgents(globalDir, projectDir string) ([]Agent, error) {
	var all []Agent

	if globalDir != "" {
		global, err := scanDir(globalDir, "global")
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("scanning global agents: %w", err)
		}
		all = append(all, global...)
	}

	if projectDir != "" {
		project, err := scanDir(projectDir, "project")
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("scanning project agents: %w", err)
		}
		all = append(all, project...)
	}

	return all, nil
}

// GetAgent reads and parses a single agent from a base directory.
func GetAgent(baseDir, fileName string) (*Agent, error) {
	agentPath := filepath.Join(baseDir, fileName+".md")
	return ParseAgentFile(agentPath)
}

// CreateAgent creates a new agent .md file.
func CreateAgent(baseDir string, ag Agent) error {
	fileName := ag.FileName
	if fileName == "" {
		fileName = sanitizeFileName(ag.Name)
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("creating agents directory: %w", err)
	}

	agentPath := filepath.Join(baseDir, fileName+".md")
	if _, err := os.Stat(agentPath); err == nil {
		return fmt.Errorf("agent file already exists: %s", fileName)
	}

	content := RenderAgentFile(ag)
	if err := os.WriteFile(agentPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing agent file: %w", err)
	}

	return nil
}

// UpdateAgent overwrites the .md file for an existing agent.
func UpdateAgent(baseDir, oldFileName string, ag Agent) error {
	oldPath := filepath.Join(baseDir, oldFileName+".md")
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("agent file not found: %s", oldFileName)
	}

	// If name changed and file name should update, handle rename
	newFileName := sanitizeFileName(ag.Name)
	if newFileName != oldFileName && ag.FileName == "" {
		newPath := filepath.Join(baseDir, newFileName+".md")
		if _, err := os.Stat(newPath); err == nil {
			return fmt.Errorf("agent file already exists: %s", newFileName)
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("renaming agent file: %w", err)
		}
		oldPath = newPath
	}

	content := RenderAgentFile(ag)
	if err := os.WriteFile(oldPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing agent file: %w", err)
	}

	return nil
}

// MoveAgent moves an agent from srcDir to dstDir.
func MoveAgent(srcDir, dstDir, fileName string) error {
	srcPath := filepath.Join(srcDir, fileName+".md")
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("source agent not found: %s", fileName)
	}

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	dstPath := filepath.Join(dstDir, fileName+".md")
	if _, err := os.Stat(dstPath); err == nil {
		return fmt.Errorf("agent already exists in target scope: %s", fileName)
	}

	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("moving agent file: %w", err)
	}

	return nil
}

// DeleteAgent removes an agent .md file.
func DeleteAgent(baseDir, fileName string) error {
	agentPath := filepath.Join(baseDir, fileName+".md")
	if _, err := os.Stat(agentPath); os.IsNotExist(err) {
		return fmt.Errorf("agent file not found: %s", fileName)
	}

	return os.Remove(agentPath)
}

// ParseAgentFile reads an agent .md file and parses its YAML frontmatter and markdown content.
func ParseAgentFile(path string) (*Agent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	frontmatter, body, err := splitFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter in %s: %w", path, err)
	}

	var ag Agent
	if frontmatter != "" {
		if err := yaml.Unmarshal([]byte(frontmatter), &ag); err != nil {
			return nil, fmt.Errorf("parsing YAML in %s: %w", path, err)
		}
	}

	ag.Content = strings.TrimSpace(body)

	// Default name from filename if not set in frontmatter
	if ag.Name == "" {
		ag.Name = strings.TrimSuffix(filepath.Base(path), ".md")
	}

	ag.FileName = strings.TrimSuffix(filepath.Base(path), ".md")

	return &ag, nil
}

// RenderAgentFile serializes an Agent back to YAML frontmatter + markdown format.
func RenderAgentFile(ag Agent) string {
	var b strings.Builder

	b.WriteString("---\n")

	b.WriteString(fmt.Sprintf("name: %s\n", ag.Name))
	if ag.Description != "" {
		if strings.Contains(ag.Description, "\n") {
			b.WriteString("description: >\n")
			for _, line := range strings.Split(ag.Description, "\n") {
				b.WriteString("  " + strings.TrimSpace(line) + "\n")
			}
		} else {
			b.WriteString(fmt.Sprintf("description: %q\n", ag.Description))
		}
	}
	if ag.Model != "" {
		b.WriteString(fmt.Sprintf("model: %s\n", ag.Model))
	}
	if ag.AllowedTools != "" {
		b.WriteString(fmt.Sprintf("allowed-tools: %s\n", ag.AllowedTools))
	}
	if ag.AgentType != "" {
		b.WriteString(fmt.Sprintf("agent: %s\n", ag.AgentType))
	}

	b.WriteString("---\n\n")

	if ag.Content != "" {
		b.WriteString(ag.Content)
		if !strings.HasSuffix(ag.Content, "\n") {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// scanDir reads all agent .md files in a directory.
func scanDir(dir, scope string) ([]Agent, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var agents []Agent
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, ".") {
			continue
		}

		ag, err := ParseAgentFile(filepath.Join(dir, name))
		if err != nil {
			// Skip files that can't be parsed
			continue
		}

		ag.Scope = scope
		agents = append(agents, *ag)
	}

	return agents, nil
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

// sanitizeFileName converts an agent name to a valid file name (without extension).
func sanitizeFileName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")

	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	s := result.String()
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}
