---
title: "VoiceBox"
description: "Capture voice memos and quick text entries for reference across sessions."
weight: 160
---

VoiceBox is a quick-capture tool for saving voice memos and text entries. Use it to record thoughts, reminders, or instructions while working across multiple Claude Code sessions.

{{< screenshot src="voicebox.png" alt="ClawIDE VoiceBox" caption="The VoiceBox panel showing captured voice memos and text entries" >}}

## Capturing Entries

1. Open the VoiceBox panel.
2. Enter your text or use voice input to capture a memo.
3. The entry is saved with a timestamp.

Entries are stored globally — they're available regardless of which project you're working in.

## Viewing Entries

All captured entries are listed with their content and creation timestamp, sorted with the most recent entries first.

## Deleting Entries

- Click the delete button on any entry to remove it.
- Use **Clear All** to remove all entries at once.

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/voicebox` | GET | List all voice entries |
| `/api/voicebox` | POST | Create a new voice entry |
| `/api/voicebox/{entryID}` | DELETE | Delete a specific entry |
| `/api/voicebox` | DELETE | Delete all entries |

See the [API Reference]({{< ref "reference/api" >}}) for full details.
