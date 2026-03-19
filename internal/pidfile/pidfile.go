package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Path returns the PID file path within the given data directory.
func Path(dataDir string) string {
	return filepath.Join(dataDir, "clawide.pid")
}

// Read reads the PID from the given file path.
func Read(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid pid file content: %w", err)
	}
	return pid, nil
}

// Write writes the current process PID to the given file path.
func Write(path string) error {
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0644)
}

// Remove deletes the PID file at the given path.
func Remove(path string) {
	os.Remove(path)
}
