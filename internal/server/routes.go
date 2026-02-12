package server

import (
	"io/fs"
	"net/http"

	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/web"
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

	// Version
	r.Get("/api/version", s.handlers.Version)

	// Dashboard
	r.Get("/", s.handlers.Dashboard)

	// Settings
	r.Get("/settings", s.handlers.SettingsPage)
	r.Put("/api/settings", s.handlers.UpdateSettings)

	// Onboarding
	r.Post("/api/onboarding/complete", s.handlers.CompleteOnboarding)
	r.Post("/api/onboarding/workspace-tour-complete", s.handlers.CompleteWorkspaceTour)
	r.Post("/api/onboarding/reset", s.handlers.ResetOnboarding)

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
			r.Route("/sessions/{sid}", func(r chi.Router) {
				r.Patch("/", s.handlers.RenameSession)
				r.Delete("/", s.handlers.DeleteSession)

				// Pane operations
				r.Post("/panes/{pid}/split", s.handlers.SplitPane)
				r.Delete("/panes/{pid}", s.handlers.ClosePane)
				r.Patch("/panes/{pid}/resize", s.handlers.ResizePane)
			})

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
			r.Post("/api/checkout", s.handlers.CheckoutBranch)
			r.Post("/api/branches", s.handlers.CreateBranch)

			// Port detection
			r.Get("/api/ports", s.handlers.DetectPorts)

			// Feature routes
			r.Post("/features/", s.handlers.CreateFeature)
			r.Route("/features/{fid}", func(r chi.Router) {
				r.Get("/", s.handlers.FeatureWorkspace)
				r.Delete("/", s.handlers.DeleteFeature)

				// Feature sessions
				r.Post("/sessions/", s.handlers.CreateFeatureSession)

				// Feature file browser
				r.Get("/api/files", s.handlers.FeatureListFiles)
				r.Get("/api/file", s.handlers.FeatureReadFile)
				r.Put("/api/file", s.handlers.FeatureWriteFile)

				// Feature git operations
				r.Get("/api/status", s.handlers.FeatureGitStatus)
				r.Post("/api/commit", s.handlers.FeatureGitCommit)
				r.Post("/api/merge", s.handlers.FeatureMerge)
			})
		})
	})

	// Snippets API (global, not project-scoped)
	r.Route("/api/snippets", func(r chi.Router) {
		r.Get("/", s.handlers.ListSnippets)
		r.Post("/", s.handlers.CreateSnippet)
		r.Put("/{snippetID}", s.handlers.UpdateSnippet)
		r.Delete("/{snippetID}", s.handlers.DeleteSnippet)
	})

	// WebSocket endpoints (no project middleware, session ID is in URL)
	r.Get("/ws/terminal/{sessionID}/{paneID}", s.handlers.TerminalWS)
	r.Get("/ws/docker/{projectID}/logs/{svc}", s.handlers.DockerLogsWS)

	return r
}
