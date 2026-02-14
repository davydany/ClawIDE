# ClawIDE

A web-based IDE for managing multiple Claude Code sessions across projects, with terminal multiplexing, file editing, Docker integration, and git worktree support — all from a single Go binary.

<!-- TODO: Add hero screenshot -->
<!-- ![ClawIDE Dashboard](docs/screenshots/dashboard.png) -->

## Features

- **Terminal Sessions** — Run multiple Claude Code sessions side-by-side with split panes, powered by xterm.js and WebSocket streaming
- **File Editor** — Browse and edit project files with CodeMirror 6, syntax highlighting, and language detection
- **Docker Integration** — View container status, start/stop/restart services, and stream logs from docker-compose stacks
- **Git Worktrees** — Create and manage git worktrees to run parallel Claude sessions on different branches
- **Port Detection** — Automatically discover listening ports from running processes and docker-compose configurations
- **Settings** — Configure projects directory, max sessions, scrollback size, and more from the UI
- **Mobile-First** — Responsive design with touch targets, bottom tab bar, and full-screen mobile workflows

<!-- TODO: Add feature screenshots/GIFs -->
<!--
![Terminal Sessions](docs/screenshots/terminal.png)
![File Editor](docs/screenshots/editor.png)
![Docker Panel](docs/screenshots/docker.png)
-->

## Prerequisites

- **Go 1.24+**
- **Node.js** (for building frontend assets)
- **tmux** (terminal multiplexer backend)
- **Docker** (optional, for Docker integration features)

## Quick Start

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/master/scripts/install.sh | bash
clawide
```

Open [http://localhost:9800](http://localhost:9800) in your browser.

### Build from Source

```bash
git clone https://github.com/davydany/ClawIDE.git
cd ClawIDE

# Build frontend assets and Go binary
make all

# Run the server
./clawide
```

Open [http://localhost:9800](http://localhost:9800) in your browser.

### Docker

```bash
git clone https://github.com/davydany/ClawIDE.git
cd ClawIDE

# Start with docker compose
docker compose up -d
```

The default `docker-compose.yml` mounts `~/projects` (read-only) and `~/.clawide` for persistent state, and exposes port 9800.

## Configuration

ClawIDE loads configuration from three sources in order of precedence: **flags > environment variables > config file**.

The config file is located at `~/.clawide/config.json`.

| Setting | Flag | Env Var | Default | Description |
|---------|------|---------|---------|-------------|
| Host | `--host` | `CLAWIDE_HOST` | `0.0.0.0` | Listen address |
| Port | `--port` | `CLAWIDE_PORT` | `9800` | Listen port |
| Projects Dir | `--projects-dir` | `CLAWIDE_PROJECTS_DIR` | `~/projects` | Root directory for projects |
| Max Sessions | `--max-sessions` | `CLAWIDE_MAX_SESSIONS` | `10` | Maximum concurrent terminal sessions |
| Scrollback Size | — | `CLAWIDE_SCROLLBACK_SIZE` | `65536` | Terminal scrollback buffer size (bytes) |
| Claude Command | `--claude-command` | `CLAWIDE_CLAUDE_COMMAND` | `claude` | Claude CLI binary name |
| Log Level | `--log-level` | `CLAWIDE_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| Data Dir | `--data-dir` | `CLAWIDE_DATA_DIR` | `~/.clawide` | Directory for state, config, and PID file |
| Restart | `--restart` | — | `false` | Kill existing instance and start a new one |

### Example

```bash
# Run on a custom port with a specific projects directory
./clawide --port 8080 --projects-dir /home/user/code

# Or via environment variables
CLAWIDE_PORT=8080 CLAWIDE_PROJECTS_DIR=/home/user/code ./clawide
```

## Project Structure

```
ClawIDE/
├── cmd/clawide/          # Application entry point
├── internal/
│   ├── config/           # Configuration loading (file, env, flags)
│   ├── docker/           # Docker Compose CLI wrapper and YAML parser
│   ├── git/              # Git operations (branches, worktrees)
│   ├── handler/          # HTTP and WebSocket request handlers
│   ├── middleware/        # HTMX detection, project context loading
│   ├── model/            # Domain models (project, session, pane, docker)
│   ├── pidfile/          # Single-instance enforcement via PID file
│   ├── portdetect/       # Port scanning (lsof/ss) and compose port extraction
│   ├── pty/              # PTY session management and I/O streaming
│   ├── server/           # HTTP server setup and route registration
│   ├── store/            # JSON state persistence
│   └── tmpl/             # Go template renderer with HTMX partial support
├── web/
│   ├── src/              # Frontend source (xterm.js, CodeMirror bundles)
│   ├── static/           # CSS, JS, vendored libraries
│   └── templates/        # Go HTML templates (base, layouts, pages, components)
├── Dockerfile            # Multi-stage build
├── docker-compose.yml    # Local development stack
├── Makefile              # Build, dev, and Docker targets
└── go.mod                # Go module definition
```

## Documentation

- [Architecture](docs/architecture.md) — System design, request flow, and key decisions
- [Development Guide](docs/development.md) — Local setup, build pipeline, and adding features
- [Contributing](CONTRIBUTING.md) — How to contribute
- [Changelog](CHANGELOG.md) — Release history

## License

[MIT](LICENSE) — Copyright 2024 davydany
