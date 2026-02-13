---
title: "Claude Code Hooks"
description: "Receive notifications when Claude Code finishes tasks via hook integration."
weight: 150
---

ClawIDE integrates with Claude Code's hook system to notify you when Claude finishes a task. When a Claude Code session completes in any ClawIDE terminal, a notification is delivered to the [notification center]({{< ref "features/notifications" >}}) in real time.

{{< screenshot src="claude-hooks.png" alt="ClawIDE Claude Code Hooks" caption="The settings page showing Claude Code hook integration status" >}}

## How It Works

Claude Code supports lifecycle hooks — shell scripts that run when certain events occur. ClawIDE installs a "stop" hook that fires whenever Claude Code finishes processing. The hook sends a notification to ClawIDE's API with context about which project, session, and pane the task ran in.

## Setting Up the Hook

1. Open the **Settings** page.
2. ClawIDE detects whether the `claude` CLI is installed on your system.
3. Click **Enable Claude Code Integration** to install the hook.

ClawIDE creates a hook script at `~/.clawide/hooks/claude-stop-hook.sh` and registers it in Claude's settings at `~/.claude/settings.json`.

## What Gets Captured

When Claude Code finishes, the hook captures:

- **Stop reason** — Why Claude stopped (completed, interrupted, error, etc.)
- **Working directory** — Where Claude was running
- **Project context** — The ClawIDE project, session, feature, and pane IDs

This context is passed through environment variables that ClawIDE sets in terminal sessions:

| Variable | Description |
|----------|-------------|
| `CLAWIDE_PROJECT_ID` | The project the session belongs to |
| `CLAWIDE_SESSION_ID` | The terminal session ID |
| `CLAWIDE_FEATURE_ID` | The feature workspace ID (if applicable) |
| `CLAWIDE_PANE_ID` | The specific pane ID |
| `CLAWIDE_API_URL` | The ClawIDE API base URL |

## Notification Delivery

The hook sends a notification to ClawIDE's `/api/notifications` endpoint. The notification appears in the [notification center]({{< ref "features/notifications" >}}) with:

- **Title**: "Claude Code {stop_reason}"
- **Source**: `claude`
- **Level**: `success`
- **Context**: Project and session details for navigation

Duplicate notifications are prevented using idempotency keys.

## Removing the Hook

1. Open the **Settings** page.
2. Click **Remove Claude Code Integration** to uninstall the hook.

This removes the hook script and cleans up the Claude settings entry.

## Requirements

- Claude Code CLI (`claude`) must be installed and accessible in your PATH.
- `jq` is optional but recommended for cleaner JSON parsing in the hook script.

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/claude/detect` | GET | Check if the Claude CLI is installed |
| `/api/claude/setup-hook` | POST | Install the hook script and configure Claude |
| `/api/claude/hook` | DELETE | Remove the hook and clean up Claude settings |

See the [API Reference]({{< ref "reference/api" >}}) for full details.
