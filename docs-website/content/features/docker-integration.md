---
title: "Docker Integration"
description: "Manage Docker Compose services, view status, and stream logs from ClawIDE."
weight: 40
---

ClawIDE integrates with Docker Compose to let you manage your project's services directly from the IDE. Start and stop containers, view service status, and stream logs — all without switching to a terminal.

{{< screenshot src="docker-integration.png" alt="ClawIDE Docker Integration" caption="Docker Compose service management with status badges and log streaming" >}}

## Prerequisites

- Docker and Docker Compose installed on your system
- A `docker-compose.yml` file in your project directory
- If running ClawIDE in Docker, the Docker socket must be mounted (`/var/run/docker.sock`)

## Viewing Service Status

1. Open a project that contains a `docker-compose.yml`.
2. Navigate to the Docker panel in the project workspace.
3. ClawIDE runs `docker compose ps` and displays each service with a status badge showing whether it's running, stopped, or in an error state.

## Managing Services

### Start All Services

Click **Up** to start all services defined in the project's `docker-compose.yml`. This runs `docker compose up -d` in the project directory.

### Stop All Services

Click **Down** to stop and remove all containers. This runs `docker compose down`.

### Individual Service Control

For each service, you can:

- **Start** — Start a stopped service
- **Stop** — Stop a running service
- **Restart** — Restart a service (useful after configuration changes)

## Log Streaming

ClawIDE streams Docker service logs in real time via WebSocket:

1. Select a service from the Docker panel.
2. Logs appear in a streaming terminal view.
3. New log lines are pushed to the browser as they arrive.

The log streaming WebSocket endpoint is `ws://localhost:9800/ws/docker/{projectID}/logs/{service}`.

## Troubleshooting

### Docker Features Not Appearing

Verify that:
- A `docker-compose.yml` exists in your project root
- Docker is running: `docker compose ps`
- The Docker socket is accessible: `ls -la /var/run/docker.sock`

### Running ClawIDE in Docker

When running ClawIDE itself in a Docker container, mount the Docker socket:

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock
```

This allows ClawIDE to communicate with the Docker daemon on the host.
