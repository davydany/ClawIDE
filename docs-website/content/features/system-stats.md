---
title: "System Statistics"
description: "Monitor CPU, memory, network, and project statistics from the ClawIDE interface."
weight: 130
---

ClawIDE provides a system statistics view that shows real-time information about your machine's resources, active sessions, and network interfaces. This helps you monitor resource usage while running multiple Claude Code sessions.

{{< screenshot src="system-stats.png" alt="ClawIDE System Statistics" caption="System statistics showing CPU, memory, network interfaces, and project counts" >}}

## Available Metrics

### CPU

Per-core usage percentages, sampled over a 200ms window. This gives you a quick view of how much processing power your Claude Code sessions are consuming.

### Memory

- **Total** — System memory capacity
- **Used** — Current memory consumption
- **Percentage** — Usage as a percentage of total

### Network

Lists all non-loopback network interfaces with:

- Interface name
- IPv4 address
- Connection status (up/down)

Network interface information includes QR codes for quick device-to-device connection.

### Sessions and Projects

- Active tmux sessions managed by ClawIDE
- Total project count and starred project count
- Server listening port

## Accessing Statistics

System statistics are available from the ClawIDE interface. All metrics are gathered fresh on each request — there is no caching.

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/system/stats` | GET | Returns current system metrics |

The response includes CPU cores, memory usage, network interfaces, tmux session count, project statistics, and the server port.

See the [API Reference]({{< ref "reference/api" >}}) for full details.
