package cron

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCommand(t *testing.T) {
	t.Run("claude agent", func(t *testing.T) {
		cmd := BuildCommand("claude", "check CI status", "/home/user/myproject", "/tmp/log.log")
		assert.Contains(t, cmd, "claude -p")
		assert.Contains(t, cmd, "check CI status")
		assert.Contains(t, cmd, "cd '/home/user/myproject'")
		assert.Contains(t, cmd, ">> '/tmp/log.log' 2>&1")
	})

	t.Run("codex agent", func(t *testing.T) {
		cmd := BuildCommand("codex", "fix tests", "/home/user/proj", "")
		assert.Contains(t, cmd, "codex exec")
		assert.Contains(t, cmd, "fix tests")
		assert.NotContains(t, cmd, ">>")
	})

	t.Run("gemini agent", func(t *testing.T) {
		cmd := BuildCommand("gemini", "review code", "/proj", "/log.log")
		assert.Contains(t, cmd, "gemini")
		assert.Contains(t, cmd, "review code")
	})

	t.Run("shell escaping", func(t *testing.T) {
		cmd := BuildCommand("claude", "it's a test", "/path/with spaces", "")
		assert.Contains(t, cmd, "'it'\\''s a test'")
		assert.Contains(t, cmd, "'/path/with spaces'")
	})
}

func TestShellEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "'simple'"},
		{"with spaces", "'with spaces'"},
		{"it's", "'it'\\''s'"},
		{"", "''"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, shellEscape(tt.input))
		})
	}
}

func TestRemoveTagged(t *testing.T) {
	lines := []string{
		"0 * * * * /usr/bin/something # user cron",
		"*/5 * * * * cd /proj && claude -p 'test' # clawide-job:abc-123",
		"30 2 * * * /usr/bin/backup # nightly",
	}
	result := removeTagged(lines, "clawide-job:abc-123")
	assert.Len(t, result, 2)
	assert.Equal(t, "0 * * * * /usr/bin/something # user cron", result[0])
	assert.Equal(t, "30 2 * * * /usr/bin/backup # nightly", result[1])
}
