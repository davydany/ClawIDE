---
title: "Installation"
description: "Install ClawIDE using the one-line installation script."
weight: 20
---

ClawIDE is installed using a single command that downloads the latest pre-built binary for your system.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/refs/heads/master/scripts/install.sh | bash
```

This script will:
- Detect your OS and architecture
- Download the requested version (or latest) pre-built binary from GitHub releases
- Install to `~/.local/bin`
- Display the installation plan before proceeding

**You can inspect the script before running it:**
- View the script: [scripts/install.sh](https://github.com/davydany/ClawIDE/blob/master/scripts/install.sh)
- Review what it downloads: [GitHub Releases](https://github.com/davydany/ClawIDE/releases)

### Install a Specific Version

Pin the version with the `VERSION` environment variable:

```bash
VERSION=1.2.3 curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/refs/heads/master/scripts/install.sh | bash
```

The leading `v` prefix is optional — `VERSION=v1.2.3` and `VERSION=1.2.3` are equivalent. If the version does not exist, the script exits with an error and a link to the releases page.

### Custom Install Directory

Override the default install directory (`~/.local/bin`) with `INSTALL_DIR`:

```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/refs/heads/master/scripts/install.sh | bash
```

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

## Uninstall

To remove ClawIDE, run the uninstall script:

```bash
curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/refs/heads/master/scripts/uninstall.sh | bash
```

This will remove the `clawide` binary and the `~/.clawide` configuration directory after asking for confirmation. It does not modify your shell config files (e.g., PATH entries in `~/.bashrc` or `~/.zshrc`).

You can override the locations with `INSTALL_DIR` and `CLAWIDE_DATA_DIR`:

```bash
INSTALL_DIR=/usr/local/bin CLAWIDE_DATA_DIR=/opt/clawide curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/refs/heads/master/scripts/uninstall.sh | bash
```

## Next Steps

- [Configuration]({{< ref "reference/configuration" >}}) — Customize host, port, projects directory, and more
- [Dashboard]({{< ref "features/dashboard" >}}) — Create your first project
