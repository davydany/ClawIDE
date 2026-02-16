package wizard

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewExecutor
// ---------------------------------------------------------------------------

func TestNewExecutor(t *testing.T) {
	e := NewExecutor(30 * time.Second)
	require.NotNil(t, e)
	assert.Equal(t, 30*time.Second, e.defaultTimeout)
}

func TestNewExecutor_ZeroTimeout(t *testing.T) {
	e := NewExecutor(0)
	require.NotNil(t, e)
	assert.Equal(t, time.Duration(0), e.defaultTimeout)
}

// ---------------------------------------------------------------------------
// Run - successful commands
// ---------------------------------------------------------------------------

func TestExecutor_Run_SuccessfulCommand(t *testing.T) {
	e := NewExecutor(10 * time.Second)
	ctx := context.Background()

	result := e.Run(ctx, t.TempDir(), "echo", "hello world")

	assert.Nil(t, result.Err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "hello world")
	assert.Empty(t, result.Stderr)
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.Contains(t, result.Command, "echo")
}

func TestExecutor_Run_CapturesStdout(t *testing.T) {
	e := NewExecutor(10 * time.Second)
	ctx := context.Background()

	result := e.Run(ctx, t.TempDir(), "printf", "line1\nline2\n")

	assert.Nil(t, result.Err)
	assert.Equal(t, "line1\nline2\n", result.Stdout)
}

func TestExecutor_Run_CapturesStderr(t *testing.T) {
	e := NewExecutor(10 * time.Second)
	ctx := context.Background()

	result := e.Run(ctx, t.TempDir(), "sh", "-c", "echo error >&2")

	assert.Nil(t, result.Err)
	assert.Contains(t, result.Stderr, "error")
}

func TestExecutor_Run_WorkingDirectory(t *testing.T) {
	e := NewExecutor(10 * time.Second)
	ctx := context.Background()
	dir := t.TempDir()

	result := e.Run(ctx, dir, "pwd")

	assert.Nil(t, result.Err)
	assert.Contains(t, result.Stdout, dir)
}

// ---------------------------------------------------------------------------
// Run - failed commands
// ---------------------------------------------------------------------------

func TestExecutor_Run_NonZeroExitCode(t *testing.T) {
	e := NewExecutor(10 * time.Second)
	ctx := context.Background()

	result := e.Run(ctx, t.TempDir(), "sh", "-c", "exit 42")

	assert.NotNil(t, result.Err)
	assert.Equal(t, 42, result.ExitCode)
}

func TestExecutor_Run_CommandNotFound(t *testing.T) {
	e := NewExecutor(10 * time.Second)
	ctx := context.Background()

	result := e.Run(ctx, t.TempDir(), "nonexistent-command-xyz")

	assert.NotNil(t, result.Err)
	assert.Equal(t, -1, result.ExitCode, "non-exec errors should return exit code -1")
}

func TestExecutor_Run_ExitCodeOne(t *testing.T) {
	e := NewExecutor(10 * time.Second)
	ctx := context.Background()

	result := e.Run(ctx, t.TempDir(), "false")

	assert.NotNil(t, result.Err)
	assert.Equal(t, 1, result.ExitCode)
}

// ---------------------------------------------------------------------------
// Run - timeout handling
// ---------------------------------------------------------------------------

func TestExecutor_Run_DefaultTimeoutApplied(t *testing.T) {
	// 100ms timeout should kill a 10s sleep
	e := NewExecutor(100 * time.Millisecond)
	ctx := context.Background()

	result := e.Run(ctx, t.TempDir(), "sleep", "10")

	assert.NotNil(t, result.Err)
	assert.Less(t, result.Duration, 2*time.Second)
}

func TestExecutor_Run_ParentContextDeadlineUsed(t *testing.T) {
	// If parent context already has a deadline, executor should not override it
	e := NewExecutor(10 * time.Second) // long default
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := e.Run(ctx, t.TempDir(), "sleep", "10")

	assert.NotNil(t, result.Err)
	assert.Less(t, result.Duration, 2*time.Second)
}

// ---------------------------------------------------------------------------
// RunWithTimeout
// ---------------------------------------------------------------------------

func TestExecutor_RunWithTimeout_OverridesDefault(t *testing.T) {
	e := NewExecutor(30 * time.Second) // long default
	ctx := context.Background()

	result := e.RunWithTimeout(ctx, 100*time.Millisecond, t.TempDir(), "sleep", "10")

	assert.NotNil(t, result.Err)
	assert.Less(t, result.Duration, 2*time.Second)
}

func TestExecutor_RunWithTimeout_SuccessfulWithinTimeout(t *testing.T) {
	e := NewExecutor(30 * time.Second)
	ctx := context.Background()

	result := e.RunWithTimeout(ctx, 5*time.Second, t.TempDir(), "echo", "fast")

	assert.Nil(t, result.Err)
	assert.Contains(t, result.Stdout, "fast")
}

// ---------------------------------------------------------------------------
// Command result formatting
// ---------------------------------------------------------------------------

func TestExecutor_Run_CommandStringFormatted(t *testing.T) {
	e := NewExecutor(10 * time.Second)
	ctx := context.Background()

	result := e.Run(ctx, t.TempDir(), "echo", "-n", "hello")

	assert.Equal(t, "echo -n hello", result.Command)
}

func TestExecutor_Run_NoArgs(t *testing.T) {
	e := NewExecutor(10 * time.Second)
	ctx := context.Background()

	result := e.Run(ctx, t.TempDir(), "true")

	assert.Equal(t, "true ", result.Command)
	assert.Nil(t, result.Err)
}

// ---------------------------------------------------------------------------
// joinArgs helper
// ---------------------------------------------------------------------------

func TestJoinArgs_Empty(t *testing.T) {
	assert.Equal(t, "", joinArgs(nil))
	assert.Equal(t, "", joinArgs([]string{}))
}

func TestJoinArgs_Single(t *testing.T) {
	assert.Equal(t, "hello", joinArgs([]string{"hello"}))
}

func TestJoinArgs_Multiple(t *testing.T) {
	assert.Equal(t, "-n hello world", joinArgs([]string{"-n", "hello", "world"}))
}
