package handler

import (
	"encoding/json"
	"log"
	"net/http"

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
	if sessionID == "" {
		http.Error(w, "session ID required", http.StatusBadRequest)
		return
	}

	// Get or create PTY session
	ptySess, ok := h.ptyManager.GetSession(sessionID)
	if !ok {
		// Look up the session in the store to get work dir
		sess, ok := h.store.GetSession(sessionID)
		if !ok {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}

		var err error
		ptySess, err = h.ptyManager.CreateSession(sessionID, sess.WorkDir)
		if err != nil {
			log.Printf("Failed to create PTY session: %v", err)
			http.Error(w, "Failed to create terminal session", http.StatusInternalServerError)
			return
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
