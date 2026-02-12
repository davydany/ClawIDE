package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type createNotificationRequest struct {
	Title          string `json:"title"`
	Body           string `json:"body"`
	Source         string `json:"source"`
	Level          string `json:"level"`
	ProjectID      string `json:"project_id"`
	SessionID      string `json:"session_id"`
	FeatureID      string `json:"feature_id"`
	CWD            string `json:"cwd"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *Handlers) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var req createNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	if req.Source == "" {
		req.Source = "system"
	}
	if req.Level == "" {
		req.Level = "info"
	}

	// Resolve project/feature from CWD if not provided
	if req.ProjectID == "" && req.CWD != "" {
		req.ProjectID, req.FeatureID = h.resolveProjectFromCWD(req.CWD)
	}

	n := model.Notification{
		ID:             uuid.New().String(),
		Title:          req.Title,
		Body:           req.Body,
		Source:         req.Source,
		Level:          req.Level,
		ProjectID:      req.ProjectID,
		SessionID:      req.SessionID,
		FeatureID:      req.FeatureID,
		CWD:            req.CWD,
		IdempotencyKey: req.IdempotencyKey,
		Read:           false,
		CreatedAt:      time.Now(),
	}

	if err := h.notificationStore.Add(n); err != nil {
		log.Printf("Failed to add notification: %v", err)
		http.Error(w, "failed to store notification", http.StatusInternalServerError)
		return
	}

	// Broadcast to SSE clients
	h.sseHub.Broadcast(&n)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(n)
}

func (h *Handlers) ListNotifications(w http.ResponseWriter, r *http.Request) {
	unreadOnly := r.URL.Query().Get("unread_only") == "true"

	var notifications []model.Notification
	if unreadOnly {
		notifications = h.notificationStore.GetUnread()
	} else {
		notifications = h.notificationStore.GetAll()
	}

	if notifications == nil {
		notifications = []model.Notification{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

func (h *Handlers) UnreadNotificationCount(w http.ResponseWriter, r *http.Request) {
	count := h.notificationStore.UnreadCount()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}

func (h *Handlers) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	notifID := chi.URLParam(r, "notifID")
	if err := h.notificationStore.MarkRead(notifID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	if err := h.notificationStore.MarkAllRead(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	notifID := chi.URLParam(r, "notifID")
	if err := h.notificationStore.Delete(notifID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) NotificationStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Disable the server's write timeout for SSE
	rc := http.NewResponseController(w)
	rc.SetWriteDeadline(time.Time{})

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	clientID := uuid.New().String()
	ch := h.sseHub.Subscribe(clientID)
	defer h.sseHub.Unsubscribe(clientID)

	// Send initial unread count
	count := h.notificationStore.UnreadCount()
	fmt.Fprintf(w, "event: unread-count\ndata: %d\n\n", count)
	flusher.Flush()

	// Keepalive ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case n, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(n)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: notification\ndata: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// resolveProjectFromCWD resolves project and feature IDs from a working directory path.
// It checks feature worktree paths first (more specific), then project root paths.
func (h *Handlers) resolveProjectFromCWD(cwd string) (projectID, featureID string) {
	projects := h.store.GetProjects()

	// First pass: check feature worktree paths (more specific match)
	for _, p := range projects {
		features := h.store.GetFeatures(p.ID)
		for _, f := range features {
			if f.WorktreePath != "" && strings.HasPrefix(cwd, f.WorktreePath) {
				return p.ID, f.ID
			}
		}
	}

	// Second pass: check project root paths
	for _, p := range projects {
		if p.Path != "" && strings.HasPrefix(cwd, p.Path) {
			return p.ID, ""
		}
	}

	return "", ""
}
