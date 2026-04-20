package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/davydany/ClawIDE/internal/aicli"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/go-chi/chi/v5"
)

// maxPromptBytes caps the size of a user prompt sent to an AI CLI. Prevents runaway argv sizes
// that could exceed ARG_MAX on some platforms, and enforces a sane upper bound on a single
// comment's payload.
const maxPromptBytes = 8 * 1024

// AskTaskAI runs an AI CLI against a user prompt and appends the response as a comment on the
// target task. For providers that support streaming (e.g. claude), the response is sent as
// Server-Sent Events so the frontend can show live output. For buffered providers, the full
// result is returned as JSON after the CLI finishes.
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

	provider, ok := h.aiRegistry.Get(body.Provider)
	if !ok {
		http.Error(w, "unknown provider: "+body.Provider, http.StatusBadRequest)
		return
	}
	if !h.aiRegistry.IsInstalled(body.Provider) {
		http.Error(w, "provider "+body.Provider+" is not installed on this system", http.StatusNotImplemented)
		return
	}

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

	req := aicli.Request{
		Prompt:  body.Prompt,
		Model:   body.Model,
		WorkDir: workDir,
		Timeout: 120 * time.Second,
	}
	author := "AI/" + body.Provider + "/" + body.Model
	log.Printf("AskTaskAI: provider=%s model=%s task=%s streaming=%v", body.Provider, body.Model, taskID, provider.SupportsStreaming())

	if provider.SupportsStreaming() {
		h.askTaskAIStreaming(w, r, provider, req, taskStore, taskID, author)
	} else {
		h.askTaskAIBuffered(w, r, provider, req, taskStore, taskID, author)
	}
}

// askTaskAIStreaming writes SSE events as the CLI produces output. Events:
//
//	event: chunk\ndata: <text>\n\n       — incremental text
//	event: done\ndata: <json>\n\n        — final result + saved comment
//	event: error\ndata: <message>\n\n    — terminal error
func (h *Handlers) askTaskAIStreaming(w http.ResponseWriter, r *http.Request, provider aicli.CLIProvider, req aicli.Request, taskStore *store.TaskStore, taskID, author string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		// Fallback if response doesn't support flushing (shouldn't happen with net/http).
		h.askTaskAIBuffered(w, r, provider, req, taskStore, taskID, author)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering if behind a proxy
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	var finalText string

	err := provider.RunStreaming(r.Context(), req, func(chunk aicli.StreamChunk) {
		if chunk.Error != "" {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", chunk.Error)
			flusher.Flush()
			return
		}
		if chunk.Done {
			finalText = chunk.Text
			return // we'll write the done event after saving the comment
		}
		// Incremental text chunk.
		fmt.Fprintf(w, "event: chunk\ndata: %s\n\n", chunk.Text)
		flusher.Flush()
	})

	if err != nil {
		log.Printf("AskTaskAI streaming error: %v", err)
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	// Save as comment.
	if finalText == "" {
		fmt.Fprintf(w, "event: error\ndata: no output from provider\n\n")
		flusher.Flush()
		return
	}
	comment, err := taskStore.AppendComment(taskID, model.Comment{
		Author: author,
		Body:   finalText,
	})
	if err != nil {
		log.Printf("AskTaskAI: AppendComment error: %v", err)
		fmt.Fprintf(w, "event: error\ndata: failed to save: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	doneData, _ := json.Marshal(map[string]any{
		"comment":  comment,
		"provider": provider.ID(),
		"model":    req.Model,
	})
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", string(doneData))
	flusher.Flush()
}

// askTaskAIBuffered is the original blocking path for providers that don't stream.
func (h *Handlers) askTaskAIBuffered(w http.ResponseWriter, r *http.Request, provider aicli.CLIProvider, req aicli.Request, taskStore *store.TaskStore, taskID, author string) {
	resp, err := provider.Run(r.Context(), req)
	if err != nil {
		log.Printf("AskTaskAI: provider.Run error: %v", err)
		http.Error(w, "AI provider error: "+err.Error(), http.StatusBadGateway)
		return
	}

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
		"comment":     comment,
		"provider":    resp.Provider,
		"model":       resp.Model,
		"duration_ms": resp.DurationMs,
	})
}

// ListAIProviders returns the set of registered providers with their installed status and models.
// Used by the frontend Ask AI dropdown.
// GET /api/ai/providers
func (h *Handlers) ListAIProviders(w http.ResponseWriter, r *http.Request) {
	type providerResp struct {
		ID               string            `json:"id"`
		DisplayName      string            `json:"display_name"`
		Installed        bool              `json:"installed"`
		SupportsStreaming bool             `json:"supports_streaming"`
		DefaultModel     string            `json:"default_model"`
		Models           []aicli.ModelInfo `json:"models"`
	}
	var out []providerResp
	for _, p := range h.aiRegistry.List() {
		models := p.AvailableModels()
		def := ""
		if len(models) > 0 {
			def = models[0].ID
		}
		out = append(out, providerResp{
			ID:               p.ID(),
			DisplayName:      p.DisplayName(),
			Installed:        h.aiRegistry.IsInstalled(p.ID()),
			SupportsStreaming: p.SupportsStreaming(),
			DefaultModel:     def,
			Models:           models,
		})
	}
	if out == nil {
		out = []providerResp{}
	}
	writeJSON(w, http.StatusOK, out)
}
