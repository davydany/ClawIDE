---
title: "Configuration"
description: "Complete configuration reference for ClawIDE: flags, environment variables, and config file."
weight: 20
---

ClawIDE is configured through three sources, listed in order of precedence:

1. **CLI Flags** (highest) — Passed when launching `./clawide`
2. **Environment Variables** — Set in your shell or process environment
3. **Config File** (lowest) — `~/.clawide/config.json`

When the same setting is defined in multiple sources, the highest-precedence source wins.

## Configuration Reference

| Setting | Flag | Env Var | Default | Description |
|---------|------|---------|---------|-------------|
| Host | `--host` | `CLAWIDE_HOST` | `0.0.0.0` | Listen address for the HTTP server |
| Port | `--port` | `CLAWIDE_PORT` | `9800` | Listen port for the HTTP server |
| Projects Dir | `--projects-dir` | `CLAWIDE_PROJECTS_DIR` | `~/projects` | Root directory where projects are located |
| Max Sessions | `--max-sessions` | `CLAWIDE_MAX_SESSIONS` | `10` | Maximum number of concurrent terminal sessions |
| Scrollback Size | — | `CLAWIDE_SCROLLBACK_SIZE` | `65536` | Terminal scrollback buffer size in bytes |
| Claude Command | `--claude-command` | `CLAWIDE_CLAUDE_COMMAND` | `claude` | Name or path of the Claude CLI binary |
| Log Level | `--log-level` | `CLAWIDE_LOG_LEVEL` | `info` | Logging verbosity: `debug`, `info`, `warn`, `error` |
| Data Dir | `--data-dir` | `CLAWIDE_DATA_DIR` | `~/.clawide` | Directory for state file, config, and PID file |
| Restart | `--restart` | — | `false` | Kill any running ClawIDE instance before starting |

## Config File

The config file is located at `~/.clawide/config.json`. ClawIDE reads this file on startup. If the file doesn't exist, defaults are used.

### Example Config File

```json
{
  "host": "0.0.0.0",
  "port": 9800,
  "projects_dir": "~/projects",
  "max_sessions": 10,
  "scrollback_size": 65536,
  "claude_command": "claude",
  "log_level": "info",
  "data_dir": "~/.clawide"
}
```

### Data Directory Contents

The data directory (`~/.clawide` by default) contains:

| File | Purpose |
|------|---------|
| `config.json` | Configuration file |
| `state.json` | Persistent application state (projects, sessions) |
| `clawide.pid` | PID file for single-instance enforcement |

## Precedence Order

The precedence order determines which value is used when the same setting is defined in multiple places:

```text
CLI Flags  >  Environment Variables  >  Config File  >  Defaults
(highest)                                               (lowest)
```

### Example

If you set `port` in all three:

```bash
# Config file says port 9800 (default)
# Environment says port 8080
export CLAWIDE_PORT=8080

# CLI flag says port 3000
./clawide --port 3000
```

ClawIDE listens on port **3000** because CLI flags have the highest precedence.

## CLI Flag Usage

```bash
# View help
./clawide --help

# Run with custom settings
./clawide --port 8080 --projects-dir /home/user/code --log-level debug

# Replace a running instance
./clawide --restart

# Specify a custom data directory
./clawide --data-dir /tmp/clawide-dev

# Use a different Claude binary
./clawide --claude-command /usr/local/bin/claude-beta
```

## Environment Variable Usage

```bash
# Set via export
export CLAWIDE_PORT=8080
export CLAWIDE_PROJECTS_DIR=/home/user/code
export CLAWIDE_LOG_LEVEL=debug
./clawide

# Or inline
CLAWIDE_PORT=8080 CLAWIDE_PROJECTS_DIR=/home/user/code ./clawide
```

## In-App Settings

ClawIDE also provides a settings page in the web UI at `/settings`. Changes made through the settings page are written to the config file (`~/.clawide/config.json`) and take effect without restarting the server.

## Single-Instance Enforcement

ClawIDE writes a PID file to `{data-dir}/clawide.pid` on startup. If an instance is already running:

- **Without `--restart`**: The new process exits with an error message.
- **With `--restart`**: The existing process is killed and the new instance starts.

To manually clear a stale PID file:

```bash
rm ~/.clawide/clawide.pid
```
