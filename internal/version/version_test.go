package version

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaults(t *testing.T) {
	info := Get()
	assert.Equal(t, "dev", info.Version)
	assert.Equal(t, "none", info.Commit)
	assert.Equal(t, "unknown", info.Date)
}

func TestString(t *testing.T) {
	s := String()
	assert.True(t, strings.HasPrefix(s, "ClawIDE "))
	assert.Contains(t, s, "dev")
	assert.Contains(t, s, "none")
	assert.Contains(t, s, "unknown")
}

func TestGetWithInjectedValues(t *testing.T) {
	// Save originals
	origVersion, origCommit, origDate := Version, Commit, Date
	defer func() {
		Version, Commit, Date = origVersion, origCommit, origDate
	}()

	Version = "v1.2.3"
	Commit = "abc1234"
	Date = "2025-01-15T10:00:00Z"

	info := Get()
	assert.Equal(t, "v1.2.3", info.Version)
	assert.Equal(t, "abc1234", info.Commit)
	assert.Equal(t, "2025-01-15T10:00:00Z", info.Date)

	s := String()
	assert.Contains(t, s, "v1.2.3")
	assert.Contains(t, s, "abc1234")
	assert.Contains(t, s, "2025-01-15T10:00:00Z")
}
