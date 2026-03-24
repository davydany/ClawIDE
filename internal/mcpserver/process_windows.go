//go:build windows

package mcpserver

import (
	"os/exec"
	"syscall"
)

// setProcGroup configures the command to create a new process group on Windows.
func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// termProcess sends a graceful termination signal.
// On Windows, there's no SIGTERM equivalent for console apps, so we just kill.
func termProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// killProcess forcefully kills the process.
func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
