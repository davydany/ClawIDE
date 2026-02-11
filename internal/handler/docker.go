package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/davydany/ClawIDE/internal/docker"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// DockerPS returns the list of Docker Compose services as JSON.
func (h *Handlers) DockerPS(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	if !docker.HasComposeFile(project.Path) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]any{})
		return
	}

	services, err := docker.PS(project.Path)
	if err != nil {
		log.Printf("DockerPS error for project %s: %v", project.ID, err)
		http.Error(w, "Failed to list Docker services", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docker.ToDockerServices(services))
}

// DockerUp runs `docker compose up -d` and returns a status response.
func (h *Handlers) DockerUp(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	if !docker.HasComposeFile(project.Path) {
		http.Error(w, "No compose file found", http.StatusBadRequest)
		return
	}

	if err := docker.Up(project.Path); err != nil {
		log.Printf("DockerUp error for project %s: %v", project.ID, err)
		http.Error(w, "Failed to start Docker Compose stack", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// DockerDown runs `docker compose down` and returns a status response.
func (h *Handlers) DockerDown(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	if !docker.HasComposeFile(project.Path) {
		http.Error(w, "No compose file found", http.StatusBadRequest)
		return
	}

	if err := docker.Down(project.Path); err != nil {
		log.Printf("DockerDown error for project %s: %v", project.ID, err)
		http.Error(w, "Failed to stop Docker Compose stack", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// DockerServiceStart starts a single Docker Compose service.
func (h *Handlers) DockerServiceStart(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	svc := chi.URLParam(r, "svc")
	if svc == "" {
		http.Error(w, "service name required", http.StatusBadRequest)
		return
	}

	if err := docker.StartService(project.Path, svc); err != nil {
		log.Printf("DockerServiceStart error for %s/%s: %v", project.ID, svc, err)
		http.Error(w, "Failed to start service", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// DockerServiceStop stops a single Docker Compose service.
func (h *Handlers) DockerServiceStop(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	svc := chi.URLParam(r, "svc")
	if svc == "" {
		http.Error(w, "service name required", http.StatusBadRequest)
		return
	}

	if err := docker.StopService(project.Path, svc); err != nil {
		log.Printf("DockerServiceStop error for %s/%s: %v", project.ID, svc, err)
		http.Error(w, "Failed to stop service", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// DockerServiceRestart restarts a single Docker Compose service.
func (h *Handlers) DockerServiceRestart(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	svc := chi.URLParam(r, "svc")
	if svc == "" {
		http.Error(w, "service name required", http.StatusBadRequest)
		return
	}

	if err := docker.RestartService(project.Path, svc); err != nil {
		log.Printf("DockerServiceRestart error for %s/%s: %v", project.ID, svc, err)
		http.Error(w, "Failed to restart service", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// DockerLogsWS is a WebSocket handler that streams Docker Compose logs for a
// specific service. The project ID and service name are extracted from the URL
// path parameters (defined in routes.go as /ws/docker/{projectID}/logs/{svc}).
// It uses the same gorilla/websocket upgrader defined in terminal.go.
func (h *Handlers) DockerLogsWS(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	svc := chi.URLParam(r, "svc")

	if projectID == "" || svc == "" {
		http.Error(w, "project ID and service name required", http.StatusBadRequest)
		return
	}

	project, ok := h.store.GetProject(projectID)
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	if !docker.HasComposeFile(project.Path) {
		http.Error(w, "No compose file found", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("DockerLogsWS upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Create a cancellable context tied to the WebSocket connection lifetime.
	// When the client disconnects, the read goroutine detects the error and
	// cancels the context, which terminates the docker compose logs process.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Drain reads from the client side. When the connection closes, cancel
	// the context to stop the log streaming process.
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	reader, err := docker.LogsStream(ctx, project.Path, svc)
	if err != nil {
		log.Printf("DockerLogsWS stream error for %s/%s: %v", projectID, svc, err)
		conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
		return
	}
	defer reader.Close()

	// Stream log lines from the docker compose process to the WebSocket client.
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(line+"\n")); err != nil {
			// Client disconnected.
			return
		}
	}

	if err := scanner.Err(); err != nil {
		// Context cancellation errors are expected when the client disconnects.
		if ctx.Err() == nil {
			log.Printf("DockerLogsWS scanner error for %s/%s: %v", projectID, svc, err)
		}
	}
}
