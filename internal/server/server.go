package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/config"
	"github.com/davydany/ClawIDE/internal/handler"
	"github.com/davydany/ClawIDE/internal/pty"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/davydany/ClawIDE/internal/tmpl"
	"github.com/davydany/ClawIDE/internal/tmux"
)

type Server struct {
	cfg        *config.Config
	store      *store.Store
	renderer   *tmpl.Renderer
	ptyManager *pty.Manager
	handlers   *handler.Handlers
	http       *http.Server
}

func New(cfg *config.Config, st *store.Store, renderer *tmpl.Renderer) *Server {
	// Check tmux dependency
	if err := tmux.Check(); err != nil {
		log.Fatalf("tmux is required: %v", err)
	}
	log.Println("tmux check passed")

	ptyMgr := pty.NewManager(cfg.MaxSessions, cfg.ScrollbackSize, cfg.ClaudeCommand)

	snippetStore, err := store.NewSnippetStore(cfg.SnippetsFilePath())
	if err != nil {
		log.Fatalf("failed to load snippet store: %v", err)
	}

	// Recover tmux sessions from previous run
	recoverTmuxSessions(st)

	s := &Server{
		cfg:        cfg,
		store:      st,
		renderer:   renderer,
		ptyManager: ptyMgr,
		handlers:   handler.New(cfg, st, renderer, ptyMgr, snippetStore),
	}

	router := s.setupRoutes()

	s.http = &http.Server{
		Addr:         cfg.Addr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// recoverTmuxSessions cleans up orphan tmux sessions that are not referenced in state.json.
func recoverTmuxSessions(st *store.Store) {
	tmuxSessions, err := tmux.ListClawIDESessions()
	if err != nil {
		log.Printf("Warning: could not list tmux sessions: %v", err)
		return
	}

	if len(tmuxSessions) == 0 {
		return
	}

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

	// Kill orphan tmux sessions (not referenced in state.json)
	surviving := 0
	for _, tmuxSess := range tmuxSessions {
		if validPanes[tmuxSess] {
			paneID := strings.TrimPrefix(tmuxSess, "clawide-")
			log.Printf("Surviving tmux session: %s (pane %s) — will reconnect lazily", tmuxSess, paneID)
			surviving++
		} else {
			log.Printf("Killing orphan tmux session: %s", tmuxSess)
			if err := tmux.KillSession(tmuxSess); err != nil {
				log.Printf("Warning: failed to kill orphan tmux session %s: %v", tmuxSess, err)
			}
		}
	}

	if surviving > 0 {
		log.Printf("Found %d surviving tmux sessions (will reconnect on WebSocket connect)", surviving)
	}
}

func (s *Server) Start() error {
	log.Printf("Starting ClawIDE on http://%s", s.cfg.Addr())
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	s.ptyManager.CloseAll()
	return s.http.Shutdown(ctx)
}
