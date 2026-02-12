---
title: "Dashboard"
description: "Manage all your projects from the ClawIDE dashboard."
weight: 10
---

The dashboard is your central hub for managing projects in ClawIDE. From here you can view all imported projects, create new ones, and navigate to any project workspace.

{{< screenshot src="dashboard.png" alt="ClawIDE Dashboard" caption="The project dashboard showing all imported projects" >}}

## Project List

When you open ClawIDE, the dashboard displays all your projects as cards. Each project card shows:

- The project name
- The project path on disk
- Quick-access navigation to the project workspace

Projects are loaded from the configured projects directory (default: `~/projects`).

## Creating a New Project

To create a new project:

1. Click the **Create Project** button on the dashboard.
2. Enter a name for your project.
3. ClawIDE creates a new directory in your projects folder and sets up the project workspace.

## Importing Existing Projects

ClawIDE automatically discovers directories within your configured projects path. Any directory in `~/projects` (or your configured projects directory) appears on the dashboard and can be opened as a workspace.

To change the projects directory, see [Settings]({{< ref "features/settings" >}}) or [Configuration]({{< ref "reference/configuration" >}}).

## Navigating to a Project

Click on any project card to open its workspace. The workspace provides access to:

- [Terminal Sessions]({{< ref "features/terminal-sessions" >}}) — Run Claude Code sessions
- [File Editor]({{< ref "features/file-editor" >}}) — Browse and edit files
- [Docker Integration]({{< ref "features/docker-integration" >}}) — Manage Docker services
- [Git Worktrees]({{< ref "features/git-worktrees" >}}) — Work on multiple branches
- [Port Detection]({{< ref "features/port-detection" >}}) — View discovered ports

## Deleting a Project

To remove a project from ClawIDE:

1. Open the project workspace.
2. Use the project delete action.

Deleting a project from ClawIDE removes it from the dashboard and cleans up associated sessions. The project files on disk are not deleted.
