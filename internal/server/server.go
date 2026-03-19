package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/banner"
	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/handler"
	"github.com/davydany/ClawIDE/internal/migration"
	"github.com/davydany/ClawIDE/internal/pty"
	"github.com/davydany/ClawIDE/internal/sse"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/davydany/ClawIDE/internal/tmpl"
	"github.com/davydany/ClawIDE/internal/tmux"
	"github.com/davydany/ClawIDE/internal/updater"
	"github.com/davydany/ClawIDE/internal/version"
	"github.com/davydany/ClawIDE/internal/wizard"
)

type Server struct {
	cfg        *config.Config
	store      *store.Store
	renderer   *tmpl.Renderer
	ptyManager *pty.Manager
	handlers   *handler.Handlers
	http       *http.Server
	updater    *updater.Updater
}

func New(cfg *config.Config, st *store.Store, renderer *tmpl.Renderer) *Server {
	// Configure and check multiplexer dependency
	tmux.SetBinary(cfg.Multiplexer)
	if err := tmux.Check(); err != nil {
		log.Fatalf("%s is required: %v", tmux.Binary(), err)
	}
	log.Printf("%s check passed", tmux.Binary())

	ptyMgr := pty.NewManager(cfg.MaxSessions, cfg.ScrollbackSize, cfg.AgentCommand)

	snippetStore, err := store.NewSnippetStore(cfg.SnippetsFilePath())
	if err != nil {
		log.Fatalf("failed to load snippet store: %v", err)
	}

	notificationStore, err := store.NewNotificationStore(cfg.NotificationsFilePath(), cfg.MaxNotifications)
	if err != nil {
		log.Fatalf("failed to load notification store: %v", err)
	}

	noteStore, err := store.NewNoteStore(cfg.NotesFilePath())
	if err != nil {
		log.Fatalf("failed to load note store: %v", err)
	}

	bookmarkStore, err := store.NewBookmarkStore(cfg.BookmarksFilePath())
	if err != nil {
		log.Fatalf("failed to load bookmark store: %v", err)
	}

	voiceBoxStore, err := store.NewVoiceBoxStore(cfg.VoiceBoxFilePath(), 50)
	if err != nil {
		log.Fatalf("failed to load voicebox store: %v", err)
	}

	scratchpadStore, err := store.NewScratchpadStore(cfg.ScratchpadFilePath())
	if err != nil {
		log.Fatalf("failed to load scratchpad store: %v", err)
	}

	sseHub := sse.NewHub()

	// Backfill ActiveBranch for projects that don't have one set
	migration.BackfillActiveBranch(st)

	// Recover tmux sessions from previous run
	recoverTmuxSessions(st)

	upd := updater.New(cfg, notificationStore, sseHub)

	// Initialize wizard components
	wizardJobs := wizard.NewJobTracker()
	tmplRegistry, err := wizard.NewTemplateRegistry(wizard.TemplatesFS)
	if err != nil {
		log.Fatalf("failed to load wizard templates: %v", err)
	}
	wizardGen := wizard.NewGenerator(tmplRegistry, wizardJobs)

	s := &Server{
		cfg:        cfg,
		store:      st,
		renderer:   renderer,
		ptyManager: ptyMgr,
		handlers:   handler.New(cfg, st, renderer, ptyMgr, snippetStore, notificationStore, noteStore, bookmarkStore, voiceBoxStore, scratchpadStore, sseHub, upd, wizardJobs, wizardGen),
		updater:    upd,
	}

	router := s.setupRoutes()

	s.http = &http.Server{
		Addr:         cfg.Addr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	upd.Start()

	return s
}

// recoverTmuxSessions cleans up orphan tmux sessions that are not referenced in state.json.
func recoverTmuxSessions(st *store.Store) {
	tmuxSessions, err := tmux.ListClawIDESessions()
	if err != nil {
		log.Printf("Warning: could not list %s sessions: %v", tmux.Binary(), err)
		return
	}

	if len(tmuxSessions) == 0 {
		return
	}

	muxName := tmux.Binary()

	// Build set of all valid pane IDs from stored sessions
	validPanes := make(map[string]bool)
	allSessions := st.GetAllSessions()
	for _, sess := range allSessions {
		if sess.Layout != nil {
			for _, paneID := range sess.Layout.CollectLeaves() {
				validPanes["clawide-"+paneID] = true
			}
		}
	}

	// Kill orphan multiplexer sessions (not referenced in state.json)
	surviving := 0
	for _, tmuxSess := range tmuxSessions {
		if validPanes[tmuxSess] {
			paneID := strings.TrimPrefix(tmuxSess, "clawide-")
			log.Printf("Surviving %s session: %s (pane %s) — will reconnect lazily", muxName, tmuxSess, paneID)
			surviving++
		} else {
			log.Printf("Killing orphan %s session: %s", muxName, tmuxSess)
			if err := tmux.KillSession(tmuxSess); err != nil {
				log.Printf("Warning: failed to kill orphan %s session %s: %v", muxName, tmuxSess, err)
			}
		}
	}

	if surviving > 0 {
		log.Printf("Found %d surviving %s sessions (will reconnect on WebSocket connect)", surviving, muxName)
	}
}

func (s *Server) Start() error {
	banner.Print(s.cfg.Host, s.cfg.Port, version.String())
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	s.updater.Stop()
	s.ptyManager.CloseAll()
	return s.http.Shutdown(ctx)
}
