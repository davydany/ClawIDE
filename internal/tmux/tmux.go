package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

const prefix = "clawide-"

// Check verifies that tmux is installed and accessible.
func Check() error {
	out, err := exec.Command("tmux", "-V").CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux not found (is it installed?): %w\nOutput: %s", err, string(out))
	}
	return nil
}

// HasSession checks whether a tmux session with the given name exists.
func HasSession(name string) bool {
	err := exec.Command("tmux", "has-session", "-t", name).Run()
	return err == nil
}

// ListClawIDESessions returns the names of all tmux sessions that start with the clawide- prefix.
func ListClawIDESessions() ([]string, error) {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		// tmux returns error when no server is running (no sessions) — that's fine
		if strings.Contains(err.Error(), "exit status") {
			return nil, nil
		}
		return nil, fmt.Errorf("listing tmux sessions: %w", err)
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

// KillSession kills a tmux session by name.
func KillSession(name string) error {
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

// SessionCommand returns the command and args needed to attach-or-create a tmux session.
// The -A flag means: attach if exists, create if not.
func SessionCommand(name, workDir string) (string, []string) {
	return "tmux", []string{"new-session", "-A", "-s", name, "-c", workDir}
}

// SendKeys sends keystrokes to a tmux session. The keys are followed by Enter.
func SendKeys(sessionName, keys string) error {
	return exec.Command("tmux", "send-keys", "-t", sessionName, keys, "Enter").Run()
}

// TmuxName returns the tmux session name for a given pane ID.
func TmuxName(paneID string) string {
	return prefix + paneID
}
