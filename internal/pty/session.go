package pty

import (
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/davydany/ClawIDE/internal/tmux"
)

type Session struct {
	ID        string
	WorkDir   string
	Command   string
	Args      []string
	Env       map[string]string
	ptmx      *os.File
	cmd       *exec.Cmd
	mu        sync.RWMutex
	clients   map[string]chan []byte
	scrollback *RingBuffer
	done      chan struct{}
	closed    bool
}

type RingBuffer struct {
	mu   sync.Mutex
	buf  []byte
	size int
	pos  int
	full bool
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buf:  make([]byte, size),
		size: size,
	}
}

func (rb *RingBuffer) Write(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for _, b := range p {
		rb.buf[rb.pos] = b
		rb.pos = (rb.pos + 1) % rb.size
		if rb.pos == 0 {
			rb.full = true
		}
	}
	return len(p), nil
}

func (rb *RingBuffer) Bytes() []byte {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if !rb.full {
		out := make([]byte, rb.pos)
		copy(out, rb.buf[:rb.pos])
		return out
	}
	out := make([]byte, rb.size)
	copy(out, rb.buf[rb.pos:])
	copy(out[rb.size-rb.pos:], rb.buf[:rb.pos])
	return out
}

func NewSession(id, workDir, command string, args []string, scrollbackSize int, env map[string]string) *Session {
	return &Session{
		ID:         id,
		WorkDir:    workDir,
		Command:    command,
		Args:       args,
		Env:        env,
		clients:    make(map[string]chan []byte),
		scrollback: NewRingBuffer(scrollbackSize),
		done:       make(chan struct{}),
	}
}

func (s *Session) Start() error {
	s.cmd = exec.Command(s.Command, s.Args...)
	s.cmd.Dir = s.WorkDir
	s.cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	for k, v := range s.Env {
		s.cmd.Env = append(s.cmd.Env, k+"="+v)
	}

	var err error
	s.ptmx, err = pty.Start(s.cmd)
	if err != nil {
		return err
	}

	// Start fan-out goroutine
	go s.fanOut()

	return nil
}

func (s *Session) fanOut() {
	defer func() {
		close(s.done)
	}()

	buf := make([]byte, 4096)
	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			// Write to scrollback
			s.scrollback.Write(data)

			// Fan out to all clients
			s.mu.RLock()
			for _, ch := range s.clients {
				select {
				case ch <- data:
				default:
					// Client too slow, drop data
				}
			}
			s.mu.RUnlock()
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("PTY read error for session %s: %v", s.ID, err)
			}
			return
		}
	}
}

func (s *Session) Write(data []byte) (int, error) {
	if s.ptmx == nil {
		return 0, io.ErrClosedPipe
	}
	return s.ptmx.Write(data)
}

func (s *Session) Resize(rows, cols uint16) error {
	if s.ptmx == nil {
		return nil
	}
	return pty.Setsize(s.ptmx, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

func (s *Session) Subscribe(clientID string) (<-chan []byte, []byte) {
	ch := make(chan []byte, 256)

	s.mu.Lock()
	s.clients[clientID] = ch
	s.mu.Unlock()

	// Return scrollback history
	history := s.scrollback.Bytes()
	return ch, history
}

func (s *Session) Unsubscribe(clientID string) {
	s.mu.Lock()
	if ch, ok := s.clients[clientID]; ok {
		close(ch)
		delete(s.clients, clientID)
	}
	s.mu.Unlock()
}

func (s *Session) Done() <-chan struct{} {
	return s.done
}

func (s *Session) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	// Close all client channels
	s.mu.Lock()
	for id, ch := range s.clients {
		close(ch)
		delete(s.clients, id)
	}
	s.mu.Unlock()

	// Close PTY
	if s.ptmx != nil {
		s.ptmx.Close()
	}

	// Kill the process (the tmux client, not the tmux server-side session)
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}

	return nil
}

// Destroy closes the PTY session and kills the underlying tmux session.
// Use this for explicit user-initiated pane close (not graceful shutdown).
func (s *Session) Destroy(tmuxName string) error {
	if err := s.Close(); err != nil {
		log.Printf("Error closing PTY session %s: %v", s.ID, err)
	}
	// Kill the actual tmux server-side session
	if tmux.HasSession(tmuxName) {
		return tmux.KillSession(tmuxName)
	}
	return nil
}
