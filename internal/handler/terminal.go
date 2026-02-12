package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/tmux"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local development
	},
}

type resizeMsg struct {
	Type string `json:"type"`
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

func (h *Handlers) TerminalWS(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	paneID := chi.URLParam(r, "paneID")

	if sessionID == "" || paneID == "" {
		http.Error(w, "session ID and pane ID required", http.StatusBadRequest)
		return
	}

	// Validate that session exists and pane is in its layout
	sess, ok := h.store.GetSession(sessionID)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if sess.Layout == nil || !sess.Layout.HasPane(paneID) {
		http.Error(w, "pane not found in session layout", http.StatusNotFound)
		return
	}

	// Get or create PTY session keyed by paneID
	ptySess, ok := h.ptyManager.GetSession(paneID)
	if !ok {
		tmuxName := "clawide-" + paneID
		isNewSession := !tmux.HasSession(tmuxName)

		env := map[string]string{
			"CLAWIDE_PROJECT_ID": sess.ProjectID,
			"CLAWIDE_SESSION_ID": sess.ID,
			"CLAWIDE_PANE_ID":    paneID,
			"CLAWIDE_API_URL":    fmt.Sprintf("http://localhost:%d", h.cfg.Port),
		}
		if sess.FeatureID != "" {
			env["CLAWIDE_FEATURE_ID"] = sess.FeatureID
		}

		var err error
		ptySess, err = h.ptyManager.CreateSession(paneID, sess.WorkDir, env)
		if err != nil {
			log.Printf("Failed to create PTY session for pane %s: %v", paneID, err)
			http.Error(w, "Failed to create terminal session", http.StatusInternalServerError)
			return
		}

		// Auto-launch the agent command for new agent panes.
		if isNewSession && h.cfg.AgentCommand != "" {
			paneNode, _ := sess.Layout.FindPane(paneID)
			if paneNode != nil && paneNode.EffectivePaneType() == model.PaneTypeAgent {
				agentCmd := h.cfg.AgentCommand
				if h.cfg.AgentArgs != "" {
					agentCmd = agentCmd + " " + h.cfg.AgentArgs
				}
				go func() {
					time.Sleep(300 * time.Millisecond)
					if err := tmux.SendKeys(tmuxName, agentCmd); err != nil {
						log.Printf("Failed to send agent command to %s: %v", tmuxName, err)
					}
				}()
			}
		}
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	clientID := uuid.New().String()
	dataCh, history := ptySess.Subscribe(clientID)
	defer ptySess.Unsubscribe(clientID)

	// Send scrollback history
	if len(history) > 0 {
		if err := conn.WriteMessage(websocket.BinaryMessage, history); err != nil {
			log.Printf("Failed to send scrollback: %v", err)
			return
		}
	}

	// Read from WebSocket -> write to PTY
	go func() {
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			if msgType == websocket.TextMessage {
				// Check for resize messages
				var resize resizeMsg
				if json.Unmarshal(msg, &resize) == nil && resize.Type == "resize" {
					ptySess.Resize(resize.Rows, resize.Cols)
					continue
				}
			}

			// Write to PTY
			ptySess.Write(msg)
		}
	}()

	// Read from PTY -> write to WebSocket
	for data := range dataCh {
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			return
		}
	}
}
