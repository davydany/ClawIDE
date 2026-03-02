package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	MissingEnvFiles []string                     `json:"missing_env_files,omitempty"`
	Error           string                       `json:"error,omitempty"`
}

// dockerStatusForDir returns the combined Docker status for a given directory.
func dockerStatusForDir(dir, label string) dockerStatusResponse {
	resp := dockerStatusResponse{}
	resp.DaemonRunning = docker.IsDockerRunning()
	resp.ComposeFile = docker.HasComposeFile(dir)

	if resp.DaemonRunning && resp.ComposeFile {
		services, err := docker.PS(dir)
		if err != nil {
			log.Printf("DockerStatus PS error for %s: %v", label, err)
			resp.Error = err.Error()
		}
		if len(services) > 0 {
			resp.Services = docker.ToDockerServices(services)
		}
	}
	if resp.Services == nil {
		resp.Services = []model.DockerService{}
	}

	if resp.ComposeFile {
		cfg, err := docker.ParseComposeFile(dir)
		if err != nil {
			log.Printf("DockerStatus compose parse error for %s: %v", label, err)
		} else {
			resp.ComposeServices = docker.ExtractServiceDetails(cfg)
		}
	}
	if resp.ComposeServices == nil {
		resp.ComposeServices = []docker.ComposeServiceDetail{}
	}

	resp.WebAppURL = docker.FindWebAppURL(dir)

	// Check for missing env files referenced by the compose file.
	if resp.ComposeFile {
		resp.MissingEnvFiles = docker.FindMissingEnvFiles(dir)
	}

	return resp
}

// DockerStatus returns a combined JSON response with daemon health, compose
// file presence, running services, compose service details, OS ports, and
// an optional web app URL.
func (h *Handlers) DockerStatus(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	resp := dockerStatusForDir(project.Path, project.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// dockerPS is the shared implementation for listing Docker Compose services.
func dockerPS(w http.ResponseWriter, dir, label string) {
	if !docker.HasComposeFile(dir) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]any{})
		return
	}

	services, err := docker.PS(dir)
	if err != nil {
		log.Printf("DockerPS error for %s: %v", label, err)
		http.Error(w, "Failed to list Docker services", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docker.ToDockerServices(services))
}

// dockerComposeAction runs a compose-level action (up/down/restart) in the given dir.
func dockerComposeAction(w http.ResponseWriter, dir, label, action string, fn func(string) error) {
	if !docker.HasComposeFile(dir) {
		http.Error(w, "No compose file found", http.StatusBadRequest)
		return
	}

	if err := fn(dir); err != nil {
		log.Printf("Docker%s error for %s: %v", action, label, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// dockerServiceAction runs a per-service action (start/stop/restart) in the given dir.
func dockerServiceAction(w http.ResponseWriter, r *http.Request, dir, label string, fn func(string, string) error) {
	svc := chi.URLParam(r, "svc")
	if svc == "" {
		http.Error(w, "service name required", http.StatusBadRequest)
		return
	}

	if err := fn(dir, svc); err != nil {
		log.Printf("DockerService error for %s/%s: %v", label, svc, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// DockerPS returns the list of Docker Compose services as JSON.
func (h *Handlers) DockerPS(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	dockerPS(w, project.Path, project.ID)
}

// DockerUp runs `docker compose up -d` and returns a status response.
func (h *Handlers) DockerUp(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	dockerComposeAction(w, project.Path, project.ID, "Up", docker.Up)
}

// DockerDown runs `docker compose down` and returns a status response.
func (h *Handlers) DockerDown(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	dockerComposeAction(w, project.Path, project.ID, "Down", docker.Down)
}

// DockerRestart runs `docker compose restart` and returns a status response.
func (h *Handlers) DockerRestart(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	dockerComposeAction(w, project.Path, project.ID, "Restart", docker.Restart)
}

// DockerServiceStart starts a single Docker Compose service.
func (h *Handlers) DockerServiceStart(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	dockerServiceAction(w, r, project.Path, project.ID, docker.StartService)
}

// DockerServiceStop stops a single Docker Compose service.
func (h *Handlers) DockerServiceStop(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	dockerServiceAction(w, r, project.Path, project.ID, docker.StopService)
}

// DockerServiceRestart restarts a single Docker Compose service.
func (h *Handlers) DockerServiceRestart(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	dockerServiceAction(w, r, project.Path, project.ID, docker.RestartService)
}

// dockerLogsWSForDir streams Docker Compose logs for a service in the given dir.
func dockerLogsWSForDir(w http.ResponseWriter, r *http.Request, dir, label, svc string) {
	if !docker.HasComposeFile(dir) {
		http.Error(w, "No compose file found", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("DockerLogsWS upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

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

	reader, err := docker.LogsStream(ctx, dir, svc, tail)
	if err != nil {
		log.Printf("DockerLogsWS stream error for %s/%s: %v", label, svc, err)
		conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
		return
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(line+"\n")); err != nil {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() == nil {
			log.Printf("DockerLogsWS scanner error for %s/%s: %v", label, svc, err)
		}
	}
}

// dockerBuildWSForDir streams Docker Compose build output for a service in the given dir.
func dockerBuildWSForDir(w http.ResponseWriter, r *http.Request, dir, label, svc string) {
	if !docker.HasComposeFile(dir) {
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

	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	reader, err := docker.BuildStream(ctx, dir, svc)
	if err != nil {
		log.Printf("DockerBuildWS stream error for %s/%s: %v", label, svc, err)
		conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
		return
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(line+"\n")); err != nil {
			reader.Close()
			return
		}
	}

	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		log.Printf("DockerBuildWS scanner error for %s/%s: %v", label, svc, err)
	}

	reader.Close()

	if ctx.Err() != nil {
		return
	}

	if reader.WaitErr() != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("[Build failed]\n"))
	} else {
		conn.WriteMessage(websocket.TextMessage, []byte("[Build complete]\n"))
	}
}

// DockerLogsWS streams Docker Compose logs for a project service via WebSocket.
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

	dockerLogsWSForDir(w, r, project.Path, projectID, svc)
}

// DockerBuildWS streams Docker Compose build output for a project service via WebSocket.
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

	dockerBuildWSForDir(w, r, project.Path, projectID, svc)
}

// ─── Feature Docker Handlers ──────────────────────────────────────────────

// featureDockerDir resolves the worktree path for a feature from URL params.
func (h *Handlers) featureDockerDir(r *http.Request) (string, string, error) {
	fid := chi.URLParam(r, "fid")
	feature, ok := h.store.GetFeature(fid)
	if !ok {
		return "", "", fmt.Errorf("feature not found")
	}
	return feature.WorktreePath, "feature:" + fid, nil
}

// FeatureDockerStatus returns Docker status scoped to a feature's worktree.
func (h *Handlers) FeatureDockerStatus(w http.ResponseWriter, r *http.Request) {
	dir, label, err := h.featureDockerDir(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	resp := dockerStatusForDir(dir, label)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// FeatureDockerPS returns the running services for a feature's worktree.
func (h *Handlers) FeatureDockerPS(w http.ResponseWriter, r *http.Request) {
	dir, label, err := h.featureDockerDir(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	dockerPS(w, dir, label)
}

// FeatureDockerUp stops any running Docker stacks from the project or other
// features, then starts the stack in this feature's worktree.
func (h *Handlers) FeatureDockerUp(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	dir, label, err := h.featureDockerDir(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if !docker.HasComposeFile(dir) {
		http.Error(w, "No compose file found", http.StatusBadRequest)
		return
	}

	// Stop other running stacks: main project + other features.
	pathsToStop := []string{}
	if docker.HasComposeFile(project.Path) {
		pathsToStop = append(pathsToStop, project.Path)
	}
	for _, f := range h.store.GetFeatures(project.ID) {
		if f.WorktreePath != dir && docker.HasComposeFile(f.WorktreePath) {
			pathsToStop = append(pathsToStop, f.WorktreePath)
		}
	}

	for _, p := range pathsToStop {
		services, _ := docker.PS(p)
		if len(services) > 0 {
			log.Printf("FeatureDockerUp: stopping stack at %s before starting %s", p, label)
			if err := docker.Down(p); err != nil {
				log.Printf("FeatureDockerUp: failed to stop stack at %s: %v", p, err)
			}
		}
	}

	// Now start the feature stack.
	if err := docker.Up(dir); err != nil {
		log.Printf("FeatureDockerUp error for %s: %v", label, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// FeatureDockerDown stops the Docker stack in a feature's worktree.
func (h *Handlers) FeatureDockerDown(w http.ResponseWriter, r *http.Request) {
	dir, label, err := h.featureDockerDir(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	dockerComposeAction(w, dir, label, "Down", docker.Down)
}

// FeatureDockerRestart restarts the Docker stack in a feature's worktree.
func (h *Handlers) FeatureDockerRestart(w http.ResponseWriter, r *http.Request) {
	dir, label, err := h.featureDockerDir(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	dockerComposeAction(w, dir, label, "Restart", docker.Restart)
}

// FeatureDockerServiceStart starts a single service in a feature's worktree.
func (h *Handlers) FeatureDockerServiceStart(w http.ResponseWriter, r *http.Request) {
	dir, label, err := h.featureDockerDir(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	dockerServiceAction(w, r, dir, label, docker.StartService)
}

// FeatureDockerServiceStop stops a single service in a feature's worktree.
func (h *Handlers) FeatureDockerServiceStop(w http.ResponseWriter, r *http.Request) {
	dir, label, err := h.featureDockerDir(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	dockerServiceAction(w, r, dir, label, docker.StopService)
}

// FeatureDockerServiceRestart restarts a single service in a feature's worktree.
func (h *Handlers) FeatureDockerServiceRestart(w http.ResponseWriter, r *http.Request) {
	dir, label, err := h.featureDockerDir(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	dockerServiceAction(w, r, dir, label, docker.RestartService)
}

// FeatureDockerLogsWS streams Docker logs for a feature's service via WebSocket.
func (h *Handlers) FeatureDockerLogsWS(w http.ResponseWriter, r *http.Request) {
	fid := chi.URLParam(r, "fid")
	svc := chi.URLParam(r, "svc")

	if fid == "" || svc == "" {
		http.Error(w, "feature ID and service name required", http.StatusBadRequest)
		return
	}

	feature, ok := h.store.GetFeature(fid)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	dockerLogsWSForDir(w, r, feature.WorktreePath, "feature:"+fid, svc)
}

// FeatureDockerBuildWS streams Docker build output for a feature's service via WebSocket.
func (h *Handlers) FeatureDockerBuildWS(w http.ResponseWriter, r *http.Request) {
	fid := chi.URLParam(r, "fid")
	svc := chi.URLParam(r, "svc")

	if fid == "" || svc == "" {
		http.Error(w, "feature ID and service name required", http.StatusBadRequest)
		return
	}

	feature, ok := h.store.GetFeature(fid)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	dockerBuildWSForDir(w, r, feature.WorktreePath, "feature:"+fid, svc)
}

// FeatureDockerCopyEnvFiles copies missing .env files from the main project
// directory to the feature's worktree so that Docker Compose can run.
func (h *Handlers) FeatureDockerCopyEnvFiles(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	dir, label, err := h.featureDockerDir(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	missing := docker.FindMissingEnvFiles(dir)
	if len(missing) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"copied": []string{}, "errors": []string{}})
		return
	}

	var copied []string
	var copyErrors []string

	for _, envFile := range missing {
		srcPath := envFile
		dstPath := envFile
		if !filepath.IsAbs(envFile) {
			srcPath = filepath.Join(project.Path, envFile)
			dstPath = filepath.Join(dir, envFile)
		}

		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			copyErrors = append(copyErrors, fmt.Sprintf("%s: not found in main project", envFile))
			continue
		}

		// Ensure destination directory exists.
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			copyErrors = append(copyErrors, fmt.Sprintf("%s: %v", envFile, err))
			continue
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			log.Printf("FeatureDockerCopyEnvFiles: failed to copy %s for %s: %v", envFile, label, err)
			copyErrors = append(copyErrors, fmt.Sprintf("%s: %v", envFile, err))
			continue
		}

		copied = append(copied, envFile)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"copied": copied, "errors": copyErrors})
}

// copyFile copies a file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
