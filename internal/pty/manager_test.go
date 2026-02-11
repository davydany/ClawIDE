package pty

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	m := NewManager(10, 65536, "claude")

	assert.NotNil(t, m.sessions)
	assert.Equal(t, 10, m.maxSessions)
	assert.Equal(t, 65536, m.scrollbackSize)
	assert.Equal(t, "claude", m.claudeCommand)
}

func TestSessionCount(t *testing.T) {
	m := NewManager(10, 65536, "claude")
	assert.Equal(t, 0, m.SessionCount())
}

func TestGetSessionNotFound(t *testing.T) {
	m := NewManager(10, 65536, "claude")
	sess, ok := m.GetSession("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, sess)
}
