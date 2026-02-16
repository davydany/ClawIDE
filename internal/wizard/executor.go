package wizard

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// CommandResult captures the outcome of an executed command.
type CommandResult struct {
	Command  string        `json:"command"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
	Err      error         `json:"-"`
}

// Executor runs shell commands with timeout support and structured results.
type Executor struct {
	defaultTimeout time.Duration
}

// NewExecutor creates a command executor with the given default timeout.
func NewExecutor(defaultTimeout time.Duration) *Executor {
	return &Executor{defaultTimeout: defaultTimeout}
}

// Run executes a command in the given directory with a context-based timeout.
// If timeout is 0, the executor's default timeout is used.
func (e *Executor) Run(ctx context.Context, dir, name string, args ...string) CommandResult {
	start := time.Now()

	// Apply timeout if the parent context doesn't already have one
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && e.defaultTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.defaultTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := CommandResult{
		Command:  fmt.Sprintf("%s %s", name, joinArgs(args)),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
	}

	if err != nil {
		result.Err = err
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}

	return result
}

// RunWithTimeout runs a command with a specific timeout override.
func (e *Executor) RunWithTimeout(ctx context.Context, timeout time.Duration, dir, name string, args ...string) CommandResult {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return e.Run(ctx, dir, name, args...)
}

// joinArgs creates a displayable string of command arguments.
func joinArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	result := args[0]
	for _, a := range args[1:] {
		result += " " + a
	}
	return result
}
