// Package cron manages system crontab entries for ClawIDE scheduled jobs.
// Each managed entry is tagged with a comment marker so ClawIDE can identify
// and remove its own entries without touching user-defined cron jobs.
package cron

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

const marker = "clawide-job:"

// IsSupported returns true if the system has a crontab command available.
// Cron is supported on macOS and Linux but not Windows.
func IsSupported() bool {
	if runtime.GOOS == "windows" {
		return false
	}
	_, err := exec.LookPath("crontab")
	return err == nil
}

// Install adds a crontab entry for the given job ID. The entry is tagged with
// a comment marker so it can be identified and removed later. If an entry for
// this job ID already exists, it is replaced.
func Install(jobID, cronExpr, command string) error {
	lines, err := readCrontab()
	if err != nil {
		return fmt.Errorf("reading crontab: %w", err)
	}

	tag := marker + jobID
	entry := cronExpr + " " + command + " # " + tag

	// Remove any existing entry for this job ID
	filtered := removeTagged(lines, tag)
	filtered = append(filtered, entry)

	return writeCrontab(filtered)
}

// Remove deletes the crontab entry for the given job ID.
func Remove(jobID string) error {
	lines, err := readCrontab()
	if err != nil {
		return fmt.Errorf("reading crontab: %w", err)
	}

	tag := marker + jobID
	filtered := removeTagged(lines, tag)

	// Only write if something actually changed
	if len(filtered) == len(lines) {
		return nil
	}
	return writeCrontab(filtered)
}

// HasEntry checks whether a crontab entry exists for the given job ID.
func HasEntry(jobID string) bool {
	lines, err := readCrontab()
	if err != nil {
		return false
	}
	tag := marker + jobID
	for _, line := range lines {
		if strings.Contains(line, tag) {
			return true
		}
	}
	return false
}

// BuildCommand constructs the shell command string for a cron entry.
// It runs the agent CLI in non-interactive mode from the project directory.
func BuildCommand(agent, prompt, projectPath, logPath string) string {
	escapedPrompt := shellEscape(prompt)
	agentCmd := agentPrintCommand(agent)

	cmd := fmt.Sprintf("cd %s && %s %s", shellEscape(projectPath), agentCmd, escapedPrompt)
	if logPath != "" {
		cmd += fmt.Sprintf(" >> %s 2>&1", shellEscape(logPath))
	}
	return cmd
}

// agentPrintCommand returns the non-interactive CLI invocation prefix for
// each supported agent.
func agentPrintCommand(agent string) string {
	switch agent {
	case "codex":
		return "codex exec"
	case "gemini":
		return "gemini"
	default: // "claude" and any unknown agent
		return "claude -p"
	}
}

func shellEscape(s string) string {
	// Use single quotes with inner single-quote escaping: replace ' with '\''
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func readCrontab() ([]string, error) {
	out, err := exec.Command("crontab", "-l").CombinedOutput()
	if err != nil {
		// "no crontab for <user>" is not a real error — just empty
		if strings.Contains(string(out), "no crontab") {
			return nil, nil
		}
		return nil, fmt.Errorf("crontab -l: %s", string(out))
	}
	raw := strings.TrimRight(string(out), "\n")
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

func writeCrontab(lines []string) error {
	content := strings.Join(lines, "\n") + "\n"
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("crontab -: %s (%w)", string(out), err)
	}
	return nil
}

func removeTagged(lines []string, tag string) []string {
	var filtered []string
	for _, line := range lines {
		if !strings.Contains(line, tag) {
			filtered = append(filtered, line)
		}
	}
	return filtered
}
