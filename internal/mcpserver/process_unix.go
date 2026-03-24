//go:build !windows

package mcpserver

import (
	"os/exec"
	"syscall"
)

// setProcGroup configures the command to run in its own process group.
func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// signalProcess sends a signal to the process group, falling back to the process itself.
func signalProcess(cmd *exec.Cmd, sig syscall.Signal) {
	if cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, sig)
	} else {
		if sig == syscall.SIGKILL {
			_ = cmd.Process.Kill()
		} else {
			_ = cmd.Process.Signal(sig)
		}
	}
}

// termProcess sends SIGTERM to the process group.
func termProcess(cmd *exec.Cmd) {
	signalProcess(cmd, syscall.SIGTERM)
}

// killProcess sends SIGKILL to the process group.
func killProcess(cmd *exec.Cmd) {
	signalProcess(cmd, syscall.SIGKILL)
}
