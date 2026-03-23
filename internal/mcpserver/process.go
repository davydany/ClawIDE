package mcpserver

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const defaultMaxLogLines = 500

// RingBuffer is a thread-safe circular buffer that stores the last N lines.
type RingBuffer struct {
	lines []string
	mu    sync.Mutex
	max   int
}

// NewRingBuffer creates a ring buffer with the given max line count.
func NewRingBuffer(max int) *RingBuffer {
	return &RingBuffer{max: max}
}

// Append adds a line to the buffer, evicting the oldest if at capacity.
func (rb *RingBuffer) Append(line string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if len(rb.lines) >= rb.max {
		rb.lines = rb.lines[1:]
	}
	rb.lines = append(rb.lines, line)
}

// Lines returns a copy of all stored lines.
func (rb *RingBuffer) Lines() []string {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	cp := make([]string, len(rb.lines))
	copy(cp, rb.lines)
	return cp
}

// Clear resets the buffer.
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.lines = nil
}

// Process represents a running (or previously run) MCP server process.
type Process struct {
	Name      string
	Scope     string
	Config    MCPServerConfig
	Cmd       *exec.Cmd
	Status    string // "running", "stopped", "error"
	StartedAt time.Time
	ExitCode  int
	Error     string
	Logs      *RingBuffer
	mu        sync.Mutex
	done      chan struct{} // closed when process exits
}

// ProcessInfo is the JSON-serializable representation of a process for the API.
type ProcessInfo struct {
	Status    string  `json:"status"`
	Uptime    float64 `json:"uptime_seconds,omitempty"` // seconds since start, if running
	ExitCode  int     `json:"exit_code,omitempty"`
	Error     string  `json:"error,omitempty"`
	StartedAt string  `json:"started_at,omitempty"`
}

// ProcessManager tracks all MCP server processes started by ClawIDE.
type ProcessManager struct {
	processes map[string]*Process // key: "scope:name"
	mu        sync.RWMutex
}

// NewProcessManager creates a new ProcessManager.
func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		processes: make(map[string]*Process),
	}
}

func processKey(scope, name string) string {
	return scope + ":" + name
}

// Start launches an MCP server process, capturing its output.
func (pm *ProcessManager) Start(scope, name string, config MCPServerConfig) error {
	key := processKey(scope, name)

	pm.mu.Lock()
	if existing, ok := pm.processes[key]; ok {
		existing.mu.Lock()
		if existing.Status == "running" {
			existing.mu.Unlock()
			pm.mu.Unlock()
			return fmt.Errorf("server %q is already running", name)
		}
		existing.mu.Unlock()
	}
	pm.mu.Unlock()

	if config.Command == "" {
		return fmt.Errorf("server %q has no command configured", name)
	}

	cmd := exec.Command(config.Command, config.Args...)

	// Set environment
	if len(config.Env) > 0 {
		cmd.Env = cmd.Environ()
		for k, v := range config.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	// Set process group so we can kill the whole tree
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Capture stdout and stderr into ring buffer
	logs := NewRingBuffer(defaultMaxLogLines)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting process: %w", err)
	}

	proc := &Process{
		Name:      name,
		Scope:     scope,
		Config:    config,
		Cmd:       cmd,
		Status:    "running",
		StartedAt: time.Now(),
		Logs:      logs,
		done:      make(chan struct{}),
	}

	// Read stdout and stderr concurrently
	go scanPipe(stdout, logs)
	go scanPipe(stderr, logs)

	// Wait for process exit in background
	go func() {
		waitErr := cmd.Wait()
		proc.mu.Lock()
		defer proc.mu.Unlock()
		if waitErr != nil {
			proc.Status = "error"
			proc.Error = waitErr.Error()
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				proc.ExitCode = exitErr.ExitCode()
			}
		} else {
			proc.Status = "stopped"
			proc.ExitCode = 0
		}
		close(proc.done)
	}()

	pm.mu.Lock()
	pm.processes[key] = proc
	pm.mu.Unlock()

	return nil
}

// Stop terminates a running MCP server process.
func (pm *ProcessManager) Stop(scope, name string) error {
	key := processKey(scope, name)

	pm.mu.RLock()
	proc, ok := pm.processes[key]
	pm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no process found for %q", name)
	}

	proc.mu.Lock()
	if proc.Status != "running" {
		proc.mu.Unlock()
		return fmt.Errorf("server %q is not running (status: %s)", name, proc.Status)
	}
	proc.mu.Unlock()

	// Send SIGTERM to the process group
	if proc.Cmd.Process != nil {
		pgid, err := syscall.Getpgid(proc.Cmd.Process.Pid)
		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGTERM)
		} else {
			_ = proc.Cmd.Process.Signal(syscall.SIGTERM)
		}
	}

	// Wait up to 5 seconds for graceful exit
	select {
	case <-proc.done:
		return nil
	case <-time.After(5 * time.Second):
	}

	// Force kill
	if proc.Cmd.Process != nil {
		pgid, err := syscall.Getpgid(proc.Cmd.Process.Pid)
		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = proc.Cmd.Process.Kill()
		}
	}

	// Wait for exit after kill
	select {
	case <-proc.done:
	case <-time.After(3 * time.Second):
	}

	return nil
}

// Restart stops and then starts a server.
func (pm *ProcessManager) Restart(scope, name string, config MCPServerConfig) error {
	key := processKey(scope, name)

	pm.mu.RLock()
	proc, ok := pm.processes[key]
	pm.mu.RUnlock()

	if ok {
		proc.mu.Lock()
		running := proc.Status == "running"
		proc.mu.Unlock()
		if running {
			if err := pm.Stop(scope, name); err != nil {
				return fmt.Errorf("stopping before restart: %w", err)
			}
		}
	}

	return pm.Start(scope, name, config)
}

// GetStatus returns the status info for a server.
func (pm *ProcessManager) GetStatus(scope, name string) ProcessInfo {
	key := processKey(scope, name)

	pm.mu.RLock()
	proc, ok := pm.processes[key]
	pm.mu.RUnlock()

	if !ok {
		return ProcessInfo{Status: "stopped"}
	}

	proc.mu.Lock()
	defer proc.mu.Unlock()

	info := ProcessInfo{
		Status:   proc.Status,
		ExitCode: proc.ExitCode,
		Error:    proc.Error,
	}

	if !proc.StartedAt.IsZero() {
		info.StartedAt = proc.StartedAt.Format(time.RFC3339)
	}
	if proc.Status == "running" {
		info.Uptime = time.Since(proc.StartedAt).Seconds()
	}

	return info
}

// GetLogs returns captured log lines for a server.
func (pm *ProcessManager) GetLogs(scope, name string) []string {
	key := processKey(scope, name)

	pm.mu.RLock()
	proc, ok := pm.processes[key]
	pm.mu.RUnlock()

	if !ok {
		return nil
	}
	return proc.Logs.Lines()
}

// StatusAll returns a map of scope:name -> status for all tracked processes.
func (pm *ProcessManager) StatusAll() map[string]string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]string, len(pm.processes))
	for key, proc := range pm.processes {
		proc.mu.Lock()
		result[key] = proc.Status
		proc.mu.Unlock()
	}
	return result
}

// StopAll gracefully stops all running processes.
func (pm *ProcessManager) StopAll() {
	pm.mu.RLock()
	var running []struct{ scope, name string }
	for key, proc := range pm.processes {
		proc.mu.Lock()
		if proc.Status == "running" {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 {
				running = append(running, struct{ scope, name string }{parts[0], parts[1]})
			}
		}
		proc.mu.Unlock()
	}
	pm.mu.RUnlock()

	for _, r := range running {
		_ = pm.Stop(r.scope, r.name)
	}
}

// scanPipe reads lines from a pipe and appends them to the ring buffer.
func scanPipe(pipe io.Reader, buf *RingBuffer) {
	scanner := bufio.NewScanner(pipe)
	// Increase scanner buffer for long lines
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)
	for scanner.Scan() {
		buf.Append(scanner.Text())
	}
}
