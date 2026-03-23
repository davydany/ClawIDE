package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/davydany/ClawIDE/internal/mcpserver"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/go-chi/chi/v5"
)

func (h *Handlers) resolveMCPFilePath(r *http.Request, scope string) string {
	switch scope {
	case "global":
		return mcpserver.GlobalMCPFilePath()
	case "project":
		project := middleware.GetProject(r)
		return mcpserver.ProjectMCPFilePath(project.Path)
	default:
		return ""
	}
}

// mcpServerResponse extends MCPServerConfig with runtime status info.
type mcpServerResponse struct {
	mcpserver.MCPServerConfig
	Status mcpserver.ProcessInfo `json:"status_info"`
}

// ListMCPServers returns all MCP servers (global + project), with runtime status.
func (h *Handlers) ListMCPServers(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	scopeFilter := r.URL.Query().Get("scope")

	globalPath := mcpserver.GlobalMCPFilePath()
	projectPath := mcpserver.ProjectMCPFilePath(project.Path)

	if scopeFilter == "global" {
		projectPath = ""
	} else if scopeFilter == "project" {
		globalPath = ""
	}

	servers, err := mcpserver.ListServers(globalPath, projectPath)
	if err != nil {
		log.Printf("Error listing MCP servers: %v", err)
		http.Error(w, "Failed to list MCP servers", http.StatusInternalServerError)
		return
	}

	if servers == nil {
		servers = []mcpserver.MCPServerConfig{}
	}

	// Enrich with runtime status
	var response []mcpServerResponse
	for _, srv := range servers {
		resp := mcpServerResponse{
			MCPServerConfig: srv,
			Status:          h.mcpProcessManager.GetStatus(srv.Scope, srv.Name),
		}
		response = append(response, resp)
	}

	if response == nil {
		response = []mcpServerResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetMCPServer returns a single MCP server by scope and name.
func (h *Handlers) GetMCPServer(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	serverName := chi.URLParam(r, "serverName")

	filePath := h.resolveMCPFilePath(r, scope)
	if filePath == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	srv, err := mcpserver.GetServer(filePath, serverName)
	if err != nil {
		log.Printf("Error getting MCP server %s/%s: %v", scope, serverName, err)
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	srv.Scope = scope
	resp := mcpServerResponse{
		MCPServerConfig: *srv,
		Status:          h.mcpProcessManager.GetStatus(scope, serverName),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CreateMCPServer adds a new MCP server to the specified scope.
func (h *Handlers) CreateMCPServer(w http.ResponseWriter, r *http.Request) {
	var srv mcpserver.MCPServerConfig
	if err := json.NewDecoder(r.Body).Decode(&srv); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if srv.Name == "" {
		http.Error(w, "Server name is required", http.StatusBadRequest)
		return
	}
	if srv.Command == "" {
		http.Error(w, "Command is required", http.StatusBadRequest)
		return
	}

	scope := srv.Scope
	if scope == "" {
		scope = "project"
	}

	filePath := h.resolveMCPFilePath(r, scope)
	if filePath == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	if err := mcpserver.CreateServer(filePath, srv); err != nil {
		log.Printf("Error creating MCP server: %v", err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// UpdateMCPServer updates an existing MCP server.
func (h *Handlers) UpdateMCPServer(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	serverName := chi.URLParam(r, "serverName")

	filePath := h.resolveMCPFilePath(r, scope)
	if filePath == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	var srv mcpserver.MCPServerConfig
	if err := json.NewDecoder(r.Body).Decode(&srv); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := mcpserver.UpdateServer(filePath, serverName, srv); err != nil {
		log.Printf("Error updating MCP server %s/%s: %v", scope, serverName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// DeleteMCPServer removes an MCP server and stops it if running.
func (h *Handlers) DeleteMCPServer(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	serverName := chi.URLParam(r, "serverName")

	filePath := h.resolveMCPFilePath(r, scope)
	if filePath == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	// Stop the process if running
	info := h.mcpProcessManager.GetStatus(scope, serverName)
	if info.Status == "running" {
		_ = h.mcpProcessManager.Stop(scope, serverName)
	}

	if err := mcpserver.DeleteServer(filePath, serverName); err != nil {
		log.Printf("Error deleting MCP server %s/%s: %v", scope, serverName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// MoveMCPServer moves an MCP server between global and project scope.
func (h *Handlers) MoveMCPServer(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	serverName := chi.URLParam(r, "serverName")

	var body struct {
		TargetScope string `json:"target_scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if body.TargetScope != "global" && body.TargetScope != "project" {
		http.Error(w, "target_scope must be 'global' or 'project'", http.StatusBadRequest)
		return
	}
	if body.TargetScope == scope {
		http.Error(w, "Server is already in that scope", http.StatusBadRequest)
		return
	}

	srcPath := h.resolveMCPFilePath(r, scope)
	dstPath := h.resolveMCPFilePath(r, body.TargetScope)
	if srcPath == "" || dstPath == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	if err := mcpserver.MoveServer(srcPath, dstPath, serverName); err != nil {
		log.Printf("Error moving MCP server %s/%s to %s: %v", scope, serverName, body.TargetScope, err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "moved", "new_scope": body.TargetScope})
}

// StartMCPServer starts an MCP server process.
func (h *Handlers) StartMCPServer(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	serverName := chi.URLParam(r, "serverName")

	filePath := h.resolveMCPFilePath(r, scope)
	if filePath == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	srv, err := mcpserver.GetServer(filePath, serverName)
	if err != nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	if err := h.mcpProcessManager.Start(scope, serverName, *srv); err != nil {
		log.Printf("Error starting MCP server %s/%s: %v", scope, serverName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// StopMCPServer stops a running MCP server process.
func (h *Handlers) StopMCPServer(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	serverName := chi.URLParam(r, "serverName")

	if err := h.mcpProcessManager.Stop(scope, serverName); err != nil {
		log.Printf("Error stopping MCP server %s/%s: %v", scope, serverName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

// RestartMCPServer restarts an MCP server process.
func (h *Handlers) RestartMCPServer(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	serverName := chi.URLParam(r, "serverName")

	filePath := h.resolveMCPFilePath(r, scope)
	if filePath == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	srv, err := mcpserver.GetServer(filePath, serverName)
	if err != nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	if err := h.mcpProcessManager.Restart(scope, serverName, *srv); err != nil {
		log.Printf("Error restarting MCP server %s/%s: %v", scope, serverName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "restarted"})
}

// MCPServerLogs returns captured log lines for an MCP server.
func (h *Handlers) MCPServerLogs(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	serverName := chi.URLParam(r, "serverName")

	logs := h.mcpProcessManager.GetLogs(scope, serverName)
	if logs == nil {
		logs = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"lines": logs,
		"count": len(logs),
	})
}

// MCPServerStatus returns the runtime status of an MCP server.
func (h *Handlers) MCPServerStatus(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	serverName := chi.URLParam(r, "serverName")

	info := h.mcpProcessManager.GetStatus(scope, serverName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// StopAllMCPProcesses stops all running MCP server processes (for graceful shutdown).
func (h *Handlers) StopAllMCPProcesses() {
	h.mcpProcessManager.StopAll()
}
