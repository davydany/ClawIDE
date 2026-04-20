package breakdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskSlug(t *testing.T) {
	cases := []struct {
		title, taskID, want string
	}{
		{"Wire up OAuth callback", "abc-123-def-456", "wire-up-oauth-callback"},
		{"  Trim me!  ", "id1", "trim-me"},
		{"Multiple   spaces --- and_underscores", "id2", "multiple-spaces-and-underscores"},
		{"", "abc-12345-xyz", "task-abc12345"},
		{"!!!", "abc-12345", "task-abc12345"},
		{"Release 2.0 🚀", "id3", "release-2-0"},
		{"中文タスク", "abc1234567", "task-abc12345"},
	}
	for _, tc := range cases {
		got := TaskSlug(tc.title, tc.taskID)
		if got != tc.want {
			t.Errorf("TaskSlug(%q, %q) = %q, want %q", tc.title, tc.taskID, got, tc.want)
		}
	}
}

func TestTaskSlug_LengthCap(t *testing.T) {
	long := strings.Repeat("word ", 30)
	got := TaskSlug(long, "id")
	if len(got) > slugMaxLen {
		t.Errorf("slug length %d exceeds cap %d: %q", len(got), slugMaxLen, got)
	}
	if !strings.HasPrefix(got, "word-word") {
		t.Errorf("expected word-prefixed slug, got %q", got)
	}
	if strings.HasSuffix(got, "-") {
		t.Errorf("slug should not end with dash: %q", got)
	}
}

func TestUpdateClaudeMD_NoFile(t *testing.T) {
	dir := t.TempDir()
	if err := UpdateClaudeMD(dir, "wire-oauth", "Wire OAuth"); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	if !strings.Contains(s, regionBegin) || !strings.Contains(s, regionEnd) {
		t.Errorf("missing markers:\n%s", s)
	}
	if !strings.Contains(s, "tasks/wire-oauth.md") {
		t.Errorf("missing entry:\n%s", s)
	}
	if !strings.Contains(s, "Wire OAuth") {
		t.Errorf("missing title:\n%s", s)
	}
}

func TestUpdateClaudeMD_ExistingFile_NoRegion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	existing := "# My project\n\nSome prose."
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	if err := UpdateClaudeMD(dir, "wire-oauth", "Wire OAuth"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	s := string(got)
	if !strings.HasPrefix(s, "# My project") {
		t.Errorf("preexisting content not preserved:\n%s", s)
	}
	if !strings.Contains(s, "Some prose.") {
		t.Errorf("preexisting prose lost:\n%s", s)
	}
	if !strings.Contains(s, regionBegin) {
		t.Errorf("region not appended:\n%s", s)
	}
}

func TestUpdateClaudeMD_ReplaceAndMerge(t *testing.T) {
	dir := t.TempDir()
	// First run — creates the region.
	if err := UpdateClaudeMD(dir, "task-a", "Task A title"); err != nil {
		t.Fatal(err)
	}
	// Second run for a different task — should accumulate.
	if err := UpdateClaudeMD(dir, "task-b", "Task B title"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	s := string(got)
	if strings.Count(s, regionBegin) != 1 || strings.Count(s, regionEnd) != 1 {
		t.Errorf("expected exactly one region, got:\n%s", s)
	}
	if !strings.Contains(s, "tasks/task-a.md") || !strings.Contains(s, "tasks/task-b.md") {
		t.Errorf("region missing entries:\n%s", s)
	}
	// Order should be stable: A before B.
	if strings.Index(s, "task-a") >= strings.Index(s, "task-b") {
		t.Errorf("order not preserved:\n%s", s)
	}

	// Third run re-titling task-a — should update in place, not duplicate.
	if err := UpdateClaudeMD(dir, "task-a", "Task A renamed"); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	s = string(got)
	// Each entry line contains "tasks/<slug>.md" twice (once in backticks, once as link target).
	// So a single entry produces count==2; duplication would produce count==4.
	if strings.Count(s, "tasks/task-a.md") != 2 {
		t.Errorf("task-a line count wrong (want 2, got %d):\n%s", strings.Count(s, "tasks/task-a.md"), s)
	}
	if !strings.Contains(s, "Task A renamed") {
		t.Errorf("title not updated:\n%s", s)
	}
	if strings.Contains(s, "Task A title") {
		t.Errorf("old title not replaced:\n%s", s)
	}
}

func TestUpdateClaudeMD_CRLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	crlf := "# Project\r\n\r\nLine 1\r\nLine 2\r\n"
	if err := os.WriteFile(path, []byte(crlf), 0644); err != nil {
		t.Fatal(err)
	}
	if err := UpdateClaudeMD(dir, "wire", "Wire"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), regionBegin) {
		t.Errorf("CRLF file didn't get region added:\n%s", string(got))
	}
}

func TestUpdateClaudeMD_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(path, []byte("# Project\nNo trailing newline here"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := UpdateClaudeMD(dir, "wire", "Wire"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	s := string(got)
	if !strings.Contains(s, "No trailing newline here\n\n"+regionBegin) {
		t.Errorf("separator between original and region wrong:\n%s", s)
	}
}

func TestWriteSubtaskFile_Basic(t *testing.T) {
	dir := t.TempDir()
	checklist := "- [ ] Step one\n- [ ] Step two\n- [ ] Step three\n"
	path, err := WriteSubtaskFile(dir, "wire", "task-1", "Wire OAuth", checklist, true)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if got, want := path, filepath.Join(dir, "tasks", "wire.md"); got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
	body, _ := os.ReadFile(path)
	if !strings.Contains(string(body), "# Wire OAuth") {
		t.Errorf("missing H1:\n%s", string(body))
	}
	if !strings.Contains(string(body), taskIDHeaderPrefix+" task-1") {
		t.Errorf("missing task-id header:\n%s", string(body))
	}
	if !strings.Contains(string(body), "- [ ] Step one") {
		t.Errorf("missing checklist:\n%s", string(body))
	}
}

func TestWriteSubtaskFile_OverwriteProtection(t *testing.T) {
	dir := t.TempDir()
	checklist := "- [ ] One\n"

	// First task writes the file.
	if _, err := WriteSubtaskFile(dir, "shared", "task-1", "Title 1", checklist, true); err != nil {
		t.Fatal(err)
	}

	// Same task, overwrite=false → ErrExists.
	if _, err := WriteSubtaskFile(dir, "shared", "task-1", "Title 1", checklist, false); err != ErrExists {
		t.Errorf("expected ErrExists, got %v", err)
	}

	// Same task, overwrite=true → succeeds.
	if _, err := WriteSubtaskFile(dir, "shared", "task-1", "Title 1", checklist, true); err != nil {
		t.Errorf("overwrite=true failed: %v", err)
	}

	// Different task with same slug → should write to a disambiguated filename.
	path, err := WriteSubtaskFile(dir, "shared", "task-2-abcdef", "Title 2", checklist, true)
	if err != nil {
		t.Fatalf("collision resolution failed: %v", err)
	}
	if filepath.Base(path) == "shared.md" {
		t.Errorf("expected disambiguated filename, got %s", path)
	}
	if !strings.Contains(filepath.Base(path), "task-2") {
		t.Errorf("disambiguated filename missing task suffix: %s", path)
	}
}

func TestWriteSubtaskFile_EmptyChecklist(t *testing.T) {
	dir := t.TempDir()
	if _, err := WriteSubtaskFile(dir, "wire", "t1", "T", "   \n", true); err == nil {
		t.Errorf("expected error for empty checklist")
	}
}

func TestWriteSubtaskFile_TooLarge(t *testing.T) {
	dir := t.TempDir()
	big := strings.Repeat("x", MaxChecklistBytes+1)
	if _, err := WriteSubtaskFile(dir, "wire", "t1", "T", big, true); err == nil {
		t.Errorf("expected size error")
	}
}
