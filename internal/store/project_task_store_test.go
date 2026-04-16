package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davydany/ClawIDE/internal/model"
)

func newTestTaskStore(t *testing.T) *TaskStore {
	t.Helper()
	dir := t.TempDir()
	s, err := NewProjectTaskStore(dir)
	if err != nil {
		t.Fatalf("NewProjectTaskStore: %v", err)
	}
	return s
}

func TestTaskStore_DefaultScaffold(t *testing.T) {
	s := newTestTaskStore(t)
	b, err := s.Board()
	if err != nil {
		t.Fatalf("Board: %v", err)
	}
	if len(b.Columns) != 3 {
		t.Fatalf("default scaffold should have 3 columns, got %d", len(b.Columns))
	}
	wantTitles := []string{"Backlog", "In Progress", "Done"}
	for i, want := range wantTitles {
		if b.Columns[i].Title != want {
			t.Errorf("column %d title: got %q, want %q", i, b.Columns[i].Title, want)
		}
	}
	// File should exist on disk and contain the magic header.
	data, err := os.ReadFile(s.FilePath())
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.HasPrefix(string(data), model.MagicHeader) {
		t.Errorf("saved file missing magic header:\n%s", string(data))
	}
}

func TestTaskStore_AddTaskAndMove(t *testing.T) {
	s := newTestTaskStore(t)

	task, err := s.AddTask("backlog", "Research", "Understand OAuth", "Figure it out")
	if err != nil {
		t.Fatalf("AddTask: %v", err)
	}
	if task.ID == "" {
		t.Error("AddTask should mint an ID")
	}
	if task.Title != "Understand OAuth" {
		t.Errorf("title: %q", task.Title)
	}

	// Move to in-progress, ungrouped, position 0.
	if err := s.MoveTask(task.ID, "in-progress", "", 0); err != nil {
		t.Fatalf("MoveTask: %v", err)
	}
	b, _ := s.Board()
	// Task should no longer be in backlog/Research.
	for _, g := range b.Columns[0].Groups {
		for _, tk := range g.Tasks {
			if tk.ID == task.ID {
				t.Error("task still in backlog after move")
			}
		}
	}
	// Task should be in in-progress.
	found := false
	for _, g := range b.Columns[1].Groups {
		for _, tk := range g.Tasks {
			if tk.ID == task.ID {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("task not found in in-progress after move: %+v", b.Columns[1])
	}
}

func TestTaskStore_UpdateAndDelete(t *testing.T) {
	s := newTestTaskStore(t)
	task, _ := s.AddTask("backlog", "", "Original", "desc")

	updated, err := s.UpdateTask(task.ID, "Renamed", "new desc")
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if updated.Title != "Renamed" || updated.Description != "new desc" {
		t.Errorf("update wrong: %+v", updated)
	}

	if err := s.DeleteTask(task.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	b, _ := s.Board()
	if tk, _, _, _ := b.FindTask(task.ID); tk != nil {
		t.Error("task still present after delete")
	}

	// Deleting non-existent task should error.
	if err := s.DeleteTask("nonexistent"); err == nil {
		t.Error("DeleteTask nonexistent should error")
	}
}

func TestTaskStore_AppendComment(t *testing.T) {
	s := newTestTaskStore(t)
	task, _ := s.AddTask("backlog", "", "t1", "")

	c, err := s.AppendComment(task.ID, model.Comment{
		Author: "AI/claude/sonnet",
		Body:   "Here's what I found...",
	})
	if err != nil {
		t.Fatalf("AppendComment: %v", err)
	}
	if c.Timestamp.IsZero() {
		t.Error("timestamp should be filled in")
	}

	b, _ := s.Board()
	tk, _, _, _ := b.FindTask(task.ID)
	if tk == nil || len(tk.Comments) != 1 {
		t.Fatalf("comment not persisted: %+v", tk)
	}
	if tk.Comments[0].Author != "AI/claude/sonnet" {
		t.Errorf("comment author: %q", tk.Comments[0].Author)
	}
	if tk.Comments[0].Body != "Here's what I found..." {
		t.Errorf("comment body: %q", tk.Comments[0].Body)
	}

	// And the markdown file on disk should contain the comment.
	data, _ := os.ReadFile(s.FilePath())
	if !strings.Contains(string(data), "> **[") {
		t.Errorf("markdown missing comment:\n%s", string(data))
	}
}

func TestTaskStore_ColumnOps(t *testing.T) {
	s := newTestTaskStore(t)

	col, err := s.AddColumn("Review")
	if err != nil {
		t.Fatalf("AddColumn: %v", err)
	}
	if col.ID != "review" {
		t.Errorf("slug: %q", col.ID)
	}

	// Rename.
	renamed, err := s.RenameColumn("review", "Ready for QA")
	if err != nil {
		t.Fatalf("RenameColumn: %v", err)
	}
	if renamed.ID != "ready-for-qa" {
		t.Errorf("renamed slug: %q", renamed.ID)
	}

	// Delete empty column succeeds.
	if err := s.DeleteColumn("ready-for-qa"); err != nil {
		t.Errorf("DeleteColumn empty: %v", err)
	}

	// Deleting a non-empty column should refuse.
	s.AddTask("backlog", "", "task", "")
	if err := s.DeleteColumn("backlog"); err == nil {
		t.Error("DeleteColumn non-empty should fail")
	}
}

func TestTaskStore_HandEditRecovered(t *testing.T) {
	s := newTestTaskStore(t)

	// Hand-edit the file: add a task without an ID, directly.
	path := s.FilePath()
	data, _ := os.ReadFile(path)
	edited := string(data) + "\n### Hand-edited task\nSome description.\n"
	if err := os.WriteFile(path, []byte(edited), 0644); err != nil {
		t.Fatal(err)
	}

	// Next mutating op (which reloads) should mint an ID for the hand-edited task.
	_, err := s.AddTask("backlog", "", "Another", "")
	if err != nil {
		t.Fatalf("AddTask after hand-edit: %v", err)
	}

	b, _ := s.Board()
	// Find the hand-edited task (the one in "Done" since it was appended after the "# Done" heading).
	var handTask *model.Task
	for ci := range b.Columns {
		for gi := range b.Columns[ci].Groups {
			for ti := range b.Columns[ci].Groups[gi].Tasks {
				if b.Columns[ci].Groups[gi].Tasks[ti].Title == "Hand-edited task" {
					handTask = &b.Columns[ci].Groups[gi].Tasks[ti]
				}
			}
		}
	}
	if handTask == nil {
		t.Fatal("hand-edited task lost after reload")
	}
	if handTask.ID == "" {
		t.Error("hand-edited task should have been minted an ID on save")
	}
}

func TestTaskStore_GlobalStore(t *testing.T) {
	dir := t.TempDir()
	s, err := NewGlobalTaskStore(dir)
	if err != nil {
		t.Fatalf("NewGlobalTaskStore: %v", err)
	}
	if s.FilePath() != filepath.Join(dir, "tasks.md") {
		t.Errorf("global path: %q", s.FilePath())
	}
	if _, err := os.Stat(s.FilePath()); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}
