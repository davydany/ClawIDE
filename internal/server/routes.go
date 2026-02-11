package server

import (
	"io/fs"
	"net/http"

	"github.com/davydany/ccmux/internal/middleware"
	"github.com/davydany/ccmux/web"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func (s *Server) setupRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Compress(5))
	r.Use(middleware.HTMXDetect)

	// Static files
	staticFS, _ := fs.Sub(web.StaticFS, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Dashboard
	r.Get("/", s.handlers.Dashboard)

	// Settings
	r.Get("/settings", s.handlers.SettingsPage)
	r.Put("/api/settings", s.handlers.UpdateSettings)

	// Project routes
	r.Route("/projects", func(r chi.Router) {
		r.Get("/", s.handlers.ListProjects)
		r.Post("/", s.handlers.CreateProject)

		r.Route("/{id}", func(r chi.Router) {
			r.Use(middleware.ProjectLoader(s.store))

			r.Get("/", s.handlers.ProjectWorkspace)
			r.Delete("/", s.handlers.DeleteProject)

			// Sessions
			r.Get("/sessions/", s.handlers.ListSessions)
			r.Post("/sessions/", s.handlers.CreateSession)
			r.Delete("/sessions/{sid}", s.handlers.DeleteSession)

			// File browser API
			r.Get("/api/files", s.handlers.ListFiles)
			r.Get("/api/file", s.handlers.ReadFile)
			r.Put("/api/file", s.handlers.WriteFile)

			// Docker API
			r.Get("/api/docker/ps", s.handlers.DockerPS)
			r.Post("/api/docker/up", s.handlers.DockerUp)
			r.Post("/api/docker/down", s.handlers.DockerDown)
			r.Post("/api/docker/{svc}/start", s.handlers.DockerServiceStart)
			r.Post("/api/docker/{svc}/stop", s.handlers.DockerServiceStop)
			r.Post("/api/docker/{svc}/restart", s.handlers.DockerServiceRestart)

			// Git API
			r.Get("/api/worktrees", s.handlers.ListWorktrees)
			r.Post("/api/worktrees", s.handlers.CreateWorktree)
			r.Delete("/api/worktrees/{wid}", s.handlers.DeleteWorktree)
			r.Get("/api/branches", s.handlers.ListBranches)

			// Port detection
			r.Get("/api/ports", s.handlers.DetectPorts)
		})
	})

	// WebSocket endpoints (no project middleware, session ID is in URL)
	r.Get("/ws/terminal/{sessionID}", s.handlers.TerminalWS)
	r.Get("/ws/docker/{projectID}/logs/{svc}", s.handlers.DockerLogsWS)

	return r
}
