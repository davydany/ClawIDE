---
title: "Dashboard"
description: "Manage all your projects from the ClawIDE dashboard."
weight: 10
---

The dashboard is your central hub for managing projects in ClawIDE. From here you can view all imported projects, create new ones, and navigate to any project workspace.

{{< screenshot src="dashboard.png" alt="ClawIDE Dashboard" caption="The project dashboard showing all imported projects" >}}

## Project List

When you open ClawIDE, the dashboard displays your starred projects in a bar at the top, with remaining projects accessible via a dropdown. Each project shows:

- The project name with its assigned color
- The project path on disk
- Quick-access navigation to the project workspace

Projects are loaded from the configured projects directory (default: `~/projects`). Hidden files and worktree directories are automatically excluded.

## Starred Projects

Star your most-used projects to pin them to the dashboard bar. Drag starred projects to reorder them. Non-starred projects are available in a dropdown below the bar.

## Color-Coding

Each project is assigned a color that appears on the project card and throughout its workspace. [Feature workspaces]({{< ref "features/feature-workspaces" >}}) within a project also get their own colors, making it easy to visually identify where you are.

## Creating a New Project

To create a new project:

1. Click the **Create Project** button on the dashboard.
2. The [Project Wizard]({{< ref "features/project-wizard" >}}) opens with options to create from a template, generate with an LLM, or start empty.
3. ClawIDE creates the project directory and sets up the workspace.

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
