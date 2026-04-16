package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/go-chi/chi/v5"
)

// resolveTaskStore picks the project-specific store if project_id is set, otherwise the global
// store. Used by every task handler so scope is consistent across the API.
func (h *Handlers) resolveTaskStore(r *http.Request) (*store.TaskStore, string, error) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		return h.globalTaskStore, "", nil
	}
	s, err := h.getProjectTaskStore(projectID)
	if err != nil {
		return nil, projectID, err
	}
	return s, projectID, nil
}

// GetTaskBoard returns the full board for the given scope.
// GET /api/tasks/board?project_id=<id>
func (h *Handlers) GetTaskBoard(w http.ResponseWriter, r *http.Request) {
	s, _, err := h.resolveTaskStore(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	board, err := s.Board()
	if err != nil {
		log.Printf("GetTaskBoard: %v", err)
		http.Error(w, "failed to load board", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, board)
}

// GetAggregatedTaskBoard returns the global board plus a per-project snapshot of every project
// board, flattened into one response. The global board is editable; project sections are marked
// read-only via metadata fields so the UI knows not to offer drag/edit.
// GET /api/tasks/board/aggregated
func (h *Handlers) GetAggregatedTaskBoard(w http.ResponseWriter, r *http.Request) {
	type projectBoard struct {
		ProjectID   string      `json:"project_id"`
		ProjectName string      `json:"project_name"`
		Board       model.Board `json:"board"`
	}
	type aggregated struct {
		Global   model.Board    `json:"global"`
		Projects []projectBoard `json:"projects"`
	}

	globalBoard, err := h.globalTaskStore.Board()
	if err != nil {
		log.Printf("aggregated: global board error: %v", err)
		http.Error(w, "failed to load global board", http.StatusInternalServerError)
		return
	}
	out := aggregated{Global: globalBoard, Projects: []projectBoard{}}

	for _, proj := range h.store.GetProjects() {
		ps, err := h.getProjectTaskStore(proj.ID)
		if err != nil {
			log.Printf("aggregated: project %s store error: %v", proj.ID, err)
			continue
		}
		b, err := ps.Board()
		if err != nil {
			log.Printf("aggregated: project %s board error: %v", proj.ID, err)
			continue
		}
		out.Projects = append(out.Projects, projectBoard{
			ProjectID:   proj.ID,
			ProjectName: proj.Name,
			Board:       b,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateTask appends a new task to a column (and optional group) in the resolved scope.
// POST /api/tasks?project_id=<id>
func (h *Handlers) CreateTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Column      string `json:"column"`
		Group       string `json:"group"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	s, projectID, err := h.resolveTaskStore(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	task, err := s.AddTask(body.Column, body.Group, body.Title, body.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Record metric
	if m := h.getProjectTaskMetrics(projectID); m != nil {
		m.RecordCreated()
	}
	writeJSON(w, http.StatusCreated, task)
}

// UpdateTask rewrites a task's title and description.
// PUT /api/tasks/{taskID}?project_id=<id>
func (h *Handlers) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskID")
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	s, _, err := h.resolveTaskStore(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	task, err := s.UpdateTask(id, body.Title, body.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// DeleteTask removes a task by ID.
// DELETE /api/tasks/{taskID}?project_id=<id>
func (h *Handlers) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskID")
	s, _, err := h.resolveTaskStore(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := s.DeleteTask(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MoveTask relocates a task to another column/group/position.
// POST /api/tasks/{taskID}/move?project_id=<id>
func (h *Handlers) MoveTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskID")
	var body struct {
		ToColumn string `json:"to_column"`
		ToGroup  string `json:"to_group"`
		ToIndex  int    `json:"to_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	s, projectID, err := h.resolveTaskStore(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := s.MoveTask(id, body.ToColumn, body.ToGroup, body.ToIndex); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Record "closed" metric when a task lands in a "done"-ish column.
	if m := h.getProjectTaskMetrics(projectID); m != nil {
		if strings.Contains(strings.ToLower(body.ToColumn), "done") {
			m.RecordClosed()
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// AddTaskComment appends a user-authored comment to a task.
// POST /api/tasks/{taskID}/comments?project_id=<id>
func (h *Handlers) AddTaskComment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskID")
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Body == "" {
		http.Error(w, "comment body is required", http.StatusBadRequest)
		return
	}
	s, _, err := h.resolveTaskStore(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	c, err := s.AppendComment(id, model.Comment{Author: "user", Body: body.Body})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// CreateTaskColumn appends a new column to the board.
// POST /api/tasks/columns?project_id=<id>
func (h *Handlers) CreateTaskColumn(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	s, _, err := h.resolveTaskStore(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	col, err := s.AddColumn(body.Title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, col)
}

// RenameTaskColumn changes a column's title (and therefore its slug).
// PUT /api/tasks/columns/{slug}?project_id=<id>
func (h *Handlers) RenameTaskColumn(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	s, _, err := h.resolveTaskStore(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	col, err := s.RenameColumn(slug, body.Title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, col)
}

// DeleteTaskColumn removes an empty column. Non-empty columns return 409 Conflict so the UI can
// surface a confirm-or-move flow.
// DELETE /api/tasks/columns/{slug}?project_id=<id>
func (h *Handlers) DeleteTaskColumn(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	s, _, err := h.resolveTaskStore(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := s.DeleteColumn(slug); err != nil {
		// Non-empty columns get 409 so the UI can prompt "move tasks first"; everything else
		// (missing column, I/O error) falls through to 404/500.
		status := http.StatusNotFound
		if strings.HasSuffix(err.Error(), "is not empty") {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetTaskSettings returns the current task storage mode for a project.
// GET /api/tasks/settings?project_id=<id>
func (h *Handlers) GetTaskSettings(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}
	project, ok := h.store.GetProject(projectID)
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}
	mode := string(project.TaskStorage)
	if mode == "" {
		mode = "in-project"
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"task_storage": mode,
	})
}

// SetTaskSettings toggles the task storage location for a project between "in-project" and
// "global". When switching, the cached store is invalidated so the next access creates a fresh
// store pointing to the new location. The existing file at the old location is NOT moved or
// deleted — the user can manually move it, or the new location will scaffold a fresh board.
// PUT /api/tasks/settings?project_id=<id>
func (h *Handlers) SetTaskSettings(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}
	var body struct {
		TaskStorage string `json:"task_storage"` // "in-project" or "global"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	var mode model.TaskStorageMode
	switch body.TaskStorage {
	case "in-project", "":
		mode = model.TaskStorageInProject
	case "global":
		mode = model.TaskStorageGlobal
	default:
		http.Error(w, "invalid task_storage value; must be 'in-project' or 'global'", http.StatusBadRequest)
		return
	}
	project, ok := h.store.GetProject(projectID)
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}
	project.TaskStorage = mode
	project.UpdatedAt = time.Now()
	if err := h.store.UpdateProject(project); err != nil {
		log.Printf("SetTaskSettings: UpdateProject error: %v", err)
		http.Error(w, "failed to update project", http.StatusInternalServerError)
		return
	}
	// Evict the cached store so the next access creates one pointing to the new path.
	h.invalidateProjectTaskStore(projectID)

	result := string(mode)
	if result == "" {
		result = "in-project"
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"task_storage": result,
	})
}

// MoveTaskColumn reorders a column to a new position.
// POST /api/tasks/columns/{slug}/move?project_id=<id>
func (h *Handlers) MoveTaskColumn(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var body struct {
		ToIndex int `json:"to_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	s, _, err := h.resolveTaskStore(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := s.MoveColumn(slug, body.ToIndex); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetTaskMetrics returns daily created/closed counts for the last N days (default 30).
// GET /api/tasks/metrics?project_id=<id>&days=30
func (h *Handlers) GetTaskMetrics(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}
	m := h.getProjectTaskMetrics(projectID)
	if m == nil {
		writeJSON(w, http.StatusOK, []store.DaySummary{})
		return
	}
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	writeJSON(w, http.StatusOK, m.Recent(days))
}

// writeJSON is a tiny helper so handlers don't repeat the boilerplate.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}
