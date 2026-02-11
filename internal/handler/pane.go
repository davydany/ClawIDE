package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/davydany/ccmux/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type splitResponse struct {
	Layout    *model.PaneNode `json:"layout"`
	NewPaneID string          `json:"new_pane_id"`
}

type closeResponse struct {
	Layout        *model.PaneNode `json:"layout"`
	SessionClosed bool            `json:"session_closed"`
}

func (h *Handlers) SplitPane(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sid")
	paneID := chi.URLParam(r, "pid")

	sess, ok := h.store.GetSession(sessionID)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	direction := r.FormValue("direction")
	if direction != "horizontal" && direction != "vertical" {
		http.Error(w, "direction must be 'horizontal' or 'vertical'", http.StatusBadRequest)
		return
	}

	target, parent := sess.Layout.FindPane(paneID)
	if target == nil {
		http.Error(w, "pane not found", http.StatusNotFound)
		return
	}

	newPaneID := uuid.New().String()
	newLeaf := model.NewLeafPane(newPaneID)

	// Create a split node containing the original pane and the new pane
	splitNode := &model.PaneNode{
		Type:      "split",
		Direction: direction,
		Ratio:     0.5,
		First:     target.Clone(),
		Second:    newLeaf,
	}

	if parent == nil {
		// Target is root — replace root with split
		sess.Layout = splitNode
	} else {
		parent.ReplaceChild(target, splitNode)
	}

	sess.UpdatedAt = time.Now()

	if err := h.store.UpdateSession(sess); err != nil {
		log.Printf("Error updating session after split: %v", err)
		http.Error(w, "failed to save layout", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(splitResponse{
		Layout:    sess.Layout,
		NewPaneID: newPaneID,
	})
}

func (h *Handlers) ClosePane(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sid")
	paneID := chi.URLParam(r, "pid")

	sess, ok := h.store.GetSession(sessionID)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	// Destroy the tmux session for this pane
	if err := h.ptyManager.DestroySession(paneID); err != nil {
		log.Printf("Error destroying pane %s: %v", paneID, err)
	}

	// Check if this is the only pane (root is a leaf with matching ID)
	if sess.Layout.Type == "leaf" && sess.Layout.PaneID == paneID {
		// Only pane — delete the entire session
		if err := h.store.DeleteSession(sessionID); err != nil {
			log.Printf("Error deleting session: %v", err)
			http.Error(w, "failed to delete session", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(closeResponse{
			Layout:        nil,
			SessionClosed: true,
		})
		return
	}

	// Find the pane and its parent, then collapse
	target, parent := sess.Layout.FindPane(paneID)
	if target == nil {
		http.Error(w, "pane not found", http.StatusNotFound)
		return
	}

	if parent == nil {
		// Should not happen since we checked root case above
		http.Error(w, "cannot close root pane", http.StatusBadRequest)
		return
	}

	// Determine the surviving sibling
	var sibling *model.PaneNode
	if parent.First == target {
		sibling = parent.Second
	} else {
		sibling = parent.First
	}

	// Walk the tree to find what points to parent and replace it with sibling
	if sess.Layout == parent {
		// Parent is root — replace root with sibling
		sess.Layout = sibling
	} else {
		replaceNodeInTree(sess.Layout, parent, sibling)
	}

	sess.UpdatedAt = time.Now()

	if err := h.store.UpdateSession(sess); err != nil {
		log.Printf("Error updating session after close: %v", err)
		http.Error(w, "failed to save layout", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(closeResponse{
		Layout:        sess.Layout,
		SessionClosed: false,
	})
}

// replaceNodeInTree walks the tree and replaces old with replacement.
func replaceNodeInTree(root, old, replacement *model.PaneNode) {
	if root == nil || root.Type != "split" {
		return
	}
	if root.First == old {
		root.First = replacement
		return
	}
	if root.Second == old {
		root.Second = replacement
		return
	}
	replaceNodeInTree(root.First, old, replacement)
	replaceNodeInTree(root.Second, old, replacement)
}

func (h *Handlers) ResizePane(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sid")
	paneID := chi.URLParam(r, "pid")

	sess, ok := h.store.GetSession(sessionID)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	var body struct {
		Ratio float64 `json:"ratio"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if body.Ratio < 0.1 || body.Ratio > 0.9 {
		http.Error(w, "ratio must be between 0.1 and 0.9", http.StatusBadRequest)
		return
	}

	// Find the parent split node that contains this pane
	_, parent := sess.Layout.FindPane(paneID)
	if parent == nil {
		http.Error(w, "pane has no parent split to resize", http.StatusBadRequest)
		return
	}

	parent.Ratio = body.Ratio
	sess.UpdatedAt = time.Now()

	if err := h.store.UpdateSession(sess); err != nil {
		log.Printf("Error updating session after resize: %v", err)
		http.Error(w, "failed to save layout", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}
