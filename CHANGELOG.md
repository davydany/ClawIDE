# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [1.2.0] - 2026-04-08

### Added

- **Theme Support**: Light and dark mode with manual toggle and system preference detection.
- **Movable Panes**: Drag-and-drop tab reordering to customize your workspace layout.
- **Rename Terminal Panes**: Rename terminal tabs for better session identification.
- **Hidden Files Toggle**: Show or hide dotfiles and hidden directories in the file editor tree.
- **Soft-Delete for Branches**: Deleting features/branches now sends them to trash instead of permanent removal, with the ability to restore.

### Fixed

- **Mobile Keyboard Handling**: Resolved issues with the virtual keyboard interfering with terminal input on mobile devices.
- **Copy Toast**: Fixed spurious copy-to-clipboard toast notifications appearing unexpectedly.
- **Touch Text Selection**: Improved text selection behavior on touch devices.

### Changed

- Added release helper commands to the Makefile for streamlined version tagging.

## [1.1.0] - 2026-03-24

### Added

- **MCP Server Management**: Full CRUD for MCP servers with process lifecycle management and log viewer.
- **Agent Management**: Agent CRUD with scope filtering and sidebar integration.
- **Skills Management**: UI for managing Claude Code skills with global and project scope support.
- **Command Palette**: VS Code-style unified command palette with file finder (Cmd+K / Cmd+Shift+P).
- **Clone from Git URL**: New project wizard option to clone directly from a Git repository URL.
- **Preferred Editor Support**: Choose your preferred external editor and open projects directly in it.
- **File/Folder Renaming**: Rename files and folders directly in the File Editor tab.
- **Markdown Preview**: Live preview for Markdown files in the editor.
- **Branch Management**: Improved branch support beyond just the main branch.
- **Auto-Create Default Session**: Automatically creates a session when opening a project with none.
- **Windows Support**: psmux backend for Windows terminal multiplexing.

### Changed

- Bind to localhost by default; use `--mobile` flag for LAN access.
- Sidebar collapse behavior improved with better state management.
- Replaced bash hook notifications with MCP server integration.

### Fixed

- Windows cross-compilation by extracting syscall usage into build-tagged files.
- Terminal copy behavior with tmux mouse mode.
- Markdown preview button behavior.

## [1.0.0] - 2026-03-04

### Added

- **Notes System**: Title-based filenames, folder directories, drag-and-drop organization, and save functionality.
- **Bookmarks Overhaul**: Root bookmarks populate the bar directly; removed star/InBar distinction.
- **Snippets**: Code snippet storage and management.
- **Project Bar**: All projects shown in sidebar bar with star/unstar and drag-to-reorder.
- **Color-Coded Projects**: Color-coding for projects and features for visual organization.
- **Docker Compose for Features**: Docker Compose support within feature workspaces.
- **Docker UI Overhaul**: Health checks, inline logs, prominent actions, card-based layout.
- **Docker Build Button**: Streaming Docker build output directly in the UI.
- **Docker Stack Restart**: One-click Docker Compose stack restart.
- **Auto-Copy on Select**: Terminal text selection automatically copies to clipboard.
- **New Project Wizard**: Guided project creation with empty project option.
- **ASCII Banner & QR Code**: Startup banner with URL display and QR code for mobile access.
- **Crab Mascot**: Tamagotchi-style crab mascot for startup and shutdown screens.
- **Favicon**: Added application favicon.
- **Google Analytics**: Build-time analytics injection for docs site.
- **Configurable Max Sessions**: Min/max bounds for concurrent session limits.

### Fixed

- Docker Up/Down error handling with stderr capture, timeouts, and UI error surfacing.
- Git identity initialization for CI environments.
- Job snapshot mutex copy issue.

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
