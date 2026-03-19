//go:build windows

package pidfile

import (
	"fmt"
	"os"
	"time"
)

// IsRunning checks if a process with the given PID is alive.
func IsRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Windows, FindProcess always succeeds. Try to signal it.
	err = proc.Signal(os.Signal(nil))
	// If the process exists, Signal(nil) returns "not supported" but no "process finished" error
	return err != nil && err.Error() != "OpenProcess: The parameter is incorrect."
}

// Kill terminates a process on Windows. Sends os.Kill since Windows doesn't support SIGTERM.
func Kill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}

	if err := proc.Kill(); err != nil {
		if !IsRunning(pid) {
			return nil
		}
		return fmt.Errorf("killing process %d: %w", pid, err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !IsRunning(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("process %d did not exit after kill", pid)
}
