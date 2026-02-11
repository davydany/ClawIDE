package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Path returns the PID file path within the given data directory.
func Path(dataDir string) string {
	return filepath.Join(dataDir, "clawide.pid")
}

// Read reads the PID from the given file path.
func Read(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid pid file content: %w", err)
	}
	return pid, nil
}

// Write writes the current process PID to the given file path.
func Write(path string) error {
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0644)
}

// Remove deletes the PID file at the given path.
func Remove(path string) {
	os.Remove(path)
}

// IsRunning checks if a process with the given PID is alive.
func IsRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// Kill sends SIGTERM to the process, waits up to 5 seconds for it to exit,
// then sends SIGKILL if it's still alive.
func Kill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to %d: %w", pid, err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !IsRunning(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err := proc.Signal(syscall.SIGKILL); err != nil {
		if !IsRunning(pid) {
			return nil
		}
		return fmt.Errorf("sending SIGKILL to %d: %w", pid, err)
	}

	return nil
}
