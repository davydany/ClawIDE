package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAgentFile(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: test-agent
description: "A test agent"
model: claude-sonnet-4-6
allowed-tools: Read, Write, Bash
agent: Explore
---

You are a test agent that does testing things.
`
	path := filepath.Join(dir, "test-agent.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ag, err := ParseAgentFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if ag.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got %q", ag.Name)
	}
	if ag.Description != "A test agent" {
		t.Errorf("expected description 'A test agent', got %q", ag.Description)
	}
	if ag.Model != "claude-sonnet-4-6" {
		t.Errorf("expected model 'claude-sonnet-4-6', got %q", ag.Model)
	}
	if ag.AllowedTools != "Read, Write, Bash" {
		t.Errorf("expected allowed_tools 'Read, Write, Bash', got %q", ag.AllowedTools)
	}
	if ag.AgentType != "Explore" {
		t.Errorf("expected agent type 'Explore', got %q", ag.AgentType)
	}
	if ag.Content != "You are a test agent that does testing things." {
		t.Errorf("unexpected content: %q", ag.Content)
	}
	if ag.FileName != "test-agent" {
		t.Errorf("expected file_name 'test-agent', got %q", ag.FileName)
	}
}

func TestParseAgentFile_NameFromFile(t *testing.T) {
	dir := t.TempDir()
	content := `---
description: "No name in frontmatter"
---

Some instructions.
`
	path := filepath.Join(dir, "my-agent.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ag, err := ParseAgentFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if ag.Name != "my-agent" {
		t.Errorf("expected name 'my-agent' from filename, got %q", ag.Name)
	}
}

func TestParseAgentFile_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "Just plain markdown instructions."
	path := filepath.Join(dir, "plain.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ag, err := ParseAgentFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if ag.Name != "plain" {
		t.Errorf("expected name 'plain', got %q", ag.Name)
	}
	if ag.Content != "Just plain markdown instructions." {
		t.Errorf("unexpected content: %q", ag.Content)
	}
}

func TestCreateAndGetAgent(t *testing.T) {
	dir := t.TempDir()
	ag := Agent{
		Name:        "Code Reviewer",
		Description: "Reviews code for quality",
		Model:       "claude-opus-4-6",
		Content:     "Review code carefully.",
	}

	if err := CreateAgent(dir, ag); err != nil {
		t.Fatal(err)
	}

	got, err := GetAgent(dir, "code-reviewer")
	if err != nil {
		t.Fatal(err)
	}

	if got.Name != "Code Reviewer" {
		t.Errorf("expected name 'Code Reviewer', got %q", got.Name)
	}
	if got.Description != "Reviews code for quality" {
		t.Errorf("expected description, got %q", got.Description)
	}
	if got.Model != "claude-opus-4-6" {
		t.Errorf("expected model 'claude-opus-4-6', got %q", got.Model)
	}
	if got.Content != "Review code carefully." {
		t.Errorf("unexpected content: %q", got.Content)
	}
}

func TestCreateAgent_Duplicate(t *testing.T) {
	dir := t.TempDir()
	ag := Agent{Name: "test", Content: "hello"}

	if err := CreateAgent(dir, ag); err != nil {
		t.Fatal(err)
	}
	if err := CreateAgent(dir, ag); err == nil {
		t.Error("expected error on duplicate create")
	}
}

func TestUpdateAgent(t *testing.T) {
	dir := t.TempDir()
	ag := Agent{Name: "my-agent", Content: "original"}
	if err := CreateAgent(dir, ag); err != nil {
		t.Fatal(err)
	}

	ag.Content = "updated content"
	ag.Description = "now with description"
	if err := UpdateAgent(dir, "my-agent", ag); err != nil {
		t.Fatal(err)
	}

	got, err := GetAgent(dir, "my-agent")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "updated content" {
		t.Errorf("expected updated content, got %q", got.Content)
	}
	if got.Description != "now with description" {
		t.Errorf("expected description, got %q", got.Description)
	}
}

func TestDeleteAgent(t *testing.T) {
	dir := t.TempDir()
	ag := Agent{Name: "doomed", Content: "bye"}
	if err := CreateAgent(dir, ag); err != nil {
		t.Fatal(err)
	}

	if err := DeleteAgent(dir, "doomed"); err != nil {
		t.Fatal(err)
	}

	if _, err := GetAgent(dir, "doomed"); err == nil {
		t.Error("expected error after delete")
	}
}

func TestListAgents(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	CreateAgent(globalDir, Agent{Name: "global-agent", Content: "g"})
	CreateAgent(projectDir, Agent{Name: "project-agent", Content: "p"})

	all, err := ListAgents(globalDir, projectDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(all))
	}

	scopeMap := map[string]string{}
	for _, a := range all {
		scopeMap[a.Name] = a.Scope
	}
	if scopeMap["global-agent"] != "global" {
		t.Error("expected global-agent to have global scope")
	}
	if scopeMap["project-agent"] != "project" {
		t.Error("expected project-agent to have project scope")
	}
}

func TestListAgents_EmptyDirs(t *testing.T) {
	dir := t.TempDir()
	all, err := ListAgents(dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 agents, got %d", len(all))
	}
}

func TestMoveAgent(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	CreateAgent(srcDir, Agent{Name: "movable", Content: "move me"})

	if err := MoveAgent(srcDir, dstDir, "movable"); err != nil {
		t.Fatal(err)
	}

	// Should not exist in source
	if _, err := GetAgent(srcDir, "movable"); err == nil {
		t.Error("expected agent gone from source")
	}

	// Should exist in destination
	got, err := GetAgent(dstDir, "movable")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "move me" {
		t.Errorf("unexpected content after move: %q", got.Content)
	}
}

func TestMoveAgent_TargetExists(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	CreateAgent(srcDir, Agent{Name: "conflict", Content: "src"})
	CreateAgent(dstDir, Agent{Name: "conflict", Content: "dst"})

	if err := MoveAgent(srcDir, dstDir, "conflict"); err == nil {
		t.Error("expected error on move conflict")
	}
}

func TestMoveAgent_SourceNotFound(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	if err := MoveAgent(srcDir, dstDir, "nonexistent"); err == nil {
		t.Error("expected error for missing source")
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My Agent", "my-agent"},
		{"Code Reviewer!", "code-reviewer"},
		{"test--agent", "test-agent"},
		{"  spaces  ", "spaces"},
		{"UPPER-case", "upper-case"},
		{"123-numeric", "123-numeric"},
	}

	for _, tc := range tests {
		got := sanitizeFileName(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizeFileName(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestRenderAgentFile(t *testing.T) {
	ag := Agent{
		Name:         "test",
		Description:  "A test",
		Model:        "claude-sonnet-4-6",
		AllowedTools: "Read, Write",
		AgentType:    "Explore",
		Content:      "Do the thing.",
	}

	rendered := RenderAgentFile(ag)
	parsed, err := ParseAgentFile(writeTempFile(t, rendered))
	if err != nil {
		t.Fatal(err)
	}

	if parsed.Name != ag.Name {
		t.Errorf("round-trip name: got %q, want %q", parsed.Name, ag.Name)
	}
	if parsed.Description != ag.Description {
		t.Errorf("round-trip description: got %q, want %q", parsed.Description, ag.Description)
	}
	if parsed.Model != ag.Model {
		t.Errorf("round-trip model: got %q, want %q", parsed.Model, ag.Model)
	}
	if parsed.AllowedTools != ag.AllowedTools {
		t.Errorf("round-trip allowed_tools: got %q, want %q", parsed.AllowedTools, ag.AllowedTools)
	}
	if parsed.AgentType != ag.AgentType {
		t.Errorf("round-trip agent_type: got %q, want %q", parsed.AgentType, ag.AgentType)
	}
	if parsed.Content != ag.Content {
		t.Errorf("round-trip content: got %q, want %q", parsed.Content, ag.Content)
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
