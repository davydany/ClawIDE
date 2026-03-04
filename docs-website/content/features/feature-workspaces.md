---
title: "Feature Workspaces"
description: "Isolated development environments with dedicated branches, sessions, and Docker stacks per feature."
weight: 45
---

Feature workspaces give each feature its own isolated environment — a git branch, worktree, terminal sessions, file browser, Docker stack, and merge review — all scoped to a single unit of work. When the feature is done, merge it back with a side-by-side diff review.

{{< screenshot src="feature-workspaces.png" alt="ClawIDE Feature Workspaces" caption="A feature workspace with its own sessions, file browser, and git status" >}}

## How It Works

Each feature workspace is backed by a git worktree. When you create a feature, ClawIDE:

1. Creates a new branch from the current HEAD.
2. Creates a git worktree checked out to that branch.
3. Assigns a color for visual identification.
4. Sets up an isolated workspace with its own tabs for sessions, files, Docker, and more.

This means you can work on multiple features simultaneously without stashing or switching branches.

## Creating a Feature Workspace

1. Open a project from the [Dashboard]({{< ref "features/dashboard" >}}).
2. Click **New Feature** in the project workspace.
3. Enter a name for the feature.
4. ClawIDE creates the branch, worktree, and workspace automatically.

The feature appears in the sidebar with its assigned color.

## Working in a Feature Workspace

Within a feature workspace, you have access to:

- **Terminal Sessions** — Sessions run in the feature's worktree directory, isolated from other features and the main branch.
- **File Browser** — Shows only the files in this feature's worktree.
- **Git Status** — See changed files specific to this branch.
- **Docker** — Run a Docker Compose stack scoped to this feature. Each feature can have its own running containers.
- **Scratchpad** — A per-feature scratch area for notes and quick text.

## Color-Coding

Each feature workspace is assigned a color. The color appears on the feature tab, sidebar entry, and throughout the workspace UI. Projects also have their own color, making it easy to distinguish which project and feature you're working in.

## Switching Between Features

Click on any feature in the sidebar to switch to its workspace. Each feature maintains its own state — open files, terminal sessions, and Docker containers persist independently.

## Merging a Feature

When your feature is complete:

1. Open the feature workspace.
2. Navigate to the **Merge Review** tab.
3. Review changes in the [side-by-side diff viewer]({{< ref "features/merge-review" >}}).
4. Click **Merge** to merge the feature branch back into the parent branch.
5. ClawIDE handles the merge and reports the result.

## Deleting a Feature

1. Select the feature workspace.
2. Click **Delete** to remove the feature, its worktree, branch, and associated sessions.

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/projects/{id}/features/` | POST | Create a feature workspace |
| `/projects/{id}/features/{fid}/` | GET | Open a feature workspace |
| `/projects/{id}/features/{fid}/` | DELETE | Delete a feature workspace |

See the [API Reference]({{< ref "reference/api" >}}) for full details.
