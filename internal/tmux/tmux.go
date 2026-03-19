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

// SessionCommand returns the command and args needed to attach-or-create a multiplexer session.
// The -A flag means: attach if exists, create if not.
func SessionCommand(name, workDir string) (string, []string) {
	return binary, []string{"new-session", "-A", "-s", name, "-c", workDir}
}

// SendKeys sends keystrokes to a multiplexer session. The keys are followed by Enter.
func SendKeys(sessionName, keys string) error {
	return exec.Command(binary, "send-keys", "-t", sessionName, keys, "Enter").Run()
}

// TmuxName returns the tmux session name for a given pane ID.
func TmuxName(paneID string) string {
	return prefix + paneID
}
