package store

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
)

func TestParseBoard_Empty(t *testing.T) {
	b, err := ParseBoard([]byte(""))
	if err != nil {
		t.Fatalf("ParseBoard empty: %v", err)
	}
	if len(b.Columns) != 0 {
		t.Errorf("expected 0 columns, got %d", len(b.Columns))
	}
}

func TestParseBoard_SingleTaskMinimal(t *testing.T) {
	src := `<!-- clawide-tasks v1 -->

# Backlog

### Understand OAuth <!-- id: task-1 -->
Figure out the auth code flow.
`
	b, err := ParseBoard([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(b.Columns) != 1 {
		t.Fatalf("want 1 column, got %d", len(b.Columns))
	}
	col := b.Columns[0]
	if col.Title != "Backlog" || col.ID != "backlog" {
		t.Errorf("column title/id wrong: %+v", col)
	}
	if len(col.Groups) != 1 || col.Groups[0].Title != "" {
		t.Fatalf("want 1 ungrouped group, got %+v", col.Groups)
	}
	tasks := col.Groups[0].Tasks
	if len(tasks) != 1 {
		t.Fatalf("want 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "task-1" {
		t.Errorf("task ID: got %q, want task-1", tasks[0].ID)
	}
	if tasks[0].Title != "Understand OAuth" {
		t.Errorf("task title: got %q, want 'Understand OAuth'", tasks[0].Title)
	}
	if tasks[0].Description != "Figure out the auth code flow." {
		t.Errorf("description: got %q", tasks[0].Description)
	}
}

func TestParseBoard_GroupsAndComments(t *testing.T) {
	src := `<!-- clawide-tasks v1 -->

# Backlog

## Research
### Understand OAuth <!-- id: t1 -->
Figure out the auth code flow.

> **[2026-04-15 10:22 · AI]** OAuth 2.0 is...
> continuation line.

> **[2026-04-15 11:04 · user]** Also cover refresh tokens.

### Evaluate job queues <!-- id: t2 -->
Compare asynq vs river.

## Bugs
### Scroll jank <!-- id: t3 -->
Reproduces on long docs.

# In Progress

### Build Kanban UI <!-- id: t4 -->
Vanilla JS + Tailwind.

# Done
`
	b, err := ParseBoard([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(b.Columns) != 3 {
		t.Fatalf("want 3 columns, got %d: %+v", len(b.Columns), b.Columns)
	}

	// Backlog column has an ungrouped group (empty) + Research + Bugs.
	backlog := b.Columns[0]
	if backlog.Title != "Backlog" {
		t.Errorf("col 0 title: %q", backlog.Title)
	}
	// Should have at least Research and Bugs groups. The implicit ungrouped group may also be
	// present at index 0 — we allow it as long as it's empty.
	var research, bugs *model.Group
	for i := range backlog.Groups {
		g := &backlog.Groups[i]
		switch g.Title {
		case "Research":
			research = g
		case "Bugs":
			bugs = g
		case "":
			if len(g.Tasks) != 0 {
				t.Errorf("ungrouped group has %d tasks, want 0", len(g.Tasks))
			}
		}
	}
	if research == nil {
		t.Fatal("no Research group")
	}
	if bugs == nil {
		t.Fatal("no Bugs group")
	}

	if len(research.Tasks) != 2 {
		t.Fatalf("Research tasks: want 2, got %d", len(research.Tasks))
	}
	oauth := research.Tasks[0]
	if oauth.ID != "t1" || oauth.Title != "Understand OAuth" {
		t.Errorf("oauth task wrong: %+v", oauth)
	}
	if len(oauth.Comments) != 2 {
		t.Fatalf("oauth comments: want 2, got %d: %+v", len(oauth.Comments), oauth.Comments)
	}
	if oauth.Comments[0].Author != "AI" {
		t.Errorf("comment 0 author: %q", oauth.Comments[0].Author)
	}
	if !strings.Contains(oauth.Comments[0].Body, "OAuth 2.0 is") || !strings.Contains(oauth.Comments[0].Body, "continuation line") {
		t.Errorf("comment 0 body missing lines: %q", oauth.Comments[0].Body)
	}
	if oauth.Comments[1].Author != "user" {
		t.Errorf("comment 1 author: %q", oauth.Comments[1].Author)
	}
	wantTS, _ := time.Parse(model.CommentTimestampLayout, "2026-04-15 10:22")
	if !oauth.Comments[0].Timestamp.Equal(wantTS) {
		t.Errorf("comment 0 timestamp: got %v, want %v", oauth.Comments[0].Timestamp, wantTS)
	}

	// Done column should exist and be empty.
	done := b.Columns[2]
	if done.Title != "Done" {
		t.Errorf("col 2 title: %q", done.Title)
	}
	for _, g := range done.Groups {
		if len(g.Tasks) != 0 {
			t.Errorf("Done should have no tasks, got %+v", g.Tasks)
		}
	}
}

func TestRoundTrip_Equivalence(t *testing.T) {
	// Parse → serialize → parse must produce an equivalent board.
	src := `<!-- clawide-tasks v1 -->

# Backlog

## Research
### Understand OAuth <!-- id: t1 -->
Figure out the auth code flow.
Multiple description lines should survive.

> **[2026-04-15 10:22 · AI/claude/sonnet]** OAuth 2.0 is...
> continuation line.

> **[2026-04-15 11:04 · user]** Also cover refresh tokens.

### Evaluate job queues <!-- id: t2 -->
Compare asynq vs river.

## Bugs
### Scroll jank <!-- id: t3 -->
Reproduces on long docs.

# In Progress

### Build Kanban UI <!-- id: t4 -->
Vanilla JS + Tailwind.

# Done
`
	b1, err := ParseBoard([]byte(src))
	if err != nil {
		t.Fatalf("first parse: %v", err)
	}
	out := SerializeBoard(b1)
	b2, err := ParseBoard(out)
	if err != nil {
		t.Fatalf("second parse: %v", err)
	}
	if !reflect.DeepEqual(b1, b2) {
		t.Errorf("round trip lost data\n--first--\n%#v\n--second--\n%#v\n--markdown--\n%s", b1, b2, string(out))
	}
}

func TestRoundTrip_Stable(t *testing.T) {
	// Serialize → parse → serialize must be byte-identical to the first serialize.
	b := model.Board{
		Columns: []model.Column{
			{
				ID:    "backlog",
				Title: "Backlog",
				Groups: []model.Group{
					{Title: "Research", Tasks: []model.Task{
						{
							ID:          "t1",
							Title:       "Understand OAuth",
							Description: "Figure out the auth code flow.",
							Comments: []model.Comment{
								{
									Timestamp: mustParseTS(t, "2026-04-15 10:22"),
									Author:    "AI/claude/sonnet",
									Body:      "OAuth 2.0 is...",
								},
							},
						},
					}},
				},
			},
			{ID: "done", Title: "Done", Groups: []model.Group{{}}},
		},
	}
	out1 := SerializeBoard(b)
	b2, err := ParseBoard(out1)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out2 := SerializeBoard(b2)
	if string(out1) != string(out2) {
		t.Errorf("serialize not stable:\n--first--\n%s\n--second--\n%s", string(out1), string(out2))
	}
}

func TestParseBoard_MissingTaskID(t *testing.T) {
	// Parser does not mint IDs — MintMissingIDs is the caller's job. Verify ID stays empty.
	src := `<!-- clawide-tasks v1 -->

# Backlog

### Task with no ID
Some description.
`
	b, err := ParseBoard([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	task := b.Columns[0].Groups[0].Tasks[0]
	if task.ID != "" {
		t.Errorf("parser minted an ID (%q); that's MintMissingIDs's job", task.ID)
	}
	if task.Title != "Task with no ID" {
		t.Errorf("title: %q", task.Title)
	}

	// Now mint and verify exactly one ID was assigned.
	n := b.MintMissingIDs()
	if n != 1 {
		t.Errorf("MintMissingIDs returned %d, want 1", n)
	}
	if b.Columns[0].Groups[0].Tasks[0].ID == "" {
		t.Error("ID still empty after MintMissingIDs")
	}
}

func TestParseBoard_HandEditRecovery(t *testing.T) {
	// User hand-edits the file: adds a task without an ID, moves a task between columns,
	// writes a comment with an unusual format. The parser should still produce a usable board.
	src := `<!-- clawide-tasks v1 -->

# Backlog

### New hand-edited task
No ID yet, added by the user directly.

> A raw blockquote that doesn't follow the canonical comment format.

# Done

### Previously finished <!-- id: old-1 -->
`
	b, err := ParseBoard([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(b.Columns) != 2 {
		t.Fatalf("want 2 columns, got %d", len(b.Columns))
	}
	newTask := b.Columns[0].Groups[0].Tasks[0]
	if newTask.ID != "" {
		t.Errorf("hand-edited task should have empty ID, got %q", newTask.ID)
	}
	if len(newTask.Comments) != 1 {
		t.Fatalf("want 1 comment, got %d", len(newTask.Comments))
	}
	if newTask.Comments[0].Author != "" {
		t.Errorf("unstructured comment should have empty author, got %q", newTask.Comments[0].Author)
	}
	if !strings.Contains(newTask.Comments[0].Body, "raw blockquote") {
		t.Errorf("comment body: %q", newTask.Comments[0].Body)
	}
}

func TestColumnSlug(t *testing.T) {
	cases := map[string]string{
		"Backlog":             "backlog",
		"In Progress":         "in-progress",
		"  Done  ":            "done",
		"Ready for QA":        "ready-for-qa",
		"Release 2.0":         "release-20",
		"---weird--title---":  "weird-title",
		"":                    "column",
	}
	for input, want := range cases {
		if got := model.ColumnSlug(input); got != want {
			t.Errorf("ColumnSlug(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSerialize_MagicHeaderPresent(t *testing.T) {
	out := SerializeBoard(model.DefaultBoard())
	if !strings.HasPrefix(string(out), model.MagicHeader) {
		t.Errorf("output missing magic header:\n%s", string(out))
	}
	// Default scaffold should have all three columns.
	for _, want := range []string{"# Backlog", "# In Progress", "# Done"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestFindTaskAndColumn(t *testing.T) {
	b := model.Board{
		Columns: []model.Column{
			{ID: "backlog", Title: "Backlog", Groups: []model.Group{
				{Tasks: []model.Task{{ID: "t1", Title: "one"}}},
			}},
			{ID: "done", Title: "Done", Groups: []model.Group{
				{Tasks: []model.Task{{ID: "t2", Title: "two"}}},
			}},
		},
	}
	if tk, ci, _, _ := b.FindTask("t2"); tk == nil || ci != 1 || tk.Title != "two" {
		t.Errorf("FindTask t2 wrong: task=%v ci=%d", tk, ci)
	}
	if tk, _, _, _ := b.FindTask("missing"); tk != nil {
		t.Error("FindTask missing should be nil")
	}
	if col, idx := b.FindColumn("done"); col == nil || idx != 1 {
		t.Errorf("FindColumn done: %v %d", col, idx)
	}
}

func mustParseTS(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(model.CommentTimestampLayout, s)
	if err != nil {
		t.Fatalf("parse ts %q: %v", s, err)
	}
	return ts
}
