---
title: "Quick Start"
description: "Get ClawIDE running in three steps."
weight: 10
---

Get ClawIDE up and running in under five minutes.

## Prerequisites

Make sure you have the following installed:

- **Go 1.24+**
- **Node.js** (LTS recommended)
- **tmux**

## Step 1: Clone the Repository

```bash
git clone https://github.com/davydany/ClawIDE.git
cd ClawIDE
```

## Step 2: Build

Run the full build, which compiles frontend assets (JavaScript bundles, Tailwind CSS) and the Go binary:

```bash
make all
```

This single command handles everything: vendoring JS dependencies, compiling CSS, bundling xterm.js and CodeMirror, and building the `clawide` binary.

## Step 3: Start ClawIDE

```bash
./clawide
```

Open [http://localhost:9800](http://localhost:9800) in your browser. You'll see the ClawIDE dashboard where you can create or import your first project.

## What's Next

- [Installation]({{< ref "getting-started/installation" >}}) — Detailed installation options including Docker
- [Dashboard]({{< ref "features/dashboard" >}}) — Learn how to manage projects
- [Terminal Sessions]({{< ref "features/terminal-sessions" >}}) — Start your first Claude Code session
- [Configuration]({{< ref "reference/configuration" >}}) — Customize ClawIDE's settings
