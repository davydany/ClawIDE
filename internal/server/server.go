package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/davydany/ccmux/internal/config"
	"github.com/davydany/ccmux/internal/handler"
	"github.com/davydany/ccmux/internal/pty"
	"github.com/davydany/ccmux/internal/store"
	"github.com/davydany/ccmux/internal/tmpl"
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
	ptyMgr := pty.NewManager(cfg.MaxSessions, cfg.ScrollbackSize, cfg.ClaudeCommand)

	s := &Server{
		cfg:        cfg,
		store:      st,
		renderer:   renderer,
		ptyManager: ptyMgr,
		handlers:   handler.New(cfg, st, renderer, ptyMgr),
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

func (s *Server) Start() error {
	log.Printf("Starting CCMux on http://%s", s.cfg.Addr())
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
