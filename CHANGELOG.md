# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.0] - 2024-12-01

Initial release of ClawIDE.

### Added

- **Foundation**: Go HTTP server with chi router, HTMX-aware template renderer, embedded static assets, Tailwind CSS, Alpine.js, and Docker-based local development setup.
- **Project Management**: Dashboard with project listing, project creation, and workspace navigation with tabbed layout.
- **Terminal Sessions**: Multiple concurrent Claude Code terminal sessions per project with xterm.js, WebSocket streaming, PTY management, split panes, and scrollback buffer.
- **Git Worktrees**: Branch listing, worktree creation/deletion, and session-to-worktree binding for parallel branch work.
- **File Browser and Editor**: Lazy-loaded directory tree, file read/write API, CodeMirror 6 editor with syntax highlighting and language detection, and responsive mobile/desktop layouts.
- **Docker Integration**: Docker Compose service management (ps, up, down, start, stop, restart), real-time log streaming via WebSocket, and service status badges.
- **Port Detection**: Automatic discovery of listening ports via OS scanning (lsof/ss) and docker-compose YAML extraction, with clickable port links in the UI.
- **Settings**: Configurable via CLI flags, environment variables, and JSON config file. In-app settings page for runtime configuration.
- **Single-Instance Enforcement**: PID file-based single-instance mode with `--restart` flag to replace running instances.
- **Mobile-First Design**: Responsive layout with touch targets, bottom tab bar, viewport handling, and full-screen mobile workflows.
- **Desktop Polish**: Resizable panels, keyboard shortcuts, and side-by-side layouts.
