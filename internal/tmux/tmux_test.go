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
	cmd, args := SessionCommand("clawide-test", "/home/user/project")

	assert.Equal(t, "tmux", cmd)
	assert.Equal(t, []string{"new-session", "-A", "-s", "clawide-test", "-c", "/home/user/project"}, args)
}
