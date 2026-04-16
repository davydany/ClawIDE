package handler

import (
	"fmt"
	"log"
	"sync"

	"github.com/davydany/ClawIDE/internal/aicli"
	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/mcpserver"
	"github.com/davydany/ClawIDE/internal/migration"
	ptyPkg "github.com/davydany/ClawIDE/internal/pty"
	"github.com/davydany/ClawIDE/internal/sse"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/davydany/ClawIDE/internal/tmpl"
	"github.com/davydany/ClawIDE/internal/updater"
	"github.com/davydany/ClawIDE/internal/wizard"
)

type Handlers struct {
	cfg               *config.Config
	store             *store.Store
	renderer          *tmpl.Renderer
	ptyManager        *ptyPkg.Manager
	snippetStore      *store.SnippetStore
	notificationStore *store.NotificationStore
	noteStore         *store.NoteStore         // global (legacy) note store
	bookmarkStore     *store.BookmarkStore     // global (legacy) bookmark store
	voiceBoxStore     *store.VoiceBoxStore
	scratchpadStore   *store.ScratchpadStore
	sseHub            *sse.Hub
	updater           *updater.Updater
	wizardJobs        *wizard.JobTracker
	wizardGenerator   *wizard.Generator
	mcpProcessManager *mcpserver.ProcessManager

	// Project-scoped stores, keyed by projectID. Lazily initialized.
	projectNoteStores     map[string]*store.ProjectNoteStore
	projectBookmarkStores map[string]*store.ProjectBookmarkStore
	projectTaskStores     map[string]*store.TaskStore
	projectTaskMetrics    map[string]*store.TaskMetrics
	projectStoreMu        sync.Mutex

	// Global tasks board (~/.clawide/tasks.md) — shared across all projects, initialized at startup.
	globalTaskStore *store.TaskStore

	// AI CLI provider registry — used by the task manager's "Ask AI" endpoint.
	aiRegistry *aicli.Registry
}

func New(cfg *config.Config, st *store.Store, renderer *tmpl.Renderer, ptyMgr *ptyPkg.Manager, snippetSt *store.SnippetStore, notifSt *store.NotificationStore, noteSt *store.NoteStore, bookmarkSt *store.BookmarkStore, voiceBoxSt *store.VoiceBoxStore, scratchpadSt *store.ScratchpadStore, globalTaskSt *store.TaskStore, aiReg *aicli.Registry, hub *sse.Hub, upd *updater.Updater, wizJobs *wizard.JobTracker, wizGen *wizard.Generator) *Handlers {
	return &Handlers{
		cfg:                   cfg,
		store:                 st,
		renderer:              renderer,
		ptyManager:            ptyMgr,
		snippetStore:          snippetSt,
		notificationStore:     notifSt,
		noteStore:             noteSt,
		bookmarkStore:         bookmarkSt,
		voiceBoxStore:         voiceBoxSt,
		scratchpadStore:       scratchpadSt,
		globalTaskStore:       globalTaskSt,
		aiRegistry:            aiReg,
		sseHub:                hub,
		updater:               upd,
		wizardJobs:            wizJobs,
		wizardGenerator:       wizGen,
		mcpProcessManager:     mcpserver.NewProcessManager(),
		projectNoteStores:     make(map[string]*store.ProjectNoteStore),
		projectBookmarkStores: make(map[string]*store.ProjectBookmarkStore),
		projectTaskStores:     make(map[string]*store.TaskStore),
		projectTaskMetrics:    make(map[string]*store.TaskMetrics),
	}
}

// getProjectTaskStore returns (or lazily creates) the task store for a given project. The storage
// location depends on the project's TaskStorage field: in-project (.clawide/tasks.md) or global
// (~/.clawide/projects/<id>/tasks.md). Same lazy init pattern as getProjectNoteStore.
func (h *Handlers) getProjectTaskStore(projectID string) (*store.TaskStore, error) {
	h.projectStoreMu.Lock()
	defer h.projectStoreMu.Unlock()

	if s, ok := h.projectTaskStores[projectID]; ok {
		return s, nil
	}

	project, ok := h.store.GetProject(projectID)
	if !ok {
		return nil, fmt.Errorf("project %s not found", projectID)
	}

	storageDir := project.TaskStorageDir(h.cfg.DataDir)
	taskStore, err := store.NewProjectTaskStore(storageDir)
	if err != nil {
		return nil, err
	}
	h.projectTaskStores[projectID] = taskStore
	return taskStore, nil
}

// getProjectTaskMetrics returns (or lazily creates) the metrics tracker for a project. Always
// stored at ~/.clawide/projects/<id>/task-metrics.json regardless of the task storage mode.
func (h *Handlers) getProjectTaskMetrics(projectID string) *store.TaskMetrics {
	if projectID == "" {
		return nil
	}
	h.projectStoreMu.Lock()
	defer h.projectStoreMu.Unlock()
	if m, ok := h.projectTaskMetrics[projectID]; ok {
		return m
	}
	m, err := store.NewTaskMetrics(h.cfg.DataDir, projectID)
	if err != nil {
		log.Printf("task metrics init error for %s: %v", projectID, err)
		return nil
	}
	h.projectTaskMetrics[projectID] = m
	return m
}

// invalidateProjectTaskStore evicts the cached task store for a project so the next access
// creates a fresh one (pointing to the potentially new storage location after a toggle).
func (h *Handlers) invalidateProjectTaskStore(projectID string) {
	h.projectStoreMu.Lock()
	defer h.projectStoreMu.Unlock()
	delete(h.projectTaskStores, projectID)
}

// getProjectNoteStore returns (or lazily creates) a project-scoped note store.
func (h *Handlers) getProjectNoteStore(projectID string) (*store.ProjectNoteStore, error) {
	h.projectStoreMu.Lock()
	defer h.projectStoreMu.Unlock()

	if s, ok := h.projectNoteStores[projectID]; ok {
		return s, nil
	}

	project, ok := h.store.GetProject(projectID)
	if !ok {
		return nil, fmt.Errorf("project %s not found", projectID)
	}

	noteStore, _, err := migration.EnsureProjectStores(
		project.Path,
		h.cfg.NotesFilePath(),
		h.cfg.BookmarksFilePath(),
		projectID,
	)
	if err != nil {
		return nil, err
	}

	h.projectNoteStores[projectID] = noteStore

	// Also cache bookmark store if not present
	if _, exists := h.projectBookmarkStores[projectID]; !exists {
		_, bookmarkStore, err := migration.EnsureProjectStores(
			project.Path,
			h.cfg.NotesFilePath(),
			h.cfg.BookmarksFilePath(),
			projectID,
		)
		if err != nil {
			log.Printf("Warning: failed to init project bookmark store for %s: %v", projectID, err)
		} else {
			h.projectBookmarkStores[projectID] = bookmarkStore
		}
	}

	return noteStore, nil
}

// getProjectBookmarkStore returns (or lazily creates) a project-scoped bookmark store.
func (h *Handlers) getProjectBookmarkStore(projectID string) (*store.ProjectBookmarkStore, error) {
	h.projectStoreMu.Lock()
	defer h.projectStoreMu.Unlock()

	if s, ok := h.projectBookmarkStores[projectID]; ok {
		return s, nil
	}

	project, ok := h.store.GetProject(projectID)
	if !ok {
		return nil, fmt.Errorf("project %s not found", projectID)
	}

	_, bookmarkStore, err := migration.EnsureProjectStores(
		project.Path,
		h.cfg.NotesFilePath(),
		h.cfg.BookmarksFilePath(),
		projectID,
	)
	if err != nil {
		return nil, err
	}

	h.projectBookmarkStores[projectID] = bookmarkStore

	// Also cache note store if not present
	if _, exists := h.projectNoteStores[projectID]; !exists {
		noteStore, _, err := migration.EnsureProjectStores(
			project.Path,
			h.cfg.NotesFilePath(),
			h.cfg.BookmarksFilePath(),
			projectID,
		)
		if err != nil {
			log.Printf("Warning: failed to init project note store for %s: %v", projectID, err)
		} else {
			h.projectNoteStores[projectID] = noteStore
		}
	}

	return bookmarkStore, nil
}
