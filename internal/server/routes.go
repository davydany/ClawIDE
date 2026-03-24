package server

import (
	"io"
	"io/fs"
	"net/http"
	"time"

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

	// Favicon at root — browsers request /favicon.ico automatically
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		f, err := staticFS.Open("favicon.ico")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "image/x-icon")
		w.Header().Set("Cache-Control", "public, max-age=604800")
		http.ServeContent(w, r, "favicon.ico", time.Time{}, f.(io.ReadSeeker))
	})

	// Version
	r.Get("/api/version", s.handlers.Version)

	// Update
	r.Post("/api/update/check", s.handlers.CheckForUpdate)
	r.Get("/api/update/status", s.handlers.UpdateStatus)
	r.Post("/api/update/apply", s.handlers.ApplyUpdate)

	// System stats
	r.Get("/api/system/stats", s.handlers.SystemStats)

	// Dashboard
	r.Get("/", s.handlers.Dashboard)

	// Settings
	r.Get("/settings", s.handlers.SettingsPage)
	r.Put("/api/settings", s.handlers.UpdateSettings)

	// AI Settings
	r.Get("/api/settings/ai", s.handlers.GetAISettings)
	r.Put("/api/settings/ai", s.handlers.SetAISettings)
	r.Post("/api/settings/ai/verify", s.handlers.VerifyAICredentials)

	// Onboarding
	r.Post("/api/onboarding/complete", s.handlers.CompleteOnboarding)
	r.Post("/api/onboarding/workspace-tour-complete", s.handlers.CompleteWorkspaceTour)
	r.Post("/api/onboarding/reset", s.handlers.ResetOnboarding)

	// Project routes
	r.Route("/projects", func(r chi.Router) {
		r.Get("/", s.handlers.ListProjects)
		r.Post("/", s.handlers.CreateProject)
		r.Post("/reorder", s.handlers.ReorderProjects)

		// Wizard routes (before /{id} to avoid conflict)
		r.Get("/wizard", s.handlers.ShowWizard)
		r.Get("/wizard/languages", s.handlers.GetWizardLanguages)
		r.Get("/wizard/providers", s.handlers.GetWizardProviders)
		r.Get("/wizard/models", s.handlers.GetWizardModels)
		r.Post("/wizard/create", s.handlers.CreateProjectFromWizard)
		r.Get("/wizard/status/{jobID}", s.handlers.GetWizardStatus)
		r.Post("/wizard/validate", s.handlers.ValidateWizardField)

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

			// Skills API
			r.Get("/api/skills", s.handlers.ListSkills)
			r.Post("/api/skills", s.handlers.CreateSkill)
			r.Get("/api/skills/{scope}/{skillName}", s.handlers.GetSkill)
			r.Put("/api/skills/{scope}/{skillName}", s.handlers.UpdateSkill)
			r.Delete("/api/skills/{scope}/{skillName}", s.handlers.DeleteSkill)
			r.Post("/api/skills/{scope}/{skillName}/move", s.handlers.MoveSkill)

			// MCP Servers API
			r.Get("/api/mcp-servers", s.handlers.ListMCPServers)
			r.Post("/api/mcp-servers", s.handlers.CreateMCPServer)
			r.Get("/api/mcp-servers/{scope}/{serverName}", s.handlers.GetMCPServer)
			r.Put("/api/mcp-servers/{scope}/{serverName}", s.handlers.UpdateMCPServer)
			r.Delete("/api/mcp-servers/{scope}/{serverName}", s.handlers.DeleteMCPServer)
			r.Post("/api/mcp-servers/{scope}/{serverName}/move", s.handlers.MoveMCPServer)
			r.Post("/api/mcp-servers/{scope}/{serverName}/start", s.handlers.StartMCPServer)
			r.Post("/api/mcp-servers/{scope}/{serverName}/stop", s.handlers.StopMCPServer)
			r.Post("/api/mcp-servers/{scope}/{serverName}/restart", s.handlers.RestartMCPServer)
			r.Get("/api/mcp-servers/{scope}/{serverName}/logs", s.handlers.MCPServerLogs)
			r.Get("/api/mcp-servers/{scope}/{serverName}/status", s.handlers.MCPServerStatus)

			// Agents API
			r.Get("/api/agents", s.handlers.ListAgents)
			r.Post("/api/agents", s.handlers.CreateAgent)
			r.Get("/api/agents/{scope}/{agentName}", s.handlers.GetAgent)
			r.Put("/api/agents/{scope}/{agentName}", s.handlers.UpdateAgent)
			r.Delete("/api/agents/{scope}/{agentName}", s.handlers.DeleteAgent)
			r.Post("/api/agents/{scope}/{agentName}/move", s.handlers.MoveAgent)

			// File browser API
			r.Get("/api/files", s.handlers.ListFiles)
			r.Get("/api/file", s.handlers.ReadFile)
			r.Put("/api/file", s.handlers.WriteFile)
			r.Post("/api/mkdir", s.handlers.Mkdir)
			r.Post("/api/rename", s.handlers.RenameFile)
			r.Delete("/api/file", s.handlers.DeleteFile)
			r.Get("/api/files/search", s.handlers.SearchFiles)

			// Docker API
			r.Get("/api/docker/status", s.handlers.DockerStatus)
			r.Get("/api/docker/ps", s.handlers.DockerPS)
			r.Post("/api/docker/up", s.handlers.DockerUp)
			r.Post("/api/docker/down", s.handlers.DockerDown)
			r.Post("/api/docker/restart", s.handlers.DockerRestart)
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
			r.Get("/api/remotes", s.handlers.ListRemotes)
			r.Post("/api/base-branch", s.handlers.SetBaseBranch)

			// Feature routes
			r.Post("/features/", s.handlers.CreateFeature)
			r.Route("/features/{fid}", func(r chi.Router) {
				r.Get("/", s.handlers.FeatureWorkspace)
				r.Delete("/", s.handlers.DeleteFeature)
				r.Patch("/color", s.handlers.UpdateFeatureColor)

				// Feature sessions
				r.Post("/sessions/", s.handlers.CreateFeatureSession)

				// Feature file browser
				r.Get("/api/files", s.handlers.FeatureListFiles)
				r.Get("/api/file", s.handlers.FeatureReadFile)
				r.Put("/api/file", s.handlers.FeatureWriteFile)
				r.Post("/api/mkdir", s.handlers.FeatureMkdir)
				r.Post("/api/rename", s.handlers.FeatureRenameFile)
				r.Delete("/api/file", s.handlers.FeatureDeleteFile)
				r.Get("/api/files/search", s.handlers.FeatureSearchFiles)

				// Feature git operations
				r.Get("/api/status", s.handlers.FeatureGitStatus)
				r.Post("/api/commit", s.handlers.FeatureGitCommit)
				r.Post("/api/merge", s.handlers.FeatureMerge)
				r.Post("/api/pull-main", s.handlers.FeaturePullMain)

				// Feature merge review
				r.Get("/api/review/files", s.handlers.FeatureReviewFiles)
				r.Get("/api/review/file-content", s.handlers.FeatureReviewFileContent)
				r.Get("/api/review/annotations", s.handlers.FeatureReviewAnnotations)

				// Feature Docker API
				r.Get("/api/docker/status", s.handlers.FeatureDockerStatus)
				r.Get("/api/docker/ps", s.handlers.FeatureDockerPS)
				r.Post("/api/docker/up", s.handlers.FeatureDockerUp)
				r.Post("/api/docker/down", s.handlers.FeatureDockerDown)
				r.Post("/api/docker/restart", s.handlers.FeatureDockerRestart)
				r.Post("/api/docker/{svc}/start", s.handlers.FeatureDockerServiceStart)
				r.Post("/api/docker/{svc}/stop", s.handlers.FeatureDockerServiceStop)
				r.Post("/api/docker/{svc}/restart", s.handlers.FeatureDockerServiceRestart)
				r.Post("/api/docker/copy-env-files", s.handlers.FeatureDockerCopyEnvFiles)
			})
		})
	})

	// Scratchpad API (global)
	r.Get("/api/scratchpad", s.handlers.GetScratchpad)
	r.Put("/api/scratchpad", s.handlers.UpdateScratchpad)

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
		r.Post("/reorder", s.handlers.ReorderNotes)

		// Note git operations
		r.Get("/git-status", s.handlers.NoteGitStatus)
		r.Post("/commit", s.handlers.NoteGitCommit)

		// Note folders
		r.Route("/folders", func(r chi.Router) {
			r.Get("/", s.handlers.ListNoteFolders)
			r.Post("/", s.handlers.CreateNoteFolder)
			r.Put("/{folderID}", s.handlers.UpdateNoteFolder)
			r.Delete("/{folderID}", s.handlers.DeleteNoteFolder)
		})
	})

	// Bookmarks API (project-scoped via query param)
	r.Route("/api/bookmarks", func(r chi.Router) {
		r.Get("/", s.handlers.ListBookmarks)
		r.Post("/", s.handlers.CreateBookmark)
		r.Put("/{bookmarkID}", s.handlers.UpdateBookmark)
		r.Delete("/{bookmarkID}", s.handlers.DeleteBookmark)
		r.Post("/reorder", s.handlers.ReorderBookmarks)

		// Bookmark git operations
		r.Get("/git-status", s.handlers.BookmarkGitStatus)
		r.Post("/commit", s.handlers.BookmarkGitCommit)

		// Bookmark folders
		r.Route("/folders", func(r chi.Router) {
			r.Get("/", s.handlers.ListBookmarkFolders)
			r.Post("/", s.handlers.CreateBookmarkFolder)
			r.Put("/{folderID}", s.handlers.UpdateBookmarkFolder)
			r.Delete("/{folderID}", s.handlers.DeleteBookmarkFolder)
		})
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

	// Editor integration API
	r.Get("/api/editors/available", s.handlers.AvailableEditors)
	r.Post("/api/editor/open", s.handlers.OpenEditor)
	r.Post("/api/editor/open-folder", s.handlers.OpenFolder)

	// WebSocket endpoints (no project middleware, session ID is in URL)
	r.Get("/ws/terminal/{sessionID}/{paneID}", s.handlers.TerminalWS)
	r.Get("/ws/docker/{projectID}/logs/{svc}", s.handlers.DockerLogsWS)
	r.Get("/ws/docker/{projectID}/build/{svc}", s.handlers.DockerBuildWS)
	r.Get("/ws/docker/{projectID}/features/{fid}/logs/{svc}", s.handlers.FeatureDockerLogsWS)
	r.Get("/ws/docker/{projectID}/features/{fid}/build/{svc}", s.handlers.FeatureDockerBuildWS)

	return r
}
