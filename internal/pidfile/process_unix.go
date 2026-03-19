//go:build !windows

package pidfile

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

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
