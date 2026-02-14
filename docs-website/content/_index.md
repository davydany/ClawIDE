---
title: "ClawIDE"
description: "A web-based IDE for managing multiple Claude Code sessions across projects."
layout: index
---

## One Binary. Multiple Sessions. Total Control.

ClawIDE is a web-based IDE for managing multiple Claude Code sessions across projects, with terminal multiplexing, file editing, Docker integration, and git worktree support — all from a single Go binary.

### Features

- **Terminal Sessions** — Run multiple Claude Code sessions side-by-side with split panes, powered by xterm.js and WebSocket streaming
- **File Editor** — Browse and edit project files with CodeMirror 6, syntax highlighting, and language detection
- **Docker Integration** — View container status, start/stop/restart services, and stream logs from docker-compose stacks
- **Git Worktrees** — Create and manage git worktrees to run parallel Claude sessions on different branches
- **Port Detection** — Automatically discover listening ports from running processes and docker-compose configurations
- **Code Snippets** — Save, search, and insert reusable code snippets across sessions
- **Settings** — Configure projects directory, max sessions, scrollback size, and more from the UI
- **Mobile-First** — Responsive design with touch targets, bottom tab bar, and full-screen mobile workflows

### Quick Start

```bash
curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/refs/heads/master/scripts/install.sh | bash
clawide
```

Open [http://localhost:9800](http://localhost:9800) in your browser.
