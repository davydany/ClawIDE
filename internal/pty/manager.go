package pty

import (
	"fmt"
	"log"
	"sync"
)

type Manager struct {
	mu             sync.RWMutex
	sessions       map[string]*Session
	maxSessions    int
	scrollbackSize int
	claudeCommand  string
}

func NewManager(maxSessions, scrollbackSize int, claudeCommand string) *Manager {
	return &Manager{
		sessions:       make(map[string]*Session),
		maxSessions:    maxSessions,
		scrollbackSize: scrollbackSize,
		claudeCommand:  claudeCommand,
	}
}

func (m *Manager) CreateSession(id, workDir string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) >= m.maxSessions {
		return nil, fmt.Errorf("max sessions (%d) reached", m.maxSessions)
	}

	if _, exists := m.sessions[id]; exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	// Start a shell session (bash or sh), not Claude directly
	// Users can run claude from the terminal
	sess := NewSession(id, workDir, "/bin/bash", []string{"-l"}, m.scrollbackSize)

	if err := sess.Start(); err != nil {
		return nil, fmt.Errorf("starting PTY: %w", err)
	}

	m.sessions[id] = sess

	// Monitor session lifecycle
	go func() {
		<-sess.Done()
		m.mu.Lock()
		delete(m.sessions, id)
		m.mu.Unlock()
		log.Printf("PTY session %s ended", id)
	}()

	log.Printf("PTY session %s started in %s", id, workDir)
	return sess, nil
}

func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[id]
	return sess, ok
}

func (m *Manager) CloseSession(id string) error {
	m.mu.Lock()
	sess, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("session %s not found", id)
	}
	delete(m.sessions, id)
	m.mu.Unlock()

	return sess.Close()
}

func (m *Manager) CloseAll() {
	m.mu.Lock()
	sessions := make(map[string]*Session)
	for k, v := range m.sessions {
		sessions[k] = v
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()

	for id, sess := range sessions {
		log.Printf("Closing PTY session %s", id)
		sess.Close()
	}
}

func (m *Manager) SessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
