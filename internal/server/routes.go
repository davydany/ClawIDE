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

	// System stats
	r.Get("/api/system/stats", s.handlers.SystemStats)

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
			r.Patch("/star", s.handlers.ToggleStar)
			r.Patch("/color", s.handlers.UpdateProjectColor)

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
			r.Patch("/panes/{pid}/rename", s.handlers.RenamePane)
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
			r.Post("/api/pull-main", s.handlers.PullMain)

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
				r.Post("/api/pull-main", s.handlers.FeaturePullMain)
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

	// Notes API (global + project-scoped via query param)
	r.Route("/api/notes", func(r chi.Router) {
		r.Get("/", s.handlers.ListNotes)
		r.Post("/", s.handlers.CreateNote)
		r.Put("/{noteID}", s.handlers.UpdateNote)
		r.Delete("/{noteID}", s.handlers.DeleteNote)
	})

	// Bookmarks API (project-scoped via query param)
	r.Route("/api/bookmarks", func(r chi.Router) {
		r.Get("/", s.handlers.ListBookmarks)
		r.Post("/", s.handlers.CreateBookmark)
		r.Put("/{bookmarkID}", s.handlers.UpdateBookmark)
		r.Delete("/{bookmarkID}", s.handlers.DeleteBookmark)
		r.Patch("/{bookmarkID}/star", s.handlers.ToggleBookmarkStar)
	})

	// Voice Box API (global, not project-scoped)
	r.Route("/api/voicebox", func(r chi.Router) {
		r.Get("/", s.handlers.ListVoiceBoxEntries)
		r.Post("/", s.handlers.CreateVoiceBoxEntry)
		r.Delete("/", s.handlers.DeleteAllVoiceBoxEntries)
		r.Delete("/{entryID}", s.handlers.DeleteVoiceBoxEntry)
	})

	// Notifications API (global, not project-scoped)
	r.Route("/api/notifications", func(r chi.Router) {
		r.Post("/", s.handlers.CreateNotification)
		r.Get("/", s.handlers.ListNotifications)
		r.Get("/unread-count", s.handlers.UnreadNotificationCount)
		r.Get("/stream", s.handlers.NotificationStream)
		r.Patch("/{notifID}/read", s.handlers.MarkNotificationRead)
		r.Post("/read-all", s.handlers.MarkAllNotificationsRead)
		r.Delete("/{notifID}", s.handlers.DeleteNotification)
	})

	// Claude Code integration API
	r.Get("/api/claude/detect", s.handlers.DetectClaudeCLI)
	r.Post("/api/claude/setup-hook", s.handlers.SetupClaudeHook)
	r.Delete("/api/claude/hook", s.handlers.RemoveClaudeHook)

	// WebSocket endpoints (no project middleware, session ID is in URL)
	r.Get("/ws/terminal/{sessionID}/{paneID}", s.handlers.TerminalWS)
	r.Get("/ws/docker/{projectID}/logs/{svc}", s.handlers.DockerLogsWS)

	return r
}
