package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/davydany/ClawIDE/internal/agent"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/go-chi/chi/v5"
)

func (h *Handlers) resolveAgentsDir(r *http.Request, scope string) string {
	switch scope {
	case "global":
		return agent.GlobalAgentsDir()
	case "project":
		project := middleware.GetProject(r)
		return agent.ProjectAgentsDir(project.Path)
	default:
		return ""
	}
}

// ListAgents returns all agents (global + project), optionally filtered by scope query param.
func (h *Handlers) ListAgents(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	scopeFilter := r.URL.Query().Get("scope")

	globalDir := agent.GlobalAgentsDir()
	projectDir := agent.ProjectAgentsDir(project.Path)

	if scopeFilter == "global" {
		projectDir = ""
	} else if scopeFilter == "project" {
		globalDir = ""
	}

	all, err := agent.ListAgents(globalDir, projectDir)
	if err != nil {
		log.Printf("Error listing agents: %v", err)
		http.Error(w, "Failed to list agents", http.StatusInternalServerError)
		return
	}

	if all == nil {
		all = []agent.Agent{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(all)
}

// GetAgent returns a single agent by scope and file name.
func (h *Handlers) GetAgent(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	agentName := chi.URLParam(r, "agentName")

	baseDir := h.resolveAgentsDir(r, scope)
	if baseDir == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	ag, err := agent.GetAgent(baseDir, agentName)
	if err != nil {
		log.Printf("Error getting agent %s/%s: %v", scope, agentName, err)
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	ag.Scope = scope
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ag)
}

// CreateAgent creates a new agent in the specified scope.
func (h *Handlers) CreateAgent(w http.ResponseWriter, r *http.Request) {
	var ag agent.Agent
	if err := json.NewDecoder(r.Body).Decode(&ag); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if ag.Name == "" {
		http.Error(w, "Agent name is required", http.StatusBadRequest)
		return
	}

	scope := ag.Scope
	if scope == "" {
		scope = "project"
	}

	baseDir := h.resolveAgentsDir(r, scope)
	if baseDir == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	if err := agent.CreateAgent(baseDir, ag); err != nil {
		log.Printf("Error creating agent: %v", err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// UpdateAgent updates an existing agent.
func (h *Handlers) UpdateAgent(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	agentName := chi.URLParam(r, "agentName")

	baseDir := h.resolveAgentsDir(r, scope)
	if baseDir == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	var ag agent.Agent
	if err := json.NewDecoder(r.Body).Decode(&ag); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := agent.UpdateAgent(baseDir, agentName, ag); err != nil {
		log.Printf("Error updating agent %s/%s: %v", scope, agentName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// DeleteAgent removes an agent file.
func (h *Handlers) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	agentName := chi.URLParam(r, "agentName")

	baseDir := h.resolveAgentsDir(r, scope)
	if baseDir == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	if err := agent.DeleteAgent(baseDir, agentName); err != nil {
		log.Printf("Error deleting agent %s/%s: %v", scope, agentName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// MoveAgent moves an agent between global and project scope.
func (h *Handlers) MoveAgent(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	agentName := chi.URLParam(r, "agentName")

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
		http.Error(w, "Agent is already in that scope", http.StatusBadRequest)
		return
	}

	srcDir := h.resolveAgentsDir(r, scope)
	dstDir := h.resolveAgentsDir(r, body.TargetScope)
	if srcDir == "" || dstDir == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	if err := agent.MoveAgent(srcDir, dstDir, agentName); err != nil {
		log.Printf("Error moving agent %s/%s to %s: %v", scope, agentName, body.TargetScope, err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "moved", "new_scope": body.TargetScope})
}
