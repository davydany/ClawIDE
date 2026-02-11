package pidfile

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPath(t *testing.T) {
	got := Path("/var/data")
	assert.Equal(t, "/var/data/clawide.pid", got)
}

func TestRead(t *testing.T) {
	t.Run("valid PID", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.pid")
		require.NoError(t, os.WriteFile(path, []byte("12345\n"), 0644))

		pid, err := Read(path)
		require.NoError(t, err)
		assert.Equal(t, 12345, pid)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := Read("/nonexistent/test.pid")
		assert.Error(t, err)
	})

	t.Run("invalid content", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.pid")
		require.NoError(t, os.WriteFile(path, []byte("not-a-pid"), 0644))

		_, err := Read(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid pid file content")
	})
}

func TestWriteAndReadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pid")

	err := Write(path)
	require.NoError(t, err)

	pid, err := Read(path)
	require.NoError(t, err)
	assert.Equal(t, os.Getpid(), pid)
}

func TestRemove(t *testing.T) {
	t.Run("file removed", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.pid")
		require.NoError(t, os.WriteFile(path, []byte("12345"), 0644))

		Remove(path)
		_, err := os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("no error on missing file", func(t *testing.T) {
		// Remove should not panic on non-existent file
		Remove("/nonexistent/test.pid")
	})
}

func TestIsRunning(t *testing.T) {
	t.Run("current process is running", func(t *testing.T) {
		assert.True(t, IsRunning(os.Getpid()))
	})

	t.Run("non-existent PID", func(t *testing.T) {
		// Use a very high PID that is unlikely to exist
		assert.False(t, IsRunning(99999999))
	})
}

func TestWriteContainsCurrentPID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pid")

	require.NoError(t, Write(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, strconv.Itoa(os.Getpid()), string(data))
}
