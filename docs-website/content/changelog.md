---
title: "Changelog"
description: "Release history and notable changes for ClawIDE."
weight: 50
---

All notable changes to ClawIDE are documented here. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [1.0.0] — 2026-03-04

The v1.0 release marks ClawIDE's graduation from early preview to a stable, full-featured IDE for managing Claude Code sessions. This release includes 48 commits since v0.1.4 with major new features, a Docker UI overhaul, and significant UX improvements.

### Added

- **Feature Workspaces**: Isolated development environments per feature — each gets its own git branch, worktree, terminal sessions, file browser, Docker stack, and merge review. Color-coded for quick identification.
- **LLM-Powered Project Wizard**: Create new projects from 15 framework templates across 8 languages, or generate a project using LLM providers (Claude, OpenAI). Includes an empty project option.
- **Merge Review**: Side-by-side diff viewer for reviewing changes before merging feature branches back to the main branch.
- **Global Scratchpad**: Persistent text area accessible from the sidebar with auto-save on blur. Also available within feature workspaces.
- **Docker Build Button**: Trigger `docker compose build` with streaming output directly from the Docker panel.
- **Docker Compose Restart**: One-click restart of the entire Docker Compose stack.
- **Docker Compose for Feature Workspaces**: Each feature workspace can run its own isolated Docker Compose stack.
- **Auto-Copy on Highlight**: Selecting text in a terminal session automatically copies it to the clipboard.
- **Project and Feature Color-Coding**: Visual color assignments for projects and feature workspaces.
- **Starred Projects with Drag-to-Reorder**: Star your most-used projects and reorder them by dragging on the dashboard.
- **New File/Folder Creation**: Create files and folders directly from the file tree UI.
- **Collapse/Expand Toggle**: Button to collapse or expand the entire file tree sidebar.
- **Sidebar Shortcuts Panel**: Quick-access panel in the sidebar for common actions.
- **Tamagotchi Crab Mascot**: Animated crab companion on startup and shutdown screens.
- **Favicon**: Custom favicon for the browser tab.
- **ASCII Banner and QR Code**: Startup console shows an ASCII art banner with the server URL and a QR code for mobile access.
- **Non-Starred Projects Dropdown**: Projects not starred are accessible via a dropdown on the dashboard bar.
- **Version Popover**: Update check icon now shows a version popover.

### Changed

- **Docker UI Overhaul**: Merged Docker and Ports tabs into a single card-based view with healthcheck indicators, inline log streaming, and prominent action buttons.
- **Notes Refactor**: Notes now support folders, drag-and-drop reordering, and title-based filenames. Notes are project-scoped.
- **Bookmarks Refactor**: Removed star/InBar mechanism — root-level bookmarks now automatically populate the bookmarks bar.
- **Settings Link**: Moved to the bottom of the sidebar, below system stats.
- **Mobile Editor**: Added word wrap, sidebar collapse, and command palette for mobile devices.
- **Project Listing**: Hidden files and worktree directories are excluded from the project listing.

### Fixed

- **Docker Up/Down**: Capture stderr, add timeouts, and surface errors in the UI.
- **Set local git identity** in `initGit` for CI environments.
- **Return pointer from `Job.Snapshot`** to avoid copying `sync.RWMutex`.
- **File path and direct content support** in supporting docs for project wizard.

---

## [0.1.4] — 2026-02-14

### Added

- AI Agent CLI settings separated into a dedicated configuration block.

### Fixed

- Screenshots not rendering in documentation.
- Documentation screenshot capture script simplified.

---

## [0.1.3] — 2026-02-14

### Fixed

- Remove hardcoded `~/projects` reference in welcome page.
- Installation script rewritten with simpler, linear structure.
- Use `~/.local/bin` as default install directory (no sudo required).
- Better error handling for sudo in non-interactive mode.
- Multiple install script fixes for piped execution, curl downloads, architecture names, and filename format.

---

## [0.1.2] — 2026-02-13

### Added

- Auto-copy with toast notifications and image paste support.
- Paste button in pane menu for manual clipboard pasting.
- Mobile paste event listener for iPad and touch devices.
- Documentation for 7 previously undocumented features.
- `/clawide-update` skill for automated release workflow.
- Value propositions section and installation script on docs site.

---

## [0.1.1] — 2026-02-13

### Changed

- Replace logo SVG with crab emoji and show version in sidebar.

---

## [0.1.0] — 2024-12-01

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
