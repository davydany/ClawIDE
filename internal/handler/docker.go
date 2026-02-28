package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/davydany/ClawIDE/internal/docker"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// dockerStatusResponse is the JSON envelope for the combined Docker status endpoint.
type dockerStatusResponse struct {
	DaemonRunning   bool                         `json:"daemon_running"`
	ComposeFile     bool                         `json:"compose_file"`
	Services        []model.DockerService        `json:"services"`
	ComposeServices []docker.ComposeServiceDetail `json:"compose_services"`
	WebAppURL       string                       `json:"web_app_url"`
	Error           string                       `json:"error,omitempty"`
}

// DockerStatus returns a combined JSON response with daemon health, compose
// file presence, running services, compose service details, OS ports, and
// an optional web app URL.
func (h *Handlers) DockerStatus(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	resp := dockerStatusResponse{}

	resp.DaemonRunning = docker.IsDockerRunning()
	resp.ComposeFile = docker.HasComposeFile(project.Path)

	// Running services (from docker compose ps). PS() returns services
	// even when the command exits non-zero (e.g. unhealthy containers),
	// so we always use whatever data came back.
	if resp.DaemonRunning && resp.ComposeFile {
		services, err := docker.PS(project.Path)
		if err != nil {
			log.Printf("DockerStatus PS error for project %s: %v", project.ID, err)
			resp.Error = err.Error()
		}
		if len(services) > 0 {
			resp.Services = docker.ToDockerServices(services)
		}
	}
	if resp.Services == nil {
		resp.Services = []model.DockerService{}
	}

	// Compose service details (parsed from YAML)
	if resp.ComposeFile {
		cfg, err := docker.ParseComposeFile(project.Path)
		if err != nil {
			log.Printf("DockerStatus compose parse error for project %s: %v", project.ID, err)
		} else {
			resp.ComposeServices = docker.ExtractServiceDetails(cfg)
		}
	}
	if resp.ComposeServices == nil {
		resp.ComposeServices = []docker.ComposeServiceDetail{}
	}

	// Web app URL
	resp.WebAppURL = docker.FindWebAppURL(project.Path)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// DockerRestart runs `docker compose restart` and returns a status response.
func (h *Handlers) DockerRestart(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	if !docker.HasComposeFile(project.Path) {
		http.Error(w, "No compose file found", http.StatusBadRequest)
		return
	}

	if err := docker.Restart(project.Path); err != nil {
		log.Printf("DockerRestart error for project %s: %v", project.ID, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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

	tail := 0
	if n, err := strconv.Atoi(r.URL.Query().Get("tail")); err == nil && n > 0 {
		tail = n
	}

	reader, err := docker.LogsStream(ctx, project.Path, svc, tail)
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

// DockerBuildWS is a WebSocket handler that streams Docker Compose build output
// for a specific service. The project ID and service name are extracted from the
// URL path parameters (defined in routes.go as /ws/docker/{projectID}/build/{svc}).
func (h *Handlers) DockerBuildWS(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("DockerBuildWS upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Drain reads from the client side. When the connection closes, cancel
	// the context to stop the build process.
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	reader, err := docker.BuildStream(ctx, project.Path, svc)
	if err != nil {
		log.Printf("DockerBuildWS stream error for %s/%s: %v", projectID, svc, err)
		conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
		return
	}

	// Stream build output line by line to the WebSocket client.
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(line+"\n")); err != nil {
			reader.Close()
			return
		}
	}

	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		log.Printf("DockerBuildWS scanner error for %s/%s: %v", projectID, svc, err)
	}

	// Close the reader to wait on the process and capture the exit status.
	reader.Close()

	// If the client disconnected, no point sending a status message.
	if ctx.Err() != nil {
		return
	}

	if reader.WaitErr() != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("[Build failed]\n"))
	} else {
		conn.WriteMessage(websocket.TextMessage, []byte("[Build complete]\n"))
	}
}
