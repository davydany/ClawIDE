---
title: "API Reference"
description: "Complete HTTP and WebSocket API documentation for all ClawIDE endpoints."
weight: 30
---

ClawIDE exposes HTTP and WebSocket endpoints for all functionality. The HTTP API is primarily consumed by the HTMX frontend, but all endpoints can be called directly.

## Global Endpoints

### Version

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/version` | Returns the ClawIDE version |

### Dashboard

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Renders the main dashboard page |

### Settings

| Method | Path | Description |
|--------|------|-------------|
| GET | `/settings` | Renders the settings page |
| PUT | `/api/settings` | Update configuration settings |

### Onboarding

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/onboarding/complete` | Mark the welcome onboarding as complete |
| POST | `/api/onboarding/workspace-tour-complete` | Mark the workspace tour as complete |
| POST | `/api/onboarding/reset` | Reset all onboarding state |

### Static Files

| Method | Path | Description |
|--------|------|-------------|
| GET | `/static/*` | Serves embedded static assets (CSS, JS, vendor libraries) |

## Snippets

Snippet endpoints are global and not scoped to a project.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/snippets/` | List all code snippets |
| POST | `/api/snippets/` | Create a new code snippet |
| PUT | `/api/snippets/{snippetID}` | Update an existing snippet |
| DELETE | `/api/snippets/{snippetID}` | Delete a snippet |

## Project Endpoints

### Projects

| Method | Path | Description |
|--------|------|-------------|
| GET | `/projects/` | List all projects |
| POST | `/projects/` | Create a new project |
| GET | `/projects/{id}/` | Render the project workspace |
| DELETE | `/projects/{id}/` | Delete a project |

All routes under `/projects/{id}/` pass through the `ProjectLoader` middleware, which loads the project from the store into the request context.

### Sessions

| Method | Path | Description |
|--------|------|-------------|
| GET | `/projects/{id}/sessions/` | List all sessions for a project |
| POST | `/projects/{id}/sessions/` | Create a new terminal session |
| PATCH | `/projects/{id}/sessions/{sid}/` | Rename a session |
| DELETE | `/projects/{id}/sessions/{sid}/` | Delete a session and its panes |

### Panes

Pane operations are scoped to a specific session.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/projects/{id}/sessions/{sid}/panes/{pid}/split` | Split a pane (horizontal or vertical) |
| DELETE | `/projects/{id}/sessions/{sid}/panes/{pid}` | Close a pane |
| PATCH | `/projects/{id}/sessions/{sid}/panes/{pid}/resize` | Resize a pane |

### Files

| Method | Path | Description |
|--------|------|-------------|
| GET | `/projects/{id}/api/files` | List files and directories in the project |
| GET | `/projects/{id}/api/file` | Read a file's contents |
| PUT | `/projects/{id}/api/file` | Write (save) a file |

### Docker

| Method | Path | Description |
|--------|------|-------------|
| GET | `/projects/{id}/api/docker/ps` | List Docker Compose service status |
| POST | `/projects/{id}/api/docker/up` | Start all Docker Compose services (`docker compose up -d`) |
| POST | `/projects/{id}/api/docker/down` | Stop all Docker Compose services (`docker compose down`) |
| POST | `/projects/{id}/api/docker/{svc}/start` | Start a specific service |
| POST | `/projects/{id}/api/docker/{svc}/stop` | Stop a specific service |
| POST | `/projects/{id}/api/docker/{svc}/restart` | Restart a specific service |

### Git

| Method | Path | Description |
|--------|------|-------------|
| GET | `/projects/{id}/api/worktrees` | List all git worktrees |
| POST | `/projects/{id}/api/worktrees` | Create a new worktree |
| DELETE | `/projects/{id}/api/worktrees/{wid}` | Delete a worktree |
| GET | `/projects/{id}/api/branches` | List all git branches |
| POST | `/projects/{id}/api/branches` | Create a new branch |
| POST | `/projects/{id}/api/checkout` | Checkout a branch |

### Ports

| Method | Path | Description |
|--------|------|-------------|
| GET | `/projects/{id}/api/ports` | Detect and list listening ports for the project |

### Features (Worktree Workspaces)

Feature endpoints create self-contained workspaces backed by git worktrees.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/projects/{id}/features/` | Create a new feature workspace |
| GET | `/projects/{id}/features/{fid}/` | Open a feature workspace |
| DELETE | `/projects/{id}/features/{fid}/` | Delete a feature workspace |
| POST | `/projects/{id}/features/{fid}/sessions/` | Create a session in the feature workspace |

#### Feature File Operations

| Method | Path | Description |
|--------|------|-------------|
| GET | `/projects/{id}/features/{fid}/api/files` | List files in the feature worktree |
| GET | `/projects/{id}/features/{fid}/api/file` | Read a file from the feature worktree |
| PUT | `/projects/{id}/features/{fid}/api/file` | Write a file in the feature worktree |

#### Feature Git Operations

| Method | Path | Description |
|--------|------|-------------|
| GET | `/projects/{id}/features/{fid}/api/status` | Get git status for the feature branch |
| POST | `/projects/{id}/features/{fid}/api/commit` | Commit changes in the feature branch |
| POST | `/projects/{id}/features/{fid}/api/merge` | Merge the feature branch back to the parent |

## WebSocket Endpoints

### Terminal

| Path | Description |
|------|-------------|
| `ws://host:port/ws/terminal/{sessionID}/{paneID}` | Bidirectional terminal I/O for a specific pane |

The terminal WebSocket streams binary frames:
- **Client → Server**: Keyboard input written to the PTY stdin
- **Server → Client**: PTY stdout output sent as binary WebSocket frames

On disconnect, the WebSocket is closed but the underlying PTY/tmux session remains alive for reconnection.

### Docker Logs

| Path | Description |
|------|-------------|
| `ws://host:port/ws/docker/{projectID}/logs/{svc}` | Real-time log streaming for a Docker Compose service |

The Docker logs WebSocket streams log output from a specific service. New log lines are pushed to the client as they arrive.

## Middleware

All requests pass through a global middleware chain:

| Middleware | Purpose |
|------------|---------|
| `Logger` | Logs all HTTP requests |
| `Recoverer` | Recovers from panics and returns 500 |
| `Compress` | Applies gzip compression to responses |
| `HTMXDetect` | Detects `HX-Request` header and sets a context flag |
| `ProjectLoader` | Loads project from store for `/projects/{id}/*` routes |

The `HTMXDetect` middleware enables the template renderer to return partial HTML fragments for HTMX requests and full-page layouts for direct browser navigation.
