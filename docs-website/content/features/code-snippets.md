---
title: "Code Snippets"
description: "Save, search, and insert reusable code snippets across sessions and projects."
weight: 80
---

Code snippets let you save frequently used code patterns, prompts, or text blocks and quickly insert them into any session. Snippets are global — they're available across all projects.

{{< screenshot src="code-snippets.png" alt="ClawIDE Code Snippets" caption="The code snippets panel with search and management options" >}}

## Creating a Snippet

1. Open the snippets panel.
2. Click **New Snippet**.
3. Enter a title and the snippet content.
4. Save the snippet.

Snippets are stored persistently, so they're available across sessions and server restarts.

## Searching Snippets

Use the search bar in the snippets panel to filter snippets by title or content. This is useful when you have a large collection and need to find a specific pattern quickly.

## Inserting a Snippet

1. Open the snippets panel while a terminal session is active.
2. Find the snippet you want to use.
3. Click **Insert** to paste the snippet content into the active terminal.

This is particularly useful for inserting common Claude Code prompts or commands that you use regularly.

## Editing a Snippet

1. Open the snippets panel.
2. Click on the snippet you want to edit.
3. Modify the title or content.
4. Save your changes.

## Deleting a Snippet

1. Open the snippets panel.
2. Select the snippet you want to remove.
3. Click **Delete** to permanently remove it.

## API

Snippets are managed through a global API (not scoped to a project):

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/snippets/` | GET | List all snippets |
| `/api/snippets/` | POST | Create a new snippet |
| `/api/snippets/{snippetID}` | PUT | Update an existing snippet |
| `/api/snippets/{snippetID}` | DELETE | Delete a snippet |

See the [API Reference]({{< ref "reference/api" >}}) for full details.
