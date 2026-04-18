package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// --------------- Folders ---------------

func (h *Handlers) ListPromptForgeFolders(w http.ResponseWriter, r *http.Request) {
	folders := h.promptForgeStore.GetFolders()
	if folders == nil {
		folders = []model.Folder{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(folders); err != nil {
		log.Printf("promptforge folders list encode error: %v", err)
	}
}

func (h *Handlers) CreatePromptForgeFolder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id"`
		Order    int    `json:"order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if err := model.ValidateFolderName(body.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now()
	f := model.Folder{
		ID:        uuid.New().String(),
		Name:      body.Name,
		ParentID:  body.ParentID,
		Order:     body.Order,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.promptForgeStore.CreateFolder(f); err != nil {
		log.Printf("promptforge folder create error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(f)
}

func (h *Handlers) UpdatePromptForgeFolder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "folderID")
	existing, ok := h.promptForgeStore.GetFolder(id)
	if !ok {
		http.Error(w, "folder not found", http.StatusNotFound)
		return
	}

	var body struct {
		Name     string  `json:"name"`
		ParentID *string `json:"parent_id"`
		Order    *int    `json:"order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name != "" {
		if err := model.ValidateFolderName(body.Name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		existing.Name = body.Name
	}
	if body.ParentID != nil {
		existing.ParentID = *body.ParentID
	}
	if body.Order != nil {
		existing.Order = *body.Order
	}
	existing.UpdatedAt = time.Now()

	if err := h.promptForgeStore.UpdateFolder(existing); err != nil {
		log.Printf("promptforge folder update error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(existing)
}

func (h *Handlers) DeletePromptForgeFolder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "folderID")
	cascade := strings.EqualFold(r.URL.Query().Get("cascade"), "true")
	if err := h.promptForgeStore.DeleteFolder(id, cascade); err != nil {
		msg := err.Error()
		status := http.StatusBadRequest
		if strings.Contains(msg, "not found") {
			status = http.StatusNotFound
		} else if strings.Contains(msg, "not empty") {
			status = http.StatusConflict
		}
		http.Error(w, msg, status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --------------- Prompts ---------------

func (h *Handlers) ListPromptForgePrompts(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	folderID := r.URL.Query().Get("folder_id")
	hasFolderFilter := r.URL.Query().Has("folder_id")

	var prompts []model.Prompt
	switch {
	case q != "":
		prompts = h.promptForgeStore.SearchPrompts(q)
	case hasFolderFilter:
		prompts = h.promptForgeStore.GetPromptsByFolder(folderID)
	default:
		prompts = h.promptForgeStore.GetAllPrompts()
	}

	if prompts == nil {
		prompts = []model.Prompt{}
	}
	// Omit body content in listings to keep payloads small.
	summaries := make([]promptSummary, 0, len(prompts))
	for _, p := range prompts {
		summaries = append(summaries, summarize(p))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summaries)
}

func (h *Handlers) GetPromptForgePrompt(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "promptID")
	p, ok := h.promptForgeStore.GetPrompt(id)
	if !ok {
		http.Error(w, "prompt not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(p)
}

func (h *Handlers) CreatePromptForgePrompt(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FolderID  string           `json:"folder_id"`
		Title     string           `json:"title"`
		Type      model.PromptType `json:"type"`
		Variables []model.Variable `json:"variables"`
		Content   string           `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Type == "" {
		body.Type = model.PromptTypePlain
	}
	if err := validatePromptBody(body.Title, body.Type, body.Variables); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now()
	p := model.Prompt{
		ID:        uuid.New().String(),
		FolderID:  body.FolderID,
		Title:     body.Title,
		Type:      body.Type,
		Variables: body.Variables,
		Content:   body.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.promptForgeStore.AddPrompt(p); err != nil {
		log.Printf("promptforge prompt create error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}

func (h *Handlers) UpdatePromptForgePrompt(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "promptID")
	existing, ok := h.promptForgeStore.GetPrompt(id)
	if !ok {
		http.Error(w, "prompt not found", http.StatusNotFound)
		return
	}

	var body struct {
		FolderID  *string           `json:"folder_id"`
		Title     string            `json:"title"`
		Type      model.PromptType  `json:"type"`
		Variables *[]model.Variable `json:"variables"`
		Content   *string           `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Title != "" {
		if err := model.ValidatePromptTitle(body.Title); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		existing.Title = body.Title
	}
	if body.Type != "" {
		if err := model.ValidatePromptType(body.Type); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		existing.Type = body.Type
	}
	if body.Variables != nil {
		if err := model.ValidateVariables(*body.Variables); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		existing.Variables = *body.Variables
	}
	if body.FolderID != nil {
		existing.FolderID = *body.FolderID
	}
	if body.Content != nil {
		existing.Content = *body.Content
	}
	existing.UpdatedAt = time.Now()

	if err := h.promptForgeStore.UpdatePrompt(existing); err != nil {
		log.Printf("promptforge prompt update error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(existing)
}

func (h *Handlers) DeletePromptForgePrompt(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "promptID")
	if err := h.promptForgeStore.DeletePrompt(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --------------- Compiled Versions ---------------

func (h *Handlers) ListPromptForgeVersions(w http.ResponseWriter, r *http.Request) {
	promptID := chi.URLParam(r, "promptID")
	versions, err := h.promptForgeStore.GetVersions(promptID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	// Hide the body in listings to keep payloads small.
	summaries := make([]versionSummary, 0, len(versions))
	for _, v := range versions {
		summaries = append(summaries, versionSummary{
			ID:         v.ID,
			PromptID:   v.PromptID,
			Title:      v.Title,
			CompiledAt: v.CompiledAt,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summaries)
}

func (h *Handlers) GetPromptForgeVersion(w http.ResponseWriter, r *http.Request) {
	promptID := chi.URLParam(r, "promptID")
	versionID := chi.URLParam(r, "versionID")
	v, err := h.promptForgeStore.GetVersion(promptID, versionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handlers) CreatePromptForgeVersion(w http.ResponseWriter, r *http.Request) {
	promptID := chi.URLParam(r, "promptID")
	if _, ok := h.promptForgeStore.GetPrompt(promptID); !ok {
		http.Error(w, "prompt not found", http.StatusNotFound)
		return
	}
	var body struct {
		Title          string                 `json:"title"`
		VariableValues map[string]interface{} `json:"variable_values"`
		Content        string                 `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	now := time.Now()
	title := strings.TrimSpace(body.Title)
	if title == "" {
		title = now.Format("2006-01-02 15:04:05")
	}
	if err := model.ValidateVersionTitle(title); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	v := model.CompiledVersion{
		ID:             uuid.New().String(),
		PromptID:       promptID,
		Title:          title,
		VariableValues: body.VariableValues,
		Content:        body.Content,
		CompiledAt:     now,
	}
	if err := h.promptForgeStore.AddVersion(v); err != nil {
		log.Printf("promptforge version create error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handlers) UpdatePromptForgeVersion(w http.ResponseWriter, r *http.Request) {
	promptID := chi.URLParam(r, "promptID")
	versionID := chi.URLParam(r, "versionID")
	existing, err := h.promptForgeStore.GetVersion(promptID, versionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Title != "" {
		if err := model.ValidateVersionTitle(body.Title); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		existing.Title = body.Title
	}
	if err := h.promptForgeStore.UpdateVersion(existing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(existing)
}

func (h *Handlers) DeletePromptForgeVersion(w http.ResponseWriter, r *http.Request) {
	promptID := chi.URLParam(r, "promptID")
	versionID := chi.URLParam(r, "versionID")
	if err := h.promptForgeStore.DeleteVersion(promptID, versionID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --------------- internals ---------------

func validatePromptBody(title string, t model.PromptType, vars []model.Variable) error {
	if err := model.ValidatePromptTitle(title); err != nil {
		return err
	}
	if err := model.ValidatePromptType(t); err != nil {
		return err
	}
	return model.ValidateVariables(vars)
}

// promptSummary strips the markdown body so that list responses stay small.
type promptSummary struct {
	ID        string           `json:"id"`
	FolderID  string           `json:"folder_id,omitempty"`
	Title     string           `json:"title"`
	Type      model.PromptType `json:"type"`
	Variables []model.Variable `json:"variables,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

func summarize(p model.Prompt) promptSummary {
	return promptSummary{
		ID:        p.ID,
		FolderID:  p.FolderID,
		Title:     p.Title,
		Type:      p.Type,
		Variables: p.Variables,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

type versionSummary struct {
	ID         string    `json:"id"`
	PromptID   string    `json:"prompt_id"`
	Title      string    `json:"title"`
	CompiledAt time.Time `json:"compiled_at"`
}
