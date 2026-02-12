---
title: "Settings"
description: "Configure ClawIDE with CLI flags, environment variables, config file, or the UI settings page."
weight: 70
---

ClawIDE provides multiple ways to configure its behavior: CLI flags, environment variables, a JSON config file, and an in-app settings page.

{{< screenshot src="settings.png" alt="ClawIDE Settings" caption="The in-app settings page for configuring ClawIDE options" >}}

## Configuration Options

| Setting | Flag | Env Var | Default | Description |
|---------|------|---------|---------|-------------|
| Host | `--host` | `CLAWIDE_HOST` | `0.0.0.0` | Listen address |
| Port | `--port` | `CLAWIDE_PORT` | `9800` | Listen port |
| Projects Dir | `--projects-dir` | `CLAWIDE_PROJECTS_DIR` | `~/projects` | Root directory for projects |
| Max Sessions | `--max-sessions` | `CLAWIDE_MAX_SESSIONS` | `10` | Maximum concurrent terminal sessions |
| Scrollback Size | — | `CLAWIDE_SCROLLBACK_SIZE` | `65536` | Terminal scrollback buffer size (bytes) |
| Claude Command | `--claude-command` | `CLAWIDE_CLAUDE_COMMAND` | `claude` | Claude CLI binary name |
| Log Level | `--log-level` | `CLAWIDE_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| Data Dir | `--data-dir` | `CLAWIDE_DATA_DIR` | `~/.clawide` | Directory for state, config, and PID file |
| Restart | `--restart` | — | `false` | Kill existing instance and start a new one |

## Configuration Sources

ClawIDE loads configuration from three sources. When the same setting is defined in multiple sources, the highest-precedence source wins:

1. **CLI Flags** (highest precedence) — Passed when launching `./clawide`
2. **Environment Variables** — Set in your shell or `.env` file
3. **Config File** (lowest precedence) — Stored at `~/.clawide/config.json`

### Examples

**CLI flags:**

```bash
./clawide --port 8080 --projects-dir /home/user/code --log-level debug
```

**Environment variables:**

```bash
CLAWIDE_PORT=8080 CLAWIDE_PROJECTS_DIR=/home/user/code ./clawide
```

**Config file** (`~/.clawide/config.json`):

```json
{
  "host": "0.0.0.0",
  "port": 8080,
  "projects_dir": "/home/user/code",
  "max_sessions": 20,
  "log_level": "debug"
}
```

## In-App Settings Page

ClawIDE also provides a settings page accessible from the navigation bar:

1. Click **Settings** in the top navigation.
2. View and modify configuration options through the form.
3. Changes are saved to the config file and take effect without restarting.

## Restart Flag

If ClawIDE is already running and you want to replace the existing instance:

```bash
./clawide --restart
```

This kills the running ClawIDE process (identified by the PID file at `~/.clawide/clawide.pid`) and starts a new one. Without `--restart`, ClawIDE exits with an error if another instance is already running.

See the full [Configuration Reference]({{< ref "reference/configuration" >}}) for additional details.
