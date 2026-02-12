package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	home, _ := os.UserHomeDir()

	assert.Equal(t, "0.0.0.0", cfg.Host)
	assert.Equal(t, 9800, cfg.Port)
	assert.Equal(t, filepath.Join(home, "projects"), cfg.ProjectsDir)
	assert.Equal(t, 10, cfg.MaxSessions)
	assert.Equal(t, 65536, cfg.ScrollbackSize)
	assert.Equal(t, "claude", cfg.AgentCommand)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, filepath.Join(home, ".clawide"), cfg.DataDir)
	assert.False(t, cfg.Restart)
}

func TestAddr(t *testing.T) {
	cfg := &Config{Host: "127.0.0.1", Port: 8080}
	assert.Equal(t, "127.0.0.1:8080", cfg.Addr())
}

func TestStateFilePath(t *testing.T) {
	cfg := &Config{DataDir: "/tmp/clawide"}
	assert.Equal(t, "/tmp/clawide/state.json", cfg.StateFilePath())
}

func TestPidFilePath(t *testing.T) {
	cfg := &Config{DataDir: "/tmp/clawide"}
	assert.Equal(t, "/tmp/clawide/clawide.pid", cfg.PidFilePath())
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "tilde expansion",
			input: "~/projects",
			want:  filepath.Join(home, "projects"),
		},
		{
			name:  "absolute path passthrough",
			input: "/usr/local/bin",
			want:  "/usr/local/bin",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandHome(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadFile(t *testing.T) {
	t.Run("valid JSON config", func(t *testing.T) {
		dir := t.TempDir()
		cfgData := map[string]any{
			"host": "192.168.1.1",
			"port": 3000,
		}
		data, err := json.Marshal(cfgData)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), data, 0644))

		cfg := DefaultConfig()
		cfg.DataDir = dir
		err = cfg.loadFile()

		require.NoError(t, err)
		assert.Equal(t, "192.168.1.1", cfg.Host)
		assert.Equal(t, 3000, cfg.Port)
	})

	t.Run("file not found", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DataDir = "/nonexistent/path"
		err := cfg.loadFile()

		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("backward compat claude_command in config file", func(t *testing.T) {
		dir := t.TempDir()
		cfgData := map[string]any{
			"claude_command": "claude-beta",
		}
		data, err := json.Marshal(cfgData)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), data, 0644))

		cfg := DefaultConfig()
		cfg.DataDir = dir
		// Clear the default so we can test fallback
		cfg.AgentCommand = ""
		err = cfg.loadFile()

		require.NoError(t, err)
		assert.Equal(t, "claude-beta", cfg.AgentCommand)
	})

	t.Run("agent_command takes precedence over claude_command in config file", func(t *testing.T) {
		dir := t.TempDir()
		cfgData := map[string]any{
			"agent_command":  "aider",
			"claude_command": "claude-beta",
		}
		data, err := json.Marshal(cfgData)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), data, 0644))

		cfg := DefaultConfig()
		cfg.DataDir = dir
		err = cfg.loadFile()

		require.NoError(t, err)
		assert.Equal(t, "aider", cfg.AgentCommand)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte("{bad json"), 0644))

		cfg := DefaultConfig()
		cfg.DataDir = dir
		err := cfg.loadFile()

		assert.Error(t, err)
	})
}

func TestLoadEnv(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		field    func(*Config) any
		want     any
	}{
		{"host", "CLAWIDE_HOST", "10.0.0.1", func(c *Config) any { return c.Host }, "10.0.0.1"},
		{"port", "CLAWIDE_PORT", "4000", func(c *Config) any { return c.Port }, 4000},
		{"projects_dir", "CLAWIDE_PROJECTS_DIR", "/custom/projects", func(c *Config) any { return c.ProjectsDir }, "/custom/projects"},
		{"max_sessions", "CLAWIDE_MAX_SESSIONS", "20", func(c *Config) any { return c.MaxSessions }, 20},
		{"scrollback_size", "CLAWIDE_SCROLLBACK_SIZE", "131072", func(c *Config) any { return c.ScrollbackSize }, 131072},
		{"agent_command", "CLAWIDE_AGENT_COMMAND", "aider", func(c *Config) any { return c.AgentCommand }, "aider"},
		{"log_level", "CLAWIDE_LOG_LEVEL", "debug", func(c *Config) any { return c.LogLevel }, "debug"},
		{"data_dir", "CLAWIDE_DATA_DIR", "/custom/data", func(c *Config) any { return c.DataDir }, "/custom/data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			t.Setenv(tt.envKey, tt.envValue)
			cfg.loadEnv()
			assert.Equal(t, tt.want, tt.field(cfg))
		})
	}

	t.Run("backward compat CLAWIDE_CLAUDE_COMMAND", func(t *testing.T) {
		cfg := DefaultConfig()
		t.Setenv("CLAWIDE_CLAUDE_COMMAND", "claude-dev")
		cfg.loadEnv()
		assert.Equal(t, "claude-dev", cfg.AgentCommand)
	})

	t.Run("CLAWIDE_AGENT_COMMAND takes precedence over CLAWIDE_CLAUDE_COMMAND", func(t *testing.T) {
		cfg := DefaultConfig()
		t.Setenv("CLAWIDE_AGENT_COMMAND", "aider")
		t.Setenv("CLAWIDE_CLAUDE_COMMAND", "claude-dev")
		cfg.loadEnv()
		assert.Equal(t, "aider", cfg.AgentCommand)
	})

	t.Run("invalid int env vars ignored", func(t *testing.T) {
		cfg := DefaultConfig()
		t.Setenv("CLAWIDE_PORT", "not-a-number")
		t.Setenv("CLAWIDE_MAX_SESSIONS", "bad")
		cfg.loadEnv()

		assert.Equal(t, 9800, cfg.Port)
		assert.Equal(t, 10, cfg.MaxSessions)
	})
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAWIDE_DATA_DIR", dir)
	t.Setenv("CLAWIDE_HOST", "10.0.0.5")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "10.0.0.5", cfg.Host)
	assert.Equal(t, dir, cfg.DataDir)

	// Verify data dir was created
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
