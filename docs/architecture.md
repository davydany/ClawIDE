# Architecture

This document describes the system architecture of ClawIDE, including the request flow, key design decisions, and package responsibilities.

## High-Level Overview

```
┌─────────────────────────────────────────────────────────┐
│                      Browser                            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ HTMX     │  │ xterm.js │  │ Alpine.js│              │
│  │ (UI)     │  │ (term)   │  │ (state)  │              │
│  └────┬─────┘  └────┬─────┘  └──────────┘              │
│       │ HTTP         │ WebSocket                        │
└───────┼──────────────┼──────────────────────────────────┘
        │              │
┌───────┼──────────────┼──────────────────────────────────┐
│       ▼              ▼              ClawIDE Binary       │
│  ┌─────────────────────────────┐                        │
│  │        chi Router           │                        │
│  │  middleware: Logger,        │                        │
│  │  Recoverer, Compress,      │                        │
│  │  HTMXDetect                 │                        │
│  └─────────┬───────────────────┘                        │
│            │                                            │
│  ┌─────────▼───────────────────┐                        │
│  │       Handlers              │                        │
│  │  dashboard, project,        │                        │
│  │  session, terminal, pane,   │                        │
│  │  filebrowser, docker,       │                        │
│  │  git, ports, settings       │                        │
│  └──┬────────┬────────┬────────┘                        │
│     │        │        │                                 │
│     ▼        ▼        ▼                                 │
│  ┌──────┐ ┌──────┐ ┌──────────────┐                    │
│  │Store │ │Model │ │Template      │                     │
│  │(JSON)│ │      │ │Renderer      │                     │
│  └──────┘ └──────┘ └──────────────┘                     │
│     │                                                   │
│     ▼                                                   │
│  ┌──────────────────────────────┐                       │
│  │     Backend Services         │                       │
│  │  pty/    - PTY sessions      │                       │
│  │  tmux/   - tmux wrapper      │                       │
│  │  docker/ - compose CLI       │                       │
│  │  git/    - worktrees/branch  │                       │
│  │  portdetect/ - port scanner  │                       │
│  └──────────────────────────────┘                       │
└─────────────────────────────────────────────────────────┘
```

## Request Flow

### HTTP (HTMX)

```
1. Browser sends HTTP request (often with HX-Request header)
2. chi router matches route
3. Middleware chain executes:
   - Logger: request logging
   - Recoverer: panic recovery
   - Compress: gzip response compression
   - HTMXDetect: sets IsHTMX flag in request context
   - ProjectLoader (on /projects/{id}/*): loads project from store into context
4. Handler processes request:
   - Reads models from store
   - Performs business logic
   - Renders template via tmpl.Renderer
5. Renderer checks HTMX flag:
   - HTMX request: returns partial HTML (just the component)
   - Full request: wraps partial in base layout (head, nav, scripts)
6. Browser receives HTML; HTMX swaps it into the DOM
```

### WebSocket (Terminal, Docker Logs)

```
1. Browser opens WebSocket at /ws/terminal/{sessionID}/{paneID}
2. Handler upgrades connection via gorilla/websocket
3. PTY session is retrieved (or created) from pty.Manager
4. Bidirectional streaming:
   - Browser → Server: keyboard input written to PTY stdin
   - Server → Browser: PTY stdout read and sent as binary frames
5. On disconnect: WebSocket closed, PTY session kept alive for reconnection
```

## Key Design Decisions

### tmux as Terminal Backend

Terminal sessions are backed by tmux rather than raw PTY processes. This provides:

- **Session persistence**: Sessions survive server restarts and browser disconnects.
- **Pane management**: tmux handles split panes, resize, and layout natively.
- **Process isolation**: Each terminal runs in its own tmux pane with independent state.

The `internal/tmux/` package wraps tmux commands, and `internal/pty/` manages the PTY I/O bridge between tmux panes and WebSocket connections.

### Embedded Assets via `go:embed`

All static files (CSS, JS, vendor libraries) and HTML templates are embedded into the Go binary using `go:embed`. This means:

- **Single binary deployment**: No external file dependencies at runtime.
- **Immutable assets**: Assets match the build exactly.
- **Simple distribution**: Just copy and run the binary.

The `web/embed.go` file declares the embedded filesystems, which are consumed by the static file server and template renderer.

### HTMX for Server-Driven UI

The frontend uses HTMX for most UI interactions instead of a JavaScript SPA framework. Benefits:

- **Server-rendered HTML**: Go templates produce the UI; no client-side rendering framework.
- **Partial updates**: HTMX requests receive only the HTML fragment that changed.
- **Progressive enhancement**: Pages work as full navigations and as partial swaps.
- **Simplicity**: No build-time JSX, no virtual DOM, no hydration.

Alpine.js handles local UI state (dropdowns, modals, toggles) where HTMX alone isn't sufficient.

### Binary Pane Tree

Terminal panes within a session are modeled as a binary tree in `internal/model/pane.go`. Each split creates two child nodes (horizontal or vertical), enabling nested layouts. The tree structure maps directly to tmux's pane model and allows the frontend to render the layout recursively.

### JSON File Store

State is persisted to a JSON file (`~/.clawide/state.json`) rather than a database. For a single-user local tool, this is sufficient and avoids external dependencies. The `internal/store/` package provides thread-safe read/write operations.

### Single-Instance Enforcement

ClawIDE writes a PID file to `~/.clawide/clawide.pid` on startup and checks for an existing running instance. If one is found, the new process exits with an error unless `--restart` is passed, which kills the existing instance first. This prevents port conflicts and state corruption from multiple instances.

## Package Responsibilities

| Package | Purpose |
|---------|---------|
| `cmd/clawide` | Entry point: loads config, initializes store/renderer/server, handles graceful shutdown |
| `internal/config` | Loads configuration from `~/.clawide/config.json`, environment variables, and CLI flags (in that precedence order) |
| `internal/server` | Creates the HTTP server, registers routes on the chi mux, manages server lifecycle |
| `internal/handler` | Request handlers for all HTTP endpoints and WebSocket upgrades. One file per domain area. |
| `internal/middleware` | HTTP middleware: HTMX header detection, project context loading from store |
| `internal/model` | Domain types: `Project`, `Session`, `Pane` (binary tree), `DockerService` |
| `internal/store` | Thread-safe JSON file persistence for projects and sessions |
| `internal/pty` | PTY session lifecycle, I/O fan-out to multiple WebSocket consumers, scrollback ring buffer |
| `internal/tmux` | tmux CLI wrapper for session/window/pane operations |
| `internal/docker` | Docker Compose CLI wrapper (`ps`, `up`, `down`, service control) and `docker-compose.yml` parser |
| `internal/git` | Git operations: branch listing, worktree create/list/delete |
| `internal/portdetect` | Port discovery via OS tools (`lsof`/`ss`) and docker-compose YAML port extraction |
| `internal/pidfile` | PID file read/write/check for single-instance enforcement |
| `internal/tmpl` | Go `html/template` renderer with HTMX-aware partial/full page rendering |
| `web` | `go:embed` declarations for static assets and templates |

## Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/go-chi/chi/v5` | HTTP router and middleware |
| `github.com/gorilla/websocket` | WebSocket connections for terminals and log streaming |
| `github.com/creack/pty` | PTY allocation for terminal sessions |
| `github.com/google/uuid` | UUID generation for project/session identifiers |
| `golang.org/x/text` | Text processing utilities |
| `gopkg.in/yaml.v3` | Docker Compose YAML parsing |
