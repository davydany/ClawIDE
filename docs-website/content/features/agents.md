---
title: "Agent Management"
description: "Create, manage, and organize Claude Code agents with global and project-level scoping."
weight: 55
---

Agent management lets you define reusable Claude Code agents directly from ClawIDE. Agents are stored as markdown files in `.claude/agents/` directories, with global agents available across all projects and project-scoped agents isolated to a single workspace.

{{< screenshot src="agents.png" alt="ClawIDE Agent Management" caption="Agent management panel with global and project-scoped agents" >}}

## What Are Agents?

Claude Code agents are specialized configurations that define how Claude behaves for specific tasks. Each agent has a name, description, model preference, allowed tools, and custom instructions written in markdown. For example, you might create a "code-reviewer" agent that uses Opus and has access to read-only tools, or a "test-writer" agent that focuses on generating tests.

## Creating an Agent

1. Open the agent management panel from the sidebar.
2. Click **New Agent**.
3. Fill in the agent details:
   - **Name** — A unique identifier (used as the filename)
   - **Description** — What this agent does
   - **Model** — Which Claude model to use (e.g., `opus`, `sonnet`, `haiku`)
   - **Allowed Tools** — Comma-separated list of tools the agent can use
   - **Agent Type** — The type of agent behavior
   - **Content** — The agent's instructions in markdown
4. Choose the scope: **Global** or **Project**.
5. Click **Save**.

## Scoping

Agents live in one of two scopes:

- **Global** (`~/.claude/agents/`) — Available in every project. Good for general-purpose agents like reviewers or planners.
- **Project** (`<project>/.claude/agents/`) — Available only in that project. Good for project-specific workflows.

You can move an agent between scopes at any time using the **Move** action.

## Editing and Deleting

Click on any agent to view and edit its configuration. Changes are saved directly to the agent's markdown file. Delete an agent to remove it from disk.

## Storage Format

Each agent is a `.md` file with YAML frontmatter:

```markdown
---
name: code-reviewer
description: Reviews code for quality and correctness
model: opus
allowed-tools: Read, Grep, Glob
agent-type: reviewer
---

You are a code reviewer. Focus on correctness, security, and maintainability...
```
