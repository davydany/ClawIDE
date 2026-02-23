package handler

import (
	"fmt"
	"log"
	"sync"

	"github.com/davydany/ClawIDE/internal/config"
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

	// Project-scoped stores, keyed by projectID. Lazily initialized.
	projectNoteStores     map[string]*store.ProjectNoteStore
	projectBookmarkStores map[string]*store.ProjectBookmarkStore
	projectStoreMu        sync.Mutex
}

func New(cfg *config.Config, st *store.Store, renderer *tmpl.Renderer, ptyMgr *ptyPkg.Manager, snippetSt *store.SnippetStore, notifSt *store.NotificationStore, noteSt *store.NoteStore, bookmarkSt *store.BookmarkStore, voiceBoxSt *store.VoiceBoxStore, scratchpadSt *store.ScratchpadStore, hub *sse.Hub, upd *updater.Updater, wizJobs *wizard.JobTracker, wizGen *wizard.Generator) *Handlers {
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
		sseHub:                hub,
		updater:               upd,
		wizardJobs:            wizJobs,
		wizardGenerator:       wizGen,
		projectNoteStores:     make(map[string]*store.ProjectNoteStore),
		projectBookmarkStores: make(map[string]*store.ProjectBookmarkStore),
	}
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
