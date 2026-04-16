package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/davydany/ClawIDE/internal/aicli"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
)

// maxPromptBytes caps the size of a user prompt sent to an AI CLI. Prevents runaway argv sizes
// that could exceed ARG_MAX on some platforms, and enforces a sane upper bound on a single
// comment's payload.
const maxPromptBytes = 8 * 1024

// AskTaskAI runs an AI CLI against a user prompt and appends the response as a comment on the
// target task. Blocking request — frontend uses AbortController to cancel if needed.
//
// POST /api/tasks/{taskID}/ask-ai?project_id=<id>
// Body: {"provider": "claude", "model": "sonnet", "prompt": "..."}
func (h *Handlers) AskTaskAI(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "task ID required", http.StatusBadRequest)
		return
	}

	var body struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Prompt   string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Prompt == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}
	if len(body.Prompt) > maxPromptBytes {
		http.Error(w, "prompt exceeds maximum size (8 KB)", http.StatusBadRequest)
		return
	}

	// Look up the provider. Not registered = 400 (client sent a bad provider ID). Registered but
	// binary not on PATH = 501 so the client can flag the provider as unavailable.
	provider, ok := h.aiRegistry.Get(body.Provider)
	if !ok {
		http.Error(w, "unknown provider: "+body.Provider, http.StatusBadRequest)
		return
	}
	if !h.aiRegistry.IsInstalled(body.Provider) {
		http.Error(w, "provider "+body.Provider+" is not installed on this system", http.StatusNotImplemented)
		return
	}

	// Resolve scope + working directory. For global scope we run from the user's home dir so
	// relative paths in the prompt don't accidentally leak CWD from the server process.
	taskStore, projectID, err := h.resolveTaskStore(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	workDir := ""
	if projectID != "" {
		proj, ok := h.store.GetProject(projectID)
		if ok {
			workDir = proj.Path
		}
	}
	if workDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			workDir = home
		}
	}

	// Execute the provider. Model validation happens inside Run() via aicli.ValidateModel.
	req := aicli.Request{
		Prompt:  body.Prompt,
		Model:   body.Model,
		WorkDir: workDir,
		Timeout: 120 * time.Second,
	}
	log.Printf("AskTaskAI: provider=%s model=%s task=%s workdir=%s", body.Provider, body.Model, taskID, workDir)

	resp, err := provider.Run(r.Context(), req)
	if err != nil {
		log.Printf("AskTaskAI: provider.Run error: %v", err)
		http.Error(w, "AI provider error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Append the response as a comment. Author string encodes provider+model so hand-readers of
	// tasks.md can tell which system produced which comment.
	author := "AI/" + resp.Provider + "/" + resp.Model
	comment, err := taskStore.AppendComment(taskID, model.Comment{
		Author: author,
		Body:   resp.Text,
	})
	if err != nil {
		log.Printf("AskTaskAI: AppendComment error: %v", err)
		http.Error(w, "failed to save AI response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"comment":      comment,
		"provider":     resp.Provider,
		"model":        resp.Model,
		"duration_ms":  resp.DurationMs,
	})
}

// ListAIProviders returns the set of registered providers with their installed status and models.
// Used by the frontend Ask AI dropdown.
// GET /api/ai/providers
func (h *Handlers) ListAIProviders(w http.ResponseWriter, r *http.Request) {
	type providerResp struct {
		ID           string             `json:"id"`
		DisplayName  string             `json:"display_name"`
		Installed    bool               `json:"installed"`
		DefaultModel string             `json:"default_model"`
		Models       []aicli.ModelInfo  `json:"models"`
	}
	var out []providerResp
	for _, p := range h.aiRegistry.List() {
		models := p.AvailableModels()
		def := ""
		if len(models) > 0 {
			def = models[0].ID
		}
		out = append(out, providerResp{
			ID:           p.ID(),
			DisplayName:  p.DisplayName(),
			Installed:    h.aiRegistry.IsInstalled(p.ID()),
			DefaultModel: def,
			Models:       models,
		})
	}
	if out == nil {
		out = []providerResp{}
	}
	writeJSON(w, http.StatusOK, out)
}
