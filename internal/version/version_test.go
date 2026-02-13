package version

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestIsDevVersion(t *testing.T) {
	orig := Version
	defer func() { Version = orig }()

	Version = "dev"
	require.True(t, IsDevVersion())

	Version = "v1.0.0"
	require.False(t, IsDevVersion())

	Version = "v0.1.0"
	require.False(t, IsDevVersion())
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v1.0.0", "v1.0.0", 0},
		{"v1.0.0", "v1.0.1", -1},
		{"v1.0.1", "v1.0.0", 1},
		{"v1.1.0", "v1.0.0", 1},
		{"v1.0.0", "v1.1.0", -1},
		{"v2.0.0", "v1.9.9", 1},
		{"v1.9.9", "v2.0.0", -1},
		// Without v prefix
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		// Mixed prefix
		{"v1.0.0", "1.0.0", 0},
		// Pre-release suffix stripped
		{"v1.0.0-rc1", "v1.0.0", 0},
		// Partial versions
		{"v1.0", "v1.0.0", 0},
		{"v1", "v1.0.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got := CompareVersions(tt.a, tt.b)
			assert.Equal(t, tt.want, got, "CompareVersions(%q, %q)", tt.a, tt.b)
		})
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"v1.2.3", [3]int{1, 2, 3}},
		{"1.2.3", [3]int{1, 2, 3}},
		{"v0.0.0", [3]int{0, 0, 0}},
		{"v10.20.30", [3]int{10, 20, 30}},
		{"v1.2.3-rc1", [3]int{1, 2, 3}},
		{"v1", [3]int{1, 0, 0}},
		{"v1.2", [3]int{1, 2, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseSemver(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
