---
title: "Quick Start"
description: "Get ClawIDE running in two steps."
weight: 10
---

Get ClawIDE up and running in under a minute.

## Prerequisites

Make sure you have **tmux** installed:

```bash
# macOS
brew install tmux

# Debian/Ubuntu
sudo apt install tmux
```

## Step 1: Install

```bash
curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/master/scripts/install.sh | bash
```

## Step 2: Start ClawIDE

```bash
clawide
```

Open [http://localhost:9800](http://localhost:9800) in your browser. You'll see the ClawIDE dashboard where you can create or import your first project.

## What's Next

- [Installation]({{< ref "getting-started/installation" >}}) — Detailed installation options and verification
- [Dashboard]({{< ref "features/dashboard" >}}) — Learn how to manage projects
- [Terminal Sessions]({{< ref "features/terminal-sessions" >}}) — Start your first Claude Code session
- [Configuration]({{< ref "reference/configuration" >}}) — Customize ClawIDE's settings
