package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Board is the in-memory representation of a tasks.md file.
type Board struct {
	Preamble string   `json:"preamble"`
	Columns  []Column `json:"columns"`
}

type Column struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Groups []Group `json:"groups"`
}

type Group struct {
	Title string `json:"title"`
	Tasks []Task `json:"tasks"`
}

type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Comments    []Comment `json:"comments"`
}

type Comment struct {
	Timestamp time.Time `json:"timestamp"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
}

// MagicHeader marks a tasks.md file as clawide-managed and allows format versioning.
const MagicHeader = "<!-- clawide-tasks v1 -->"

// CommentTimestampLayout is the canonical timestamp format used in comment headers.
const CommentTimestampLayout = "2006-01-02 15:04"

// ColumnSlug converts a column title into a URL-safe slug used to identify the column in API routes.
func ColumnSlug(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '-' || r == '_':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if out == "" {
		return "column"
	}
	return out
}

// MintMissingIDs assigns a new UUID to any task without one. Returns the number of IDs assigned.
// The parser deliberately leaves IDs empty so Parse is deterministic for tests; the store calls
// this before saving so hand-edits picked up on reload get stable IDs.
func (b *Board) MintMissingIDs() int {
	assigned := 0
	for ci := range b.Columns {
		for gi := range b.Columns[ci].Groups {
			for ti := range b.Columns[ci].Groups[gi].Tasks {
				if b.Columns[ci].Groups[gi].Tasks[ti].ID == "" {
					b.Columns[ci].Groups[gi].Tasks[ti].ID = uuid.New().String()
					assigned++
				}
			}
		}
	}
	return assigned
}

// FindTask walks the board and returns a pointer to the task with the given ID, plus its
// column/group coordinates. Returns (nil, ...) if no task matches. The pointer lets callers mutate
// in place; callers hold the store's write lock while using the pointer.
func (b *Board) FindTask(id string) (task *Task, columnIndex, groupIndex, taskIndex int) {
	for ci := range b.Columns {
		for gi := range b.Columns[ci].Groups {
			for ti := range b.Columns[ci].Groups[gi].Tasks {
				if b.Columns[ci].Groups[gi].Tasks[ti].ID == id {
					return &b.Columns[ci].Groups[gi].Tasks[ti], ci, gi, ti
				}
			}
		}
	}
	return nil, -1, -1, -1
}

// FindColumn returns the column with the given slug plus its index, or (nil, -1) if not found.
func (b *Board) FindColumn(slug string) (*Column, int) {
	for i := range b.Columns {
		if b.Columns[i].ID == slug {
			return &b.Columns[i], i
		}
	}
	return nil, -1
}

// ValidateColumnTitle is lenient — columns are human-facing headings, not filenames. Only guards
// against empty titles and newlines (which would break the markdown format).
func ValidateColumnTitle(title string) error {
	t := strings.TrimSpace(title)
	if t == "" {
		return fmt.Errorf("column title cannot be empty")
	}
	if strings.ContainsAny(t, "\r\n") {
		return fmt.Errorf("column title cannot contain newlines")
	}
	if len(t) > 255 {
		return fmt.Errorf("column title exceeds 255 characters")
	}
	return nil
}

// ValidateTaskTitle uses the same lenient rules as ValidateColumnTitle.
func ValidateTaskTitle(title string) error {
	t := strings.TrimSpace(title)
	if t == "" {
		return fmt.Errorf("task title cannot be empty")
	}
	if strings.ContainsAny(t, "\r\n") {
		return fmt.Errorf("task title cannot contain newlines")
	}
	if len(t) > 255 {
		return fmt.Errorf("task title exceeds 255 characters")
	}
	return nil
}

// DefaultBoard returns a freshly-scaffolded board with three empty columns. Used on first access
// to a project or global store that doesn't yet have a tasks.md file.
func DefaultBoard() Board {
	return Board{
		Columns: []Column{
			{ID: "backlog", Title: "Backlog", Groups: []Group{{}}},
			{ID: "in-progress", Title: "In Progress", Groups: []Group{{}}},
			{ID: "done", Title: "Done", Groups: []Group{{}}},
		},
	}
}
