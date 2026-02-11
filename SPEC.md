# CCMux - Claude Code Multiplexer Specification

## Overview
Single Go binary serving a mobile-first web UI for managing multiple Claude Code sessions across projects via git worktrees, with Docker integration, file editing, and terminal access.

## Implementation Checklist

### Phase 1: Foundation
- [X] Initialize Go module, project directory structure
- [X] `internal/config/config.go` — config loading (file + env + flags)
- [X] `internal/store/store.go` — JSON state persistence
- [X] `internal/model/` — domain models (project, session, docker)
- [X] `internal/server/server.go` — chi router, middleware, graceful shutdown
- [X] `internal/server/routes.go` — route registration
- [X] `internal/middleware/` — htmx detection, project context
- [X] `internal/tmpl/renderer.go` — template renderer with htmx partial/full
- [X] `web/embed.go` — go:embed for static + templates
- [X] `web/templates/` — base.html, layouts, pages, components
- [X] `web/static/` — Tailwind CSS, vendored JS (htmx, alpine)
- [X] `cmd/ccmux/main.go` — wire everything, graceful shutdown
- [X] `Makefile` — build, dev, vendor targets
- [X] `Dockerfile` + `docker-compose.yml` — local dev environment
- [X] `.gitignore`
- [X] Verify: `make build` produces binary, serves empty dashboard at :9800

### Phase 2: Project Management
- [X] `internal/handler/dashboard.go` — project list page
- [X] `internal/handler/project.go` — project CRUD
- [X] Project list UI (cards, mobile/desktop layouts)
- [X] New project modal/form
- [X] Project workspace page (tabbed layout, empty tabs)
- [X] Verify: browse ~/projects, create projects, navigate to workspace

### Phase 3: Terminal Sessions
- [X] `internal/pty/session.go` — PTY process, I/O fan-out, scrollback ring buffer
- [X] `internal/pty/manager.go` — session registry, lifecycle, cleanup
- [X] `internal/handler/terminal.go` — WebSocket handler
- [X] `internal/handler/session.go` — session CRUD API
- [X] `web/src/xterm-entry.js` — esbuild bundle for xterm.js
- [X] `web/static/js/terminal.js` — xterm.js init, WebSocket bridge, resize, reconnect
- [X] Terminal tab UI (session tabs, new session button)
- [X] Verify: Claude Code in embedded terminal, multiple sessions, survive tab close

### Phase 4: Git Worktrees
- [X] `internal/git/repository.go` — branch listing, status
- [X] `internal/git/worktree.go` — worktree CRUD
- [X] Session creation flow: pick branch/worktree
- [X] Worktree management UI in project settings
- [X] Verify: create sessions on different branches via worktrees

### Phase 5: File Browser + Editor
- [X] `internal/handler/filebrowser.go` — directory listing, file read/write
- [X] `web/src/codemirror-entry.js` — esbuild bundle for CodeMirror 6
- [X] `web/static/js/editor.js` — CM6 init, file load/save, language detection
- [X] File tree component (lazy-loaded directories via htmx)
- [X] Editor toolbar (filename, save, modified indicator)
- [X] Mobile: full-screen tree -> full-screen editor flow
- [X] Desktop: side-by-side tree + editor with resizable divider
- [X] Verify: browse files, edit with syntax highlighting, save

### Phase 6: Docker Integration
- [X] `internal/docker/compose.go` — CLI wrapper (ps, up, down, start, stop, restart)
- [X] `internal/docker/parser.go` — docker-compose.yml YAML parser
- [X] `internal/handler/docker.go` — REST endpoints + log streaming WebSocket
- [X] `web/static/js/docker.js` — log viewer, service controls
- [X] Docker panel UI (service list with status badges, log viewer)
- [X] Verify: view services, start/stop/restart, stream logs

### Phase 7: Port Detection + Polish
- [X] `internal/portdetect/scanner.go` — OS port scanner (lsof/ss)
- [X] `internal/portdetect/compose.go` — port extraction from compose YAML
- [X] `internal/handler/ports.go` — port detection endpoint
- [X] Ports panel UI (cards with clickable links)
- [X] `internal/handler/settings.go` — settings page/API
- [X] Settings page UI
- [X] Mobile polish: touch targets, viewport handling, bottom tab bar
- [X] Desktop polish: resizable panels, keyboard shortcuts
- [X] Verify: full working application
