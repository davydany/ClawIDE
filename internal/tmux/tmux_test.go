package tmux

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTmuxName(t *testing.T) {
	assert.Equal(t, "clawide-abc-123", TmuxName("abc-123"))
	assert.Equal(t, "clawide-", TmuxName(""))
	assert.Equal(t, "clawide-my-pane", TmuxName("my-pane"))
}

func TestSessionCommand(t *testing.T) {
	// Save and restore original binary
	orig := Binary()
	defer SetBinary(orig)

	cmd, args := SessionCommand("clawide-test", "/home/user/project")

	assert.Equal(t, orig, cmd)
	assert.Equal(t, []string{"new-session", "-A", "-s", "clawide-test", "-c", "/home/user/project"}, args)
}

func TestSetBinaryAndBinary(t *testing.T) {
	orig := Binary()
	defer SetBinary(orig)

	assert.Equal(t, "tmux", Binary())

	SetBinary("psmux")
	assert.Equal(t, "psmux", Binary())

	SetBinary("tmux")
	assert.Equal(t, "tmux", Binary())
}

func TestSessionCommandWithCustomBinary(t *testing.T) {
	orig := Binary()
	defer SetBinary(orig)

	SetBinary("psmux")
	cmd, args := SessionCommand("clawide-test", "/home/user/project")

	assert.Equal(t, "psmux", cmd)
	assert.Equal(t, []string{"new-session", "-A", "-s", "clawide-test", "-c", "/home/user/project"}, args)
}

func TestCheckWithNonexistentBinary(t *testing.T) {
	orig := Binary()
	defer SetBinary(orig)

	SetBinary("nonexistent-mux-binary-12345")
	err := Check()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-mux-binary-12345")
	assert.Contains(t, err.Error(), "not found")
}
