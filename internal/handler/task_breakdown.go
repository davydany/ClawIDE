package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/aicli"
	"github.com/davydany/ClawIDE/internal/breakdown"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/go-chi/chi/v5"
)

// checklistItemPrefix is the exact prefix every line of a valid AI-generated checklist must
// start with. Anything else (code fences, prose, stray headings) gets filtered out before we
// write the file.
const checklistItemPrefix = "- [ ] "

// BreakdownTask asks the configured AI provider to turn a task's title + description into a
// short checklist, writes it to <worktree>/tasks/<slug>.md, updates <worktree>/CLAUDE.md's
// managed region to reference the file, and appends an audit comment to the task. Streams
// tokens back as SSE so the UI can show live output.
//
// POST /api/tasks/{taskID}/breakdown?project_id=<id>
// Body: {"provider":"claude","model":"sonnet","overwrite":true}
// Resp: text/event-stream — events: chunk, done, error
//
//	done data: {"file_path":"tasks/<slug>.md","claude_md_updated":true,"comment":{...}}
func (h *Handlers) BreakdownTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "task ID required", http.StatusBadRequest)
		return
	}

	var body struct {
		Provider  string `json:"provider"`
		Model     string `json:"model"`
		Overwrite *bool  `json:"overwrite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	overwrite := true
	if body.Overwrite != nil {
		overwrite = *body.Overwrite
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

	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "breakdown requires project scope (project_id query param)", http.StatusBadRequest)
		return
	}
	project, ok := h.store.GetProject(projectID)
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}
	taskStore, err := h.getProjectTaskStore(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	board, err := taskStore.Board()
	if err != nil {
		http.Error(w, "load board: "+err.Error(), http.StatusInternalServerError)
		return
	}
	task, _, _, _ := board.FindTask(taskID)
	if task == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if task.LinkedBranch == "" {
		http.Error(w, "task is not linked to a branch", http.StatusConflict)
		return
	}

	worktreePath, ok := worktreePathForBranch(project.Path, task.LinkedBranch)
	if !ok {
		http.Error(w, "linked branch has no worktree checkout: "+task.LinkedBranch, http.StatusConflict)
		return
	}
	if info, err := os.Stat(worktreePath); err != nil || !info.IsDir() {
		http.Error(w, "worktree path does not exist: "+worktreePath, http.StatusConflict)
		return
	}

	prompt := buildBreakdownPrompt(*task)
	if len(prompt) > maxPromptBytes {
		http.Error(w, "prompt exceeds maximum size (8 KB)", http.StatusBadRequest)
		return
	}

	req := aicli.Request{
		Prompt:  prompt,
		Model:   body.Model,
		WorkDir: worktreePath,
		Timeout: 120 * time.Second,
	}
	author := "AI/" + body.Provider + "/" + body.Model
	log.Printf("BreakdownTask: task=%s branch=%s worktree=%s provider=%s model=%s", taskID, task.LinkedBranch, worktreePath, body.Provider, body.Model)

	if provider.SupportsStreaming() {
		h.breakdownStreaming(w, r, provider, req, taskStore, *task, worktreePath, author, overwrite)
	} else {
		h.breakdownBuffered(w, r, provider, req, taskStore, *task, worktreePath, author, overwrite)
	}
}

// buildBreakdownPrompt assembles the prompt sent to the AI CLI. Kept tight so the model
// leaves prose out and emits only the checklist we want to persist verbatim.
func buildBreakdownPrompt(task model.Task) string {
	desc := strings.TrimSpace(task.Description)
	if desc == "" {
		desc = "(none)"
	}
	return fmt.Sprintf(`You are breaking down a development task into a short actionable checklist.

Task title: %s
Task description:
%s

Output ONLY a GitHub-flavored markdown checklist: between 3 and 7 items, each a single actionable line starting with "- [ ] ". No preamble, no headings, no commentary, no code fences. Each item should be a concrete verb-led step a developer can complete in one sitting.`, task.Title, desc)
}

func (h *Handlers) breakdownStreaming(w http.ResponseWriter, r *http.Request, provider aicli.CLIProvider, req aicli.Request, taskStore *store.TaskStore, task model.Task, worktreePath, author string, overwrite bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.breakdownBuffered(w, r, provider, req, taskStore, task, worktreePath, author, overwrite)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
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
			return
		}
		fmt.Fprintf(w, "event: chunk\ndata: %s\n\n", chunk.Text)
		flusher.Flush()
	})
	if err != nil {
		log.Printf("BreakdownTask streaming error: %v", err)
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	h.finalizeBreakdown(w, flusher, taskStore, task, worktreePath, finalText, author, overwrite)
}

func (h *Handlers) breakdownBuffered(w http.ResponseWriter, r *http.Request, provider aicli.CLIProvider, req aicli.Request, taskStore *store.TaskStore, task model.Task, worktreePath, author string, overwrite bool) {
	resp, err := provider.Run(r.Context(), req)
	if err != nil {
		log.Printf("BreakdownTask: provider.Run error: %v", err)
		http.Error(w, "AI provider error: "+err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	if flusher != nil {
		fmt.Fprintf(w, "event: chunk\ndata: %s\n\n", resp.Text)
		flusher.Flush()
	}
	h.finalizeBreakdown(w, flusher, taskStore, task, worktreePath, resp.Text, author, overwrite)
}

// finalizeBreakdown is the tail of the streaming/buffered paths. Filters the AI output down
// to checklist lines, writes the subtask file, updates CLAUDE.md, records an audit comment,
// and emits the terminal SSE event.
func (h *Handlers) finalizeBreakdown(w http.ResponseWriter, flusher http.Flusher, taskStore *store.TaskStore, task model.Task, worktreePath, rawText, author string, overwrite bool) {
	writeErr := func(msg string) {
		if flusher != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", msg)
			flusher.Flush()
		} else {
			http.Error(w, msg, http.StatusInternalServerError)
		}
	}

	checklist := extractChecklist(rawText)
	if checklist == "" {
		writeErr("AI output did not contain a valid checklist")
		return
	}

	slug := breakdown.TaskSlug(task.Title, task.ID)
	// Defense-in-depth: even though TaskSlug is [a-z0-9-] only, re-validate that the resolved
	// path stays inside the worktree.
	if _, ok := resolveAndValidatePath(worktreePath, filepath.Join("tasks", slug+".md")); !ok {
		writeErr("generated slug failed path validation")
		return
	}

	filePath, err := breakdown.WriteSubtaskFile(worktreePath, slug, task.ID, task.Title, checklist, overwrite)
	if err == breakdown.ErrExists {
		writeErr("subtask file already exists (pass overwrite:true to regenerate)")
		return
	}
	if err != nil {
		writeErr("write subtask file: " + err.Error())
		return
	}
	relPath, _ := filepath.Rel(worktreePath, filePath)

	claudeMDUpdated := true
	if err := breakdown.UpdateClaudeMD(worktreePath, filepath.Base(strings.TrimSuffix(filePath, ".md")), task.Title); err != nil {
		log.Printf("BreakdownTask: UpdateClaudeMD error: %v", err)
		claudeMDUpdated = false
	}

	commentBody := fmt.Sprintf("Broke down into %s (worktree: %s)", relPath, worktreePath)
	comment, err := taskStore.AppendComment(task.ID, model.Comment{Author: author, Body: commentBody})
	if err != nil {
		log.Printf("BreakdownTask: AppendComment error: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"file_path":         relPath,
		"claude_md_updated": claudeMDUpdated,
		"worktree_path":     worktreePath,
		"comment":           comment,
	})
	if flusher != nil {
		fmt.Fprintf(w, "event: done\ndata: %s\n\n", string(payload))
		flusher.Flush()
	} else {
		writeJSON(w, http.StatusOK, json.RawMessage(payload))
	}
}

// extractChecklist drops everything outside the checklist. Tolerant of a leading preamble
// or code fence (the model sometimes ignores instructions), and stops at the first non-
// checklist, non-blank trailing line. Returns "" if no valid items are found.
func extractChecklist(raw string) string {
	lines := strings.Split(raw, "\n")
	var kept []string
	started := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, checklistItemPrefix) || strings.HasPrefix(trimmed, "- [x] ") || strings.HasPrefix(trimmed, "- [X] ") {
			kept = append(kept, trimmed)
			started = true
			continue
		}
		if !started {
			continue
		}
		// After we've started collecting items, tolerate blank lines but stop at any other prose.
		if trimmed == "" {
			continue
		}
		break
	}
	if len(kept) == 0 {
		return ""
	}
	return strings.Join(kept, "\n")
}
