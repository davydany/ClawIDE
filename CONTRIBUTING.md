# Contributing to ClawIDE

Thanks for your interest in contributing to ClawIDE. This guide covers the development setup, workflow, and conventions used in the project.

## Development Environment Setup

### Prerequisites

- Go 1.24+
- Node.js (LTS)
- tmux
- Docker and Docker Compose (for Docker integration features)

### Getting Started

```bash
git clone https://github.com/davydany/ClawIDE.git
cd ClawIDE

# Install JS dependencies and build frontend assets, then compile Go binary
make all

# Or run in development mode (builds assets + runs with `go run`)
make dev
```

The server starts at [http://localhost:9800](http://localhost:9800).

### Running Tests

```bash
make test
```

## Branch and PR Workflow

1. Fork the repository and clone your fork.
2. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/my-feature main
   ```
3. Make your changes with clear, incremental commits.
4. Run `make test` and `go fmt ./...` before pushing.
5. Open a pull request against `main` with a description of what changed and why.

### Commit Messages

- Use imperative mood: "Add file browser sorting" not "Added file browser sorting"
- Keep the subject line under 72 characters
- Reference issues where applicable: "Fix session cleanup on disconnect (#42)"

## Code Style

- **Go**: Run `go fmt ./...` before committing. The project follows standard Go conventions.
- **HTML templates**: Use Go's `html/template` syntax. Templates live in `web/templates/`.
- **CSS**: Tailwind utility classes. Source CSS is in `web/static/css/input.css`, compiled via `make css`.
- **JavaScript**: Vanilla JS and Alpine.js for interactivity. HTMX for server-driven UI updates.

## Project Architecture

ClawIDE follows a **handler -> model -> template** pattern:

```
HTTP Request
  -> chi router (internal/server/routes.go)
    -> middleware (HTMX detection, project context)
      -> handler (internal/handler/*.go)
        -> model/store (internal/model/, internal/store/)
        -> template render (internal/tmpl/ + web/templates/)
```

### Where to Add New Features

1. **New API endpoint**: Add the route in `internal/server/routes.go`, create or extend a handler in `internal/handler/`, and add any necessary models in `internal/model/`.

2. **New UI page or component**: Create the template in `web/templates/` (pages go in `pages/`, reusable parts in `components/`). The renderer automatically supports HTMX partial responses — if the request has `HX-Request` header, only the partial is returned; otherwise the full page layout wraps it.

3. **New backend service**: Create a new package under `internal/` (e.g., `internal/myservice/`). Wire it into the server via `internal/server/server.go`.

4. **New JavaScript bundle**: Add source in `web/src/`, configure the esbuild step in `web/src/package.json`, and reference the output from templates.

### Key Packages

| Package | Responsibility |
|---------|---------------|
| `cmd/clawide` | Entry point, config loading, server wiring, graceful shutdown |
| `internal/config` | Configuration from file, env vars, and CLI flags |
| `internal/handler` | HTTP/WebSocket request handlers |
| `internal/model` | Domain models (Project, Session, Pane, DockerService) |
| `internal/store` | JSON file-based state persistence |
| `internal/pty` | PTY process management, I/O streaming, scrollback buffer |
| `internal/tmux` | tmux command wrapper |
| `internal/docker` | Docker Compose CLI wrapper and YAML parser |
| `internal/git` | Git branch listing and worktree management |
| `internal/portdetect` | OS port scanning and compose port extraction |
| `internal/tmpl` | Go template renderer with HTMX partial support |
| `internal/middleware` | HTMX detection, project context loading |
| `internal/server` | HTTP server, route registration |
| `internal/pidfile` | Single-instance enforcement via PID file |
| `web` | Embedded static assets and templates (`go:embed`) |

## Testing

- Write tests for new functionality. Place test files alongside the code they test (`*_test.go`).
- Run `make test` to execute all tests.
- Tests should cover the primary behavior and meaningful edge cases — don't aim for 100% coverage at the expense of test quality.

## Reporting Issues

- Search existing issues before opening a new one.
- Include steps to reproduce, expected behavior, and actual behavior.
- For bugs, include your OS, Go version, and browser.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
