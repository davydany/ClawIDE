package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
)

// TaskStore wraps a single tasks.md file (either project-scoped or global). Every mutating method
// re-reads from disk before applying the change so external hand-edits aren't stomped.
type TaskStore struct {
	mu       sync.RWMutex
	filePath string
	board    model.Board
}

// NewProjectTaskStore creates or opens <projectDir>/.clawide/tasks.md. Missing files are scaffolded
// with a default three-column board so the UI has something to render on first access.
func NewProjectTaskStore(projectDir string) (*TaskStore, error) {
	path := filepath.Join(projectDir, ".clawide", "tasks.md")
	return newTaskStore(path)
}

// NewGlobalTaskStore creates or opens <globalDir>/tasks.md (typically ~/.clawide-global/tasks.md).
func NewGlobalTaskStore(globalDir string) (*TaskStore, error) {
	path := filepath.Join(globalDir, "tasks.md")
	return newTaskStore(path)
}

func newTaskStore(path string) (*TaskStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("creating tasks dir: %w", err)
	}
	s := &TaskStore{filePath: path}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		s.board = model.DefaultBoard()
		if err := s.saveLocked(); err != nil {
			return nil, fmt.Errorf("scaffolding tasks.md: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("stat tasks.md: %w", err)
	} else {
		if err := s.loadLocked(); err != nil {
			return nil, fmt.Errorf("loading tasks.md: %w", err)
		}
	}
	return s, nil
}

// FilePath returns the on-disk location of tasks.md. Callers use this for git status/commit.
func (s *TaskStore) FilePath() string {
	return s.filePath
}

// Board returns a defensive deep copy of the current in-memory board so callers can safely encode
// or mutate without holding the store lock. Uses serialize/parse for the copy — cheap and reuses
// already-tested code.
func (s *TaskStore) Board() (model.Board, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := SerializeBoard(s.board)
	return ParseBoard(out)
}

// Reload re-reads the file from disk. Useful when the caller knows the file may have changed
// externally and wants a fresh snapshot without mutating.
func (s *TaskStore) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

// loadLocked reads and parses tasks.md. Caller must hold the write lock.
func (s *TaskStore) loadLocked() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	board, err := ParseBoard(data)
	if err != nil {
		return err
	}
	s.board = board
	return nil
}

// saveLocked serializes the in-memory board and writes it to disk atomically (write to temp, then
// rename). Caller must hold the write lock.
func (s *TaskStore) saveLocked() error {
	data := SerializeBoard(s.board)
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

// findColumn returns a pointer to the column with the given slug in the in-memory board.
// Caller must hold the lock.
func (s *TaskStore) findColumn(slug string) *model.Column {
	for i := range s.board.Columns {
		if s.board.Columns[i].ID == slug {
			return &s.board.Columns[i]
		}
	}
	return nil
}

// ensureGroup returns a pointer to the group with the given title inside column, creating it
// if it doesn't exist. groupTitle == "" refers to the implicit ungrouped group (always at index 0).
// Caller must hold the lock.
func (s *TaskStore) ensureGroup(col *model.Column, groupTitle string) *model.Group {
	for i := range col.Groups {
		if col.Groups[i].Title == groupTitle {
			return &col.Groups[i]
		}
	}
	col.Groups = append(col.Groups, model.Group{Title: groupTitle})
	return &col.Groups[len(col.Groups)-1]
}

// ---------------- Mutating operations ----------------

// AddTask appends a new task to the named column/group. If the column doesn't exist, returns an
// error. If the group doesn't exist inside the column, it's created.
func (s *TaskStore) AddTask(columnSlug, groupTitle, title, description string) (model.Task, error) {
	if err := model.ValidateTaskTitle(title); err != nil {
		return model.Task{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return model.Task{}, err
	}

	col := s.findColumn(columnSlug)
	if col == nil {
		return model.Task{}, fmt.Errorf("column %q not found", columnSlug)
	}
	group := s.ensureGroup(col, groupTitle)

	task := model.Task{
		Title:       title,
		Description: description,
	}
	group.Tasks = append(group.Tasks, task)

	// Mint ID for the new task (and any others that happen to be missing from hand-edits).
	s.board.MintMissingIDs()

	if err := s.saveLocked(); err != nil {
		return model.Task{}, err
	}
	// Return the now-saved copy so the caller sees the minted ID.
	return group.Tasks[len(group.Tasks)-1], nil
}

// UpdateTask rewrites a task's title and description in place. Comments are left untouched.
func (s *TaskStore) UpdateTask(id, title, description string) (model.Task, error) {
	if err := model.ValidateTaskTitle(title); err != nil {
		return model.Task{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return model.Task{}, err
	}
	task, _, _, _ := s.board.FindTask(id)
	if task == nil {
		return model.Task{}, fmt.Errorf("task %q not found", id)
	}
	task.Title = title
	task.Description = description
	if err := s.saveLocked(); err != nil {
		return model.Task{}, err
	}
	return *task, nil
}

// DeleteTask removes a task by ID.
func (s *TaskStore) DeleteTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return err
	}
	_, ci, gi, ti := s.board.FindTask(id)
	if ci < 0 {
		return fmt.Errorf("task %q not found", id)
	}
	tasks := s.board.Columns[ci].Groups[gi].Tasks
	s.board.Columns[ci].Groups[gi].Tasks = append(tasks[:ti], tasks[ti+1:]...)
	return s.saveLocked()
}

// MoveTask moves a task to another column/group/position. toGroupTitle "" means the ungrouped
// slot. toIndex is clamped to [0, len(destGroup.Tasks)]. A negative toIndex appends to the end.
func (s *TaskStore) MoveTask(id, toColumnSlug, toGroupTitle string, toIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return err
	}
	task, ci, gi, ti := s.board.FindTask(id)
	if task == nil {
		return fmt.Errorf("task %q not found", id)
	}
	// Snapshot the task value before removing it from its current position — the pointer becomes
	// invalid after the slice mutation below.
	moved := *task

	// Remove from current location.
	src := &s.board.Columns[ci].Groups[gi]
	src.Tasks = append(src.Tasks[:ti], src.Tasks[ti+1:]...)

	// Resolve destination.
	destCol := s.findColumn(toColumnSlug)
	if destCol == nil {
		return fmt.Errorf("destination column %q not found", toColumnSlug)
	}
	destGroup := s.ensureGroup(destCol, toGroupTitle)

	// Clamp insertion index.
	if toIndex < 0 || toIndex > len(destGroup.Tasks) {
		toIndex = len(destGroup.Tasks)
	}
	destGroup.Tasks = append(destGroup.Tasks, model.Task{})
	copy(destGroup.Tasks[toIndex+1:], destGroup.Tasks[toIndex:])
	destGroup.Tasks[toIndex] = moved

	return s.saveLocked()
}

// SetLinkedBranch updates the task's LinkedBranch field in place. An empty string clears the link.
// The caller is responsible for validating the branch exists; this method only mutates the field
// and re-serializes.
func (s *TaskStore) SetLinkedBranch(id, branch string) (model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return model.Task{}, err
	}
	task, _, _, _ := s.board.FindTask(id)
	if task == nil {
		return model.Task{}, fmt.Errorf("task %q not found", id)
	}
	task.LinkedBranch = branch
	if err := s.saveLocked(); err != nil {
		return model.Task{}, err
	}
	return *task, nil
}

// AppendComment adds a comment to the task identified by id. The comment's timestamp is
// overwritten with the current time if it's zero, so handlers can pass a bare Comment struct.
func (s *TaskStore) AppendComment(id string, c model.Comment) (model.Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return model.Comment{}, err
	}
	task, _, _, _ := s.board.FindTask(id)
	if task == nil {
		return model.Comment{}, fmt.Errorf("task %q not found", id)
	}
	if c.Timestamp.IsZero() {
		c.Timestamp = time.Now().Truncate(time.Minute)
	}
	task.Comments = append(task.Comments, c)
	if err := s.saveLocked(); err != nil {
		return model.Comment{}, err
	}
	return c, nil
}

// AddColumn appends a new column with the given title to the end of the board. The slug is
// derived from the title.
func (s *TaskStore) AddColumn(title string) (model.Column, error) {
	if err := model.ValidateColumnTitle(title); err != nil {
		return model.Column{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return model.Column{}, err
	}
	slug := model.ColumnSlug(title)
	if s.findColumn(slug) != nil {
		return model.Column{}, fmt.Errorf("column with slug %q already exists", slug)
	}
	col := model.Column{
		ID:     slug,
		Title:  title,
		Groups: []model.Group{{}},
	}
	s.board.Columns = append(s.board.Columns, col)
	if err := s.saveLocked(); err != nil {
		return model.Column{}, err
	}
	return col, nil
}

// RenameColumn changes an existing column's title. The slug is recomputed from the new title.
func (s *TaskStore) RenameColumn(oldSlug, newTitle string) (model.Column, error) {
	if err := model.ValidateColumnTitle(newTitle); err != nil {
		return model.Column{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return model.Column{}, err
	}
	col := s.findColumn(oldSlug)
	if col == nil {
		return model.Column{}, fmt.Errorf("column %q not found", oldSlug)
	}
	newSlug := model.ColumnSlug(newTitle)
	if newSlug != oldSlug && s.findColumn(newSlug) != nil {
		return model.Column{}, fmt.Errorf("column with slug %q already exists", newSlug)
	}
	col.Title = newTitle
	col.ID = newSlug
	if err := s.saveLocked(); err != nil {
		return model.Column{}, err
	}
	return *col, nil
}

// DeleteColumn removes an empty column. Refuses if the column still contains tasks so the caller
// can't silently lose work — the UI should surface this as a confirmation prompt.
func (s *TaskStore) DeleteColumn(slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return err
	}
	idx := -1
	for i := range s.board.Columns {
		if s.board.Columns[i].ID == slug {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("column %q not found", slug)
	}
	// Reject if any group in the column has tasks.
	for _, g := range s.board.Columns[idx].Groups {
		if len(g.Tasks) > 0 {
			return fmt.Errorf("column %q is not empty", slug)
		}
	}
	s.board.Columns = append(s.board.Columns[:idx], s.board.Columns[idx+1:]...)
	return s.saveLocked()
}

// MoveColumn repositions a column to toIndex. Clamped to [0, len-1].
func (s *TaskStore) MoveColumn(slug string, toIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return err
	}
	fromIdx := -1
	for i := range s.board.Columns {
		if s.board.Columns[i].ID == slug {
			fromIdx = i
			break
		}
	}
	if fromIdx < 0 {
		return fmt.Errorf("column %q not found", slug)
	}
	if toIndex < 0 {
		toIndex = 0
	}
	if toIndex >= len(s.board.Columns) {
		toIndex = len(s.board.Columns) - 1
	}
	if fromIdx == toIndex {
		return nil // no-op
	}
	col := s.board.Columns[fromIdx]
	// Remove from old position.
	s.board.Columns = append(s.board.Columns[:fromIdx], s.board.Columns[fromIdx+1:]...)
	// Insert at new position.
	s.board.Columns = append(s.board.Columns[:toIndex], append([]model.Column{col}, s.board.Columns[toIndex:]...)...)
	return s.saveLocked()
}
