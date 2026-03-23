package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSkillFile(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0755)

	content := `---
name: test-skill
description: "A test skill for unit testing"
version: "1.0"
argument-hint: "[url]"
disable-model-invocation: true
user-invocable: false
allowed-tools: Read, Grep, Bash
model: claude-opus-4-6
effort: high
context: fork
agent: Explore
homepage: https://example.com
---

# Test Skill

Do something useful with $ARGUMENTS.
`
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	skill, err := ParseSkillFile(skillPath)
	if err != nil {
		t.Fatalf("ParseSkillFile failed: %v", err)
	}

	if skill.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "test-skill")
	}
	if skill.Description != "A test skill for unit testing" {
		t.Errorf("Description = %q, want %q", skill.Description, "A test skill for unit testing")
	}
	if skill.Version != "1.0" {
		t.Errorf("Version = %q, want %q", skill.Version, "1.0")
	}
	if skill.ArgumentHint != "[url]" {
		t.Errorf("ArgumentHint = %q, want %q", skill.ArgumentHint, "[url]")
	}
	if !skill.DisableModelInvocation {
		t.Error("DisableModelInvocation should be true")
	}
	if skill.UserInvocable == nil || *skill.UserInvocable {
		t.Error("UserInvocable should be false")
	}
	if skill.AllowedTools != "Read, Grep, Bash" {
		t.Errorf("AllowedTools = %q, want %q", skill.AllowedTools, "Read, Grep, Bash")
	}
	if skill.Model != "claude-opus-4-6" {
		t.Errorf("Model = %q, want %q", skill.Model, "claude-opus-4-6")
	}
	if skill.Effort != "high" {
		t.Errorf("Effort = %q, want %q", skill.Effort, "high")
	}
	if skill.Context != "fork" {
		t.Errorf("Context = %q, want %q", skill.Context, "fork")
	}
	if skill.Agent != "Explore" {
		t.Errorf("Agent = %q, want %q", skill.Agent, "Explore")
	}
	if skill.Homepage != "https://example.com" {
		t.Errorf("Homepage = %q, want %q", skill.Homepage, "https://example.com")
	}
	if skill.Content == "" {
		t.Error("Content should not be empty")
	}
	if skill.DirName != "test-skill" {
		t.Errorf("DirName = %q, want %q", skill.DirName, "test-skill")
	}
}

func TestParseSkillFile_NameFromDir(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-cool-skill")
	os.MkdirAll(skillDir, 0755)

	content := `---
description: "No name field"
---

Some content.
`
	skillPath := filepath.Join(skillDir, "SKILL.md")
	os.WriteFile(skillPath, []byte(content), 0644)

	skill, err := ParseSkillFile(skillPath)
	if err != nil {
		t.Fatalf("ParseSkillFile failed: %v", err)
	}
	if skill.Name != "my-cool-skill" {
		t.Errorf("Name = %q, want %q (from dir)", skill.Name, "my-cool-skill")
	}
}

func TestParseSkillFile_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "bare-skill")
	os.MkdirAll(skillDir, 0755)

	content := "# Just Markdown\n\nNo frontmatter here."
	skillPath := filepath.Join(skillDir, "SKILL.md")
	os.WriteFile(skillPath, []byte(content), 0644)

	skill, err := ParseSkillFile(skillPath)
	if err != nil {
		t.Fatalf("ParseSkillFile failed: %v", err)
	}
	if skill.Name != "bare-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "bare-skill")
	}
	if skill.Content != content {
		t.Errorf("Content = %q, want %q", skill.Content, content)
	}
}

func TestCreateAndGetSkill(t *testing.T) {
	dir := t.TempDir()

	boolTrue := true
	skill := Skill{
		Name:          "new-skill",
		Description:   "A newly created skill",
		UserInvocable: &boolTrue,
		Content:       "# Instructions\n\nDo the thing.",
	}

	if err := CreateSkill(dir, skill); err != nil {
		t.Fatalf("CreateSkill failed: %v", err)
	}

	// Verify file exists
	skillPath := filepath.Join(dir, "new-skill", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Fatal("SKILL.md was not created")
	}

	// Read it back
	got, err := GetSkill(dir, "new-skill")
	if err != nil {
		t.Fatalf("GetSkill failed: %v", err)
	}
	if got.Name != "new-skill" {
		t.Errorf("Name = %q, want %q", got.Name, "new-skill")
	}
	if got.Description != "A newly created skill" {
		t.Errorf("Description = %q, want %q", got.Description, "A newly created skill")
	}
	if got.Content != "# Instructions\n\nDo the thing." {
		t.Errorf("Content = %q, want %q", got.Content, "# Instructions\n\nDo the thing.")
	}
}

func TestCreateSkill_DuplicateDir(t *testing.T) {
	dir := t.TempDir()

	skill := Skill{Name: "dup-skill", Content: "content"}
	if err := CreateSkill(dir, skill); err != nil {
		t.Fatal(err)
	}

	err := CreateSkill(dir, skill)
	if err == nil {
		t.Error("Expected error for duplicate skill directory")
	}
}

func TestUpdateSkill(t *testing.T) {
	dir := t.TempDir()

	skill := Skill{Name: "update-me", Content: "original"}
	CreateSkill(dir, skill)

	skill.Content = "updated content"
	skill.Description = "now with description"
	if err := UpdateSkill(dir, "update-me", skill); err != nil {
		t.Fatalf("UpdateSkill failed: %v", err)
	}

	got, err := GetSkill(dir, "update-me")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "updated content" {
		t.Errorf("Content = %q, want %q", got.Content, "updated content")
	}
	if got.Description != "now with description" {
		t.Errorf("Description = %q, want %q", got.Description, "now with description")
	}
}

func TestDeleteSkill(t *testing.T) {
	dir := t.TempDir()

	skill := Skill{Name: "delete-me", Content: "goodbye"}
	CreateSkill(dir, skill)

	if err := DeleteSkill(dir, "delete-me"); err != nil {
		t.Fatalf("DeleteSkill failed: %v", err)
	}

	skillDir := filepath.Join(dir, "delete-me")
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Error("Skill directory should have been removed")
	}
}

func TestDeleteSkill_NoSKILLMD(t *testing.T) {
	dir := t.TempDir()

	// Create a directory without SKILL.md — shouldn't be deletable as a "skill"
	os.MkdirAll(filepath.Join(dir, "not-a-skill"), 0755)

	err := DeleteSkill(dir, "not-a-skill")
	if err == nil {
		t.Error("Expected error when deleting directory without SKILL.md")
	}
}

func TestListSkills(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	CreateSkill(globalDir, Skill{Name: "global-skill", Content: "global"})
	CreateSkill(projectDir, Skill{Name: "project-skill", Content: "project"})

	all, err := ListSkills(globalDir, projectDir)
	if err != nil {
		t.Fatalf("ListSkills failed: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("Expected 2 skills, got %d", len(all))
	}

	// Check scopes are set correctly
	scopes := map[string]string{}
	for _, sk := range all {
		scopes[sk.Name] = sk.Scope
	}
	if scopes["global-skill"] != "global" {
		t.Error("global-skill should have scope 'global'")
	}
	if scopes["project-skill"] != "project" {
		t.Error("project-skill should have scope 'project'")
	}
}

func TestListSkills_EmptyDirs(t *testing.T) {
	all, err := ListSkills("/nonexistent/global", "/nonexistent/project")
	if err != nil {
		t.Fatalf("ListSkills with nonexistent dirs should not error: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("Expected 0 skills, got %d", len(all))
	}
}

func TestSanitizeDirName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My Skill", "my-skill"},
		{"hello_world!", "helloworld"},
		{"a--b--c", "a-b-c"},
		{"-leading-", "leading"},
		{"CamelCase", "camelcase"},
		{"with spaces and CAPS", "with-spaces-and-caps"},
	}

	for _, tt := range tests {
		got := sanitizeDirName(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeDirName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRenderSkillFile(t *testing.T) {
	boolFalse := false
	skill := Skill{
		Name:                   "render-test",
		Description:            "Test rendering",
		Version:                "2.0",
		DisableModelInvocation: true,
		UserInvocable:          &boolFalse,
		AllowedTools:           "Read, Grep",
		Model:                  "claude-opus-4-6",
		Content:                "# Hello\n\nWorld",
	}

	rendered := RenderSkillFile(skill)

	// Verify it contains key parts
	if !containsStr(rendered, "name: render-test") {
		t.Error("Missing name field")
	}
	if !containsStr(rendered, "disable-model-invocation: true") {
		t.Error("Missing disable-model-invocation field")
	}
	if !containsStr(rendered, "user-invocable: false") {
		t.Error("Missing user-invocable field")
	}
	if !containsStr(rendered, "# Hello") {
		t.Error("Missing content")
	}

	// Verify round-trip
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "render-test")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(rendered), 0644)

	parsed, err := ParseSkillFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("Round-trip parse failed: %v", err)
	}
	if parsed.Name != "render-test" {
		t.Errorf("Round-trip Name = %q, want %q", parsed.Name, "render-test")
	}
	if parsed.DisableModelInvocation != true {
		t.Error("Round-trip DisableModelInvocation should be true")
	}
}

func TestIsUserInvocable(t *testing.T) {
	// nil → true (default)
	s1 := Skill{}
	if !s1.IsUserInvocable() {
		t.Error("nil UserInvocable should default to true")
	}

	// explicit true
	bTrue := true
	s2 := Skill{UserInvocable: &bTrue}
	if !s2.IsUserInvocable() {
		t.Error("explicit true should be true")
	}

	// explicit false
	bFalse := false
	s3 := Skill{UserInvocable: &bFalse}
	if s3.IsUserInvocable() {
		t.Error("explicit false should be false")
	}
}

func TestMoveSkill(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	CreateSkill(srcDir, Skill{Name: "move-me", Content: "movable"})

	if err := MoveSkill(srcDir, dstDir, "move-me"); err != nil {
		t.Fatalf("MoveSkill failed: %v", err)
	}

	// Source should be gone
	if _, err := os.Stat(filepath.Join(srcDir, "move-me")); !os.IsNotExist(err) {
		t.Error("Source skill directory should have been removed")
	}

	// Destination should exist
	got, err := GetSkill(dstDir, "move-me")
	if err != nil {
		t.Fatalf("GetSkill from destination failed: %v", err)
	}
	if got.Name != "move-me" {
		t.Errorf("Name = %q, want %q", got.Name, "move-me")
	}
}

func TestMoveSkill_TargetExists(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	CreateSkill(srcDir, Skill{Name: "conflict", Content: "src"})
	CreateSkill(dstDir, Skill{Name: "conflict", Content: "dst"})

	err := MoveSkill(srcDir, dstDir, "conflict")
	if err == nil {
		t.Error("Expected error when target already exists")
	}
}

func TestMoveSkill_SourceNotFound(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	err := MoveSkill(srcDir, dstDir, "nonexistent")
	if err == nil {
		t.Error("Expected error when source skill not found")
	}
}

func containsStr(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && contains(haystack, needle)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
