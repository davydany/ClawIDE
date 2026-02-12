---
title: "Development Setup"
description: "Local development environment setup, build pipeline, and debugging tips for ClawIDE."
weight: 10
---

This guide covers local development setup, the build pipeline, Makefile targets, and debugging tips.

## Prerequisites

### Go 1.24+

```bash
# macOS
brew install go

# Linux — download the official tarball
# See https://go.dev/doc/install
```

### Node.js (for frontend asset builds)

```bash
# macOS
brew install node

# Or use nvm
nvm install --lts
```

### tmux

```bash
# macOS
brew install tmux

# Debian/Ubuntu
sudo apt install tmux
```

### Docker (optional)

Required only for Docker integration features and container-based development.

```bash
# See https://docs.docker.com/get-docker/
```

## Clone and Build

```bash
git clone https://github.com/davydany/ClawIDE.git
cd ClawIDE

# Full build: vendor JS deps, compile CSS, build Go binary
make all

# Run
./clawide
```

The server starts at [http://localhost:9800](http://localhost:9800).

## Development Mode

For faster iteration, use development mode which builds assets and runs with `go run` (no binary produced):

```bash
make dev
```

For CSS hot-reload during template work, run in a separate terminal:

```bash
make css-watch
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make all` | Full build: `vendor-js` + `css` + `build` |
| `make build` | Compile Go binary to `./clawide` |
| `make dev` | Build assets and run via `go run ./cmd/clawide` |
| `make run` | Build binary and run it |
| `make css` | Compile Tailwind CSS (minified) |
| `make css-watch` | Watch mode for Tailwind CSS (recompiles on changes) |
| `make vendor-js` | Download HTMX and Alpine.js, build xterm/CodeMirror bundles |
| `make clean` | Remove binary and compiled CSS |
| `make test` | Run all Go tests (`go test ./...`) |
| `make fmt` | Format Go code (`go fmt ./...`) |
| `make start` / `make up` | `docker compose up -d` |
| `make stop` / `make shutdown` | `docker compose down` |
| `make status` / `make ps` | `docker compose ps -a` |
| `make logs` | `docker compose logs -f` (use `SERVICE=foo` for a single service) |

## Frontend Build Pipeline

The frontend has three asset pipelines:

### 1. Vendored JS Libraries

HTMX and Alpine.js are downloaded once into `web/static/vendor/` by `make vendor-js`. These are committed to the repo so builds don't require network access.

### 2. esbuild Bundles

Interactive components that require npm packages (xterm.js, CodeMirror 6) are bundled with esbuild:

```text
web/src/xterm-entry.js     → web/static/dist/xterm-bundle.js
web/src/codemirror-entry.js → web/static/dist/codemirror-bundle.js
```

The esbuild config lives in `web/src/package.json`. Running `make vendor-js` installs npm dependencies and runs the build.

### 3. Tailwind CSS

```text
web/static/css/input.css → web/static/dist/app.css
```

`make css` compiles and minifies. `make css-watch` recompiles on file changes.

### Asset Embedding

All built assets under `web/static/` and templates under `web/templates/` are embedded into the Go binary via `go:embed` in `web/embed.go`. This means:

- After any frontend change, you must rebuild the binary (or use `make dev` which runs via `go run`).
- The binary is fully self-contained at runtime.

## Adding a New Feature

The typical workflow for adding a new feature:

### 1. Define the Model (if needed)

Create or extend a file in `internal/model/`:

```go
// internal/model/notification.go
package model

type Notification struct {
    ID      string `json:"id"`
    Message string `json:"message"`
    Read    bool   `json:"read"`
}
```

### 2. Create the Handler

```go
// internal/handler/notification.go
package handler

import "net/http"

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
    // Read from store, render template or return JSON
}
```

### 3. Register the Route

In `internal/server/routes.go`:

```go
r.Get("/api/notifications", s.handlers.ListNotifications)
```

### 4. Create the Template (for HTMX-rendered UI)

```text
web/templates/components/notifications.html
```

The renderer's HTMX detection handles partial vs. full page responses automatically.

### 5. Add Tests

```go
// internal/handler/notification_test.go
```

### 6. Build and Verify

```bash
make all
./clawide
# Test the new endpoint
curl http://localhost:9800/api/notifications
```

## Debugging Tips

### Server Logs

Run with debug-level logging:

```bash
./clawide --log-level debug
```

### Template Rendering

The template renderer logs template names and HTMX detection decisions at debug level. If a page renders incorrectly, check whether the HTMX partial vs. full-page path is being selected correctly.

### WebSocket Connections

Use browser DevTools (Network tab, WS filter) to inspect WebSocket frames for terminal and Docker log streaming. The terminal WebSocket endpoint is:

```text
ws://localhost:9800/ws/terminal/{sessionID}/{paneID}
```

### PTY/tmux Issues

List tmux sessions to verify ClawIDE's sessions exist:

```bash
tmux list-sessions
tmux list-panes -t <session-name>
```

### Docker Integration

If Docker features don't work, verify the Docker socket is accessible:

```bash
# Check socket exists
ls -la /var/run/docker.sock

# Test docker CLI
docker compose ps
```

When running ClawIDE in Docker, the socket must be mounted:

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock
```

### State File

ClawIDE persists state to `~/.clawide/state.json`. To reset state:

```bash
rm ~/.clawide/state.json
```

The PID file is at `~/.clawide/clawide.pid`. If ClawIDE won't start due to a stale PID file:

```bash
rm ~/.clawide/clawide.pid
```
