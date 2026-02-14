---
title: "Installation"
description: "Install ClawIDE using the one-line installation script."
weight: 20
---

ClawIDE is installed using a single command that downloads the latest pre-built binary for your system.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/master/scripts/install.sh | bash
```

This script will:
- Detect your OS and architecture
- Download the latest pre-built binary from GitHub releases
- Install to `/usr/local/bin` (with sudo if needed)
- Display the installation plan before proceeding

**You can inspect the script before running it:**
- View the script: [scripts/install.sh](https://github.com/davydany/ClawIDE/blob/master/scripts/install.sh)
- Review what it downloads: [GitHub Releases](https://github.com/davydany/ClawIDE/releases)

## Prerequisites

ClawIDE requires **tmux** as its terminal multiplexer backend:

```bash
# macOS
brew install tmux

# Debian/Ubuntu
sudo apt install tmux
```

## Start ClawIDE

Once installed, simply run:

```bash
clawide
```

Then open [http://localhost:9800](http://localhost:9800) in your browser.

## Verify the Installation

```bash
# Check the version
clawide --version

# Run on a custom port
clawide --port 8080

# Run with a specific projects directory
clawide --projects-dir /home/user/code
```

## Next Steps

- [Configuration]({{< ref "reference/configuration" >}}) — Customize host, port, projects directory, and more
- [Dashboard]({{< ref "features/dashboard" >}}) — Create your first project
