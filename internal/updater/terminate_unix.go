//go:build !windows

package updater

import "syscall"

// selfTerminate sends SIGTERM to the current process to trigger graceful shutdown.
func selfTerminate() {
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
}
