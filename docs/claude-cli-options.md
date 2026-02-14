# Claude Code CLI Options Reference

This document provides a comprehensive reference for Claude Code CLI options that can be configured in ClawIDE settings.

## Quick Start

Configure Claude Code in ClawIDE Settings:
- **AI Agent Command**: Select "Claude Code (claude)" or provide a custom path
- **Additional CLI Arguments**: Add any options from this reference

## Session & Output Options

### `-p, --print`
Print response and exit (useful for non-interactive use).
- Use case: Script integration, piping output to other tools

### `-c, --continue`
Continue the most recent conversation in the current directory.
- Use case: Resume work from previous session

### `-r, --resume [value]`
Resume a conversation by session ID, or open interactive picker.
- Use case: Resume specific past conversation

### `--session-id <uuid>`
Use a specific session ID for the conversation (must be a valid UUID).
- Use case: Programmatic session management

### `--no-session-persistence`
Disable session persistence - sessions will not be saved to disk.
- Use case: Privacy-focused temporary sessions

### `--fork-session`
When resuming, create a new session ID instead of reusing the original.
- Use case: Branch from existing conversation without modifying original

## Model & Performance

### `--model <model>`
Model for the current session.
- Options: `sonnet`, `opus`, `haiku` (aliases) or full name like `claude-sonnet-4-5-20250929`
- Example: `--model opus`

### `--effort <level>`
Effort level for the current session.
- Options: `low`, `medium`, `high`
- Example: `--effort high`

### `--fallback-model <model>`
Enable automatic fallback when default model is overloaded.
- Only works with `--print`
- Example: `--fallback-model sonnet`

## Permissions & Security

### `--permission-mode <mode>`
Set permission mode for tool access.
- Options: `default`, `acceptEdits`, `delegate`, `dontAsk`, `plan`, `bypassPermissions`
- Example: `--permission-mode acceptEdits`

### `--dangerously-skip-permissions`
Bypass all permission checks (sandbox only).
- ⚠️ Only recommended for sandboxes with no internet access

### `--allow-dangerously-skip-permissions`
Enable bypassing as an option without default bypass.
- ⚠️ Only recommended for sandboxes

## Tool & Environment Configuration

### `--tools <tools...>`
Specify available tools.
- Options: `""` (disable all), `default` (all tools), or comma-separated names
- Example: `--tools Bash,Edit,Read`

### `--allowed-tools <tools...>`
Whitelist specific tools.
- Example: `--allowed-tools Bash Edit Read`

### `--disallowed-tools <tools...>`
Deny specific tools.
- Example: `--disallowed-tools Bash`

### `--add-dir <directories...>`
Additional directories to allow tool access to.
- Example: `--add-dir /var/log /tmp`

## Prompting & System

### `--system-prompt <prompt>`
Custom system prompt for the session.
- Example: `--system-prompt "You are a backend specialist"`

### `--append-system-prompt <prompt>`
Append to default system prompt.
- Example: `--append-system-prompt "Always use async/await"`

## Advanced Features

### `--agent <agent>`
Agent for current session (overrides 'agent' setting).
- Example: `--agent researcher`

### `--agents <json>`
Define custom agents via JSON.
- Example: `--agents '{"reviewer": {"description": "Code reviewer", "prompt": "You are strict..."}}'`

### `--chrome`
Enable Claude in Chrome integration.

### `--no-chrome`
Disable Claude in Chrome integration.

### `--ide`
Automatically connect to IDE on startup if available.

### `--settings <file-or-json>`
Load additional settings from JSON file or string.
- Example: `--settings '{"verbose": true}'`

### `--setting-sources <sources>`
Which settings to load: `user`, `project`, `local` (comma-separated).
- Example: `--setting-sources user,project`

## Output Formatting (with --print)

### `--output-format <format>`
Output format for `--print` mode.
- Options: `text` (default), `json`, `stream-json`
- Example: `--output-format json`

### `--include-partial-messages`
Include partial message chunks as they arrive.
- Only with `--print` and `--output-format=stream-json`

### `--json-schema <schema>`
JSON Schema for structured output validation.
- Example: `--json-schema '{"type":"object","properties":{"name":{"type":"string"}}}'`

## Debugging

### `-d, --debug [filter]`
Enable debug mode with optional category filtering.
- Examples:
  - `--debug` (all debug logs)
  - `-d api,hooks` (specific categories)
  - `-d !1p,!file` (exclude categories)

### `--debug-file <path>`
Write debug logs to specific file path (enables debug mode).
- Example: `--debug-file /tmp/claude-debug.log`

### `--verbose`
Override verbose mode setting from config.

### `-v, --version`
Output the version number.

## MCP Servers

### `--mcp-config <configs...>`
Load MCP servers from JSON files or strings (space-separated).
- Example: `--mcp-config file.json '{"name":"server","command":"node"}'`

### `--strict-mcp-config`
Only use MCP servers from `--mcp-config`, ignore other configurations.

## Input/Output Control

### `--input-format <format>`
Input format (with `--print`).
- Options: `text` (default), `stream-json`

### `--replay-user-messages`
Re-emit user messages from stdin for acknowledgment.
- Only with `--input-format=stream-json` and `--output-stream-json`

### `--disable-slash-commands`
Disable all skills.

## API & Budget

### `--max-budget-usd <amount>`
Maximum dollar amount for API calls.
- Only works with `--print`
- Example: `--max-budget-usd 5.00`

### `--betas <betas...>`
Beta headers for API requests (API key users only).

## File Resources

### `--file <specs...>`
Download file resources at startup.
- Format: `file_id:relative_path`
- Example: `--file file_abc:doc.txt file_def:img.png`

## Examples

### Basic interactive session with Sonnet model
```
--model sonnet
```

### High-effort thinking session
```
--model opus --effort high
```

### Non-interactive print with budget limit
```
--print --max-budget-usd 5 --output-format json
```

### Custom system prompt with specific tools
```
--system-prompt "You are a Python expert" --tools Bash,Edit,Read,Grep
```

### Debug mode with logging
```
--debug --debug-file /tmp/claude-debug.log --verbose
```

### Session resumption
```
--continue
```

## Common ClawIDE Configurations

### Development focus (high effort)
```
--model opus --effort high --verbose
```

### Quick responses (budget-conscious)
```
--model sonnet --effort low
```

### Read-only analysis (safe)
```
--tools Read,Grep,Glob --permission-mode plan
```

### Full automation (trust-based)
```
--model opus --permission-mode acceptEdits --effort high
```
