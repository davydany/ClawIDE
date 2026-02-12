package sse

import (
	"sync"

	"github.com/davydany/ClawIDE/internal/model"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[string]chan *model.Notification
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]chan *model.Notification),
	}
}

func (h *Hub) Subscribe(clientID string) <-chan *model.Notification {
	ch := make(chan *model.Notification, 50)
	h.mu.Lock()
	h.clients[clientID] = ch
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(clientID string) {
	h.mu.Lock()
	if ch, ok := h.clients[clientID]; ok {
		close(ch)
		delete(h.clients, clientID)
	}
	h.mu.Unlock()
}

func (h *Hub) Broadcast(n *model.Notification) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.clients {
		select {
		case ch <- n:
		default:
			// Slow consumer, drop notification
		}
	}
}
