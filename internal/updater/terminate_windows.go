//go:build windows

package updater

import "os"

// selfTerminate sends an interrupt signal to the current process to trigger graceful shutdown.
func selfTerminate() {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		os.Exit(1)
	}
	p.Signal(os.Interrupt)
}
