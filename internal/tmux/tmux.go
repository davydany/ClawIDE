package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

const prefix = "clawide-"

// binary is the terminal multiplexer executable name. Defaults to "tmux".
// Use SetBinary() to override (e.g., "psmux" on Windows).
var binary = "tmux"

// SetBinary sets the multiplexer binary name (e.g., "tmux" or "psmux").
func SetBinary(name string) {
	binary = name
}

// Binary returns the current multiplexer binary name.
func Binary() string {
	return binary
}

// Check verifies that the multiplexer binary is installed and accessible.
func Check() error {
	path, err := exec.LookPath(binary)
	if err != nil {
		return fmt.Errorf("%s not found in PATH (is it installed?): %w", binary, err)
	}

	out, cmdErr := exec.Command(path, "-V").CombinedOutput()
	if cmdErr != nil {
		return fmt.Errorf("%s found at %s but version check failed: %w\nOutput: %s", binary, path, cmdErr, string(out))
	}
	return nil
}

// HasSession checks whether a multiplexer session with the given name exists.
func HasSession(name string) bool {
	err := exec.Command(binary, "has-session", "-t", name).Run()
	return err == nil
}

// ListClawIDESessions returns the names of all multiplexer sessions that start with the clawide- prefix.
func ListClawIDESessions() ([]string, error) {
	out, err := exec.Command(binary, "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		// multiplexer returns error when no server is running (no sessions) — that's fine
		if strings.Contains(err.Error(), "exit status") {
			return nil, nil
		}
		return nil, fmt.Errorf("listing %s sessions: %w", binary, err)
	}

	var sessions []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasPrefix(line, prefix) {
			sessions = append(sessions, line)
		}
	}
	return sessions, nil
}

// KillSession kills a multiplexer session by name.
func KillSession(name string) error {
	return exec.Command(binary, "kill-session", "-t", name).Run()
}

// SessionCommand returns the command and args needed to attach an existing
// multiplexer session. Callers must ensure the session exists first via
// PrepareSession — this is deliberately split from creation so per-session
// options (mouse, history-limit, window-size) can be applied before attach.
func SessionCommand(name string) (string, []string) {
	return binary, []string{"attach-session", "-t", name}
}

// PrepareSession ensures the named session exists and applies ClawIDE's
// per-session option overrides. It is idempotent: safe to call on every
// attach, including for sessions created by an older ClawIDE build.
//
// The options applied here fix mouse-wheel scrolling inside tmux panes:
//   - mouse off: stops tmux from capturing wheel events so xterm.js can
//     scroll its own buffer. Overrides the user's global `set -g mouse on`
//     for this session only.
//   - history-limit 50000: deeper scrollback for the Ctrl+B [ fallback.
//   - focus-events on: Claude Code reads focus in/out to pause/resume.
//   - window-size latest: session adopts the attaching client's dimensions
//     immediately, avoiding a brief paint at tmux's default 80x24.
//
// Option errors are swallowed — a failed set-option shouldn't block attach.
func PrepareSession(name, workDir string) error {
	if !HasSession(name) {
		if err := exec.Command(binary, "new-session", "-d", "-s", name, "-c", workDir).Run(); err != nil {
			return fmt.Errorf("creating detached %s session: %w", binary, err)
		}
	}
	opts := [][]string{
		{"set-option", "-t", name, "mouse", "off"},
		{"set-option", "-t", name, "history-limit", "50000"},
		{"set-option", "-t", name, "focus-events", "on"},
		{"set-option", "-t", name, "window-size", "latest"},
	}
	for _, args := range opts {
		_ = exec.Command(binary, args...).Run()
	}
	return nil
}

// SendKeys sends keystrokes to a multiplexer session. The keys are followed by Enter.
func SendKeys(sessionName, keys string) error {
	return exec.Command(binary, "send-keys", "-t", sessionName, keys, "Enter").Run()
}

// TmuxName returns the tmux session name for a given pane ID.
func TmuxName(paneID string) string {
	return prefix + paneID
}

// GetPasteBuffer returns the contents of the most recent tmux paste buffer.
func GetPasteBuffer() (string, error) {
	out, err := exec.Command(binary, "show-buffer").Output()
	if err != nil {
		return "", fmt.Errorf("tmux show-buffer: %w", err)
	}
	return string(out), nil
}
