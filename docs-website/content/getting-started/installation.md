---
title: "Installation"
description: "Detailed installation instructions for ClawIDE: build from source or run with Docker."
weight: 20
---

ClawIDE can be installed in three ways: using the installation script, building from source, or running with Docker.

## Quick Install (Recommended)

The fastest way to get started is using our installation script. It automatically detects your system and downloads the latest pre-built binary.

### Install from Script

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

Once installed, simply run:

```bash
clawide
```

Then open [http://localhost:9800](http://localhost:9800) in your browser.

---

## Build from Source

### Prerequisites

You need the following tools installed on your system:

**Go 1.24+**

```bash
# macOS
brew install go

# Linux — download the official tarball
# See https://go.dev/doc/install
```

**Node.js** (for building frontend assets)

```bash
# macOS
brew install node

# Or use nvm
nvm install --lts
```

**tmux** (terminal multiplexer backend)

```bash
# macOS
brew install tmux

# Debian/Ubuntu
sudo apt install tmux
```

**Docker** (optional — only needed for Docker integration features)

```bash
# See https://docs.docker.com/get-docker/
```

### Build Steps

1. Clone the repository:

   ```bash
   git clone https://github.com/davydany/ClawIDE.git
   cd ClawIDE
   ```

2. Run the full build:

   ```bash
   make all
   ```

   This command runs three stages:
   - **`vendor-js`** — Downloads HTMX and Alpine.js, builds xterm.js and CodeMirror bundles via esbuild
   - **`css`** — Compiles and minifies Tailwind CSS
   - **`build`** — Compiles the Go binary with all assets embedded via `go:embed`

3. Start ClawIDE:

   ```bash
   ./clawide
   ```

4. Open [http://localhost:9800](http://localhost:9800) in your browser.

### Verify the Installation

```bash
# Check the version
./clawide --version

# Run on a custom port
./clawide --port 8080

# Run with a specific projects directory
./clawide --projects-dir /home/user/code
```

## Docker

If you prefer running ClawIDE in a container, use Docker Compose.

### Prerequisites

- Docker and Docker Compose installed on your system

### Steps

1. Clone the repository:

   ```bash
   git clone https://github.com/davydany/ClawIDE.git
   cd ClawIDE
   ```

2. Start with Docker Compose:

   ```bash
   docker compose up -d
   ```

3. Open [http://localhost:9800](http://localhost:9800) in your browser.

### Default Docker Configuration

The included `docker-compose.yml` is configured with:

| Mount | Purpose |
|-------|---------|
| `~/projects` (read-only) | Your projects directory |
| `~/.clawide` | Persistent state and configuration |
| `/var/run/docker.sock` | Docker socket (for Docker integration features) |

The container exposes port **9800** by default.

### Managing the Container

```bash
# Start
docker compose up -d

# Stop
docker compose down

# View logs
docker compose logs -f

# Check status
docker compose ps
```

## Next Steps

- [Configuration]({{< ref "reference/configuration" >}}) — Customize host, port, projects directory, and more
- [Dashboard]({{< ref "features/dashboard" >}}) — Create your first project
