package pty

import (
	"fmt"
	"log"
	"sync"

	"github.com/davydany/ClawIDE/internal/tmux"
)

type Manager struct {
	mu             sync.RWMutex
	sessions       map[string]*Session // keyed by paneID
	maxSessions    int
	scrollbackSize int
	agentCommand   string
}

func NewManager(maxSessions, scrollbackSize int, agentCommand string) *Manager {
	return &Manager{
		sessions:       make(map[string]*Session),
		maxSessions:    maxSessions,
		scrollbackSize: scrollbackSize,
		agentCommand:   agentCommand,
	}
}

// AgentCommand returns the configured agent command.
func (m *Manager) AgentCommand() string {
	return m.agentCommand
}

// CreateSession creates a new PTY session backed by tmux, keyed by paneID.
func (m *Manager) CreateSession(paneID, workDir string, env map[string]string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) >= m.maxSessions {
		return nil, fmt.Errorf("max sessions (%d) reached", m.maxSessions)
	}

	if _, exists := m.sessions[paneID]; exists {
		return nil, fmt.Errorf("session %s already exists", paneID)
	}

	tmuxName := tmux.TmuxName(paneID)
	cmd, args := tmux.SessionCommand(tmuxName, workDir)
	sess := NewSession(paneID, workDir, cmd, args, m.scrollbackSize, env)

	if err := sess.Start(); err != nil {
		return nil, fmt.Errorf("starting PTY: %w", err)
	}

	m.sessions[paneID] = sess

	// Monitor session lifecycle
	go func() {
		<-sess.Done()
		m.mu.Lock()
		delete(m.sessions, paneID)
		m.mu.Unlock()
		log.Printf("PTY session %s ended", paneID)
	}()

	log.Printf("PTY session %s started (tmux: %s) in %s", paneID, tmuxName, workDir)
	return sess, nil
}

func (m *Manager) GetSession(paneID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[paneID]
	return sess, ok
}

// CloseSession closes the PTY connection (detaches tmux client) but leaves the tmux session alive.
func (m *Manager) CloseSession(paneID string) error {
	m.mu.Lock()
	sess, ok := m.sessions[paneID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("session %s not found", paneID)
	}
	delete(m.sessions, paneID)
	m.mu.Unlock()

	return sess.Close()
}

// DestroySession fully destroys a pane: closes PTY and kills the tmux session.
func (m *Manager) DestroySession(paneID string) error {
	tmuxName := tmux.TmuxName(paneID)

	m.mu.Lock()
	sess, ok := m.sessions[paneID]
	if ok {
		delete(m.sessions, paneID)
	}
	m.mu.Unlock()

	if ok {
		if err := sess.Destroy(tmuxName); err != nil {
			log.Printf("Error destroying PTY session %s: %v", paneID, err)
		}
	} else {
		// No active PTY, but tmux session may still be alive
		if tmux.HasSession(tmuxName) {
			if err := tmux.KillSession(tmuxName); err != nil {
				log.Printf("Error killing orphan tmux session %s: %v", tmuxName, err)
			}
		}
	}
	return nil
}

// CloseAll closes all PTY connections (detaches tmux clients).
// The tmux server-side sessions survive for reconnection.
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
