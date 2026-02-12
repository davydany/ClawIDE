---
title: "Contributing"
description: "How to contribute to ClawIDE: branch workflow, commit conventions, code style, and testing."
weight: 20
---

Thanks for your interest in contributing to ClawIDE. This guide covers the development workflow, coding conventions, and how to submit changes.

## Development Environment

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

# Or run in development mode (builds assets + runs with go run)
make dev
```

The server starts at [http://localhost:9800](http://localhost:9800).

### Running Tests

```bash
make test
```

## Branch and PR Workflow

1. **Fork** the repository and clone your fork.
2. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/my-feature main
   ```
3. **Make your changes** with clear, incremental commits.
4. **Run checks** before pushing:
   ```bash
   make test
   go fmt ./...
   ```
5. **Open a pull request** against `main` with a description of what changed and why.

## Commit Message Convention

- Use **imperative mood**: "Add file browser sorting" not "Added file browser sorting"
- Keep the **subject line under 72 characters**
- **Reference issues** where applicable: "Fix session cleanup on disconnect (#42)"

### Examples

```text
Add file browser sorting by name and date
Fix session cleanup on disconnect (#42)
Refactor Docker service handler for testability
Update xterm.js to 5.x and fix resize handling
```

## Code Style

### Go

Run `go fmt ./...` before committing. The project follows standard Go conventions.

### HTML Templates

Use Go's `html/template` syntax. Templates live in `web/templates/` organized into:
- `pages/` — Full page templates
- `components/` — Reusable partial templates
- `layouts/` — Base layout wrappers

### CSS

Tailwind utility classes. Source CSS is in `web/static/css/input.css`, compiled via `make css`.

### JavaScript

Vanilla JS and Alpine.js for interactivity. HTMX handles server-driven UI updates. No heavy frameworks.

## Project Architecture

ClawIDE follows a **handler → model → template** pattern:

```text
HTTP Request
  → chi router (internal/server/routes.go)
    → middleware (HTMX detection, project context)
      → handler (internal/handler/*.go)
        → model/store (internal/model/, internal/store/)
        → template render (internal/tmpl/ + web/templates/)
```

### Where to Add New Features

1. **New API endpoint**: Add the route in `internal/server/routes.go`, create or extend a handler in `internal/handler/`, and add any necessary models in `internal/model/`.

2. **New UI page or component**: Create the template in `web/templates/` (pages in `pages/`, reusable parts in `components/`). The renderer automatically supports HTMX partial responses — if the request has `HX-Request` header, only the partial is returned; otherwise the full page layout wraps it.

3. **New backend service**: Create a new package under `internal/` (e.g., `internal/myservice/`). Wire it into the server via `internal/server/server.go`.

4. **New JavaScript bundle**: Add source in `web/src/`, configure the esbuild step in `web/src/package.json`, and reference the output from templates.

## Testing

- Write tests for new functionality. Place test files alongside the code they test (`*_test.go`).
- Run `make test` to execute all tests.
- Tests should cover the primary behavior and meaningful edge cases — don't aim for 100% coverage at the expense of test quality.

## Reporting Issues

- Search existing issues before opening a new one.
- Include steps to reproduce, expected behavior, and actual behavior.
- For bugs, include your OS, Go version, and browser.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](https://github.com/davydany/ClawIDE/blob/main/LICENSE).
