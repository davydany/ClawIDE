package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	ProjectsDir    string `json:"projects_dir"`
	MaxSessions    int    `json:"max_sessions"`
	ScrollbackSize int    `json:"scrollback_size"`
	ClaudeCommand  string `json:"claude_command"`
	LogLevel       string `json:"log_level"`
	DataDir                string `json:"data_dir"`
	OnboardingCompleted    bool   `json:"onboarding_completed"`
	WorkspaceTourCompleted bool   `json:"workspace_tour_completed"`
	Restart                bool   `json:"-"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Host:           "0.0.0.0",
		Port:           9800,
		ProjectsDir:    filepath.Join(home, "projects"),
		MaxSessions:    10,
		ScrollbackSize: 65536,
		ClaudeCommand:  "claude",
		LogLevel:       "info",
		DataDir:        filepath.Join(home, ".clawide"),
	}
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	if err := cfg.loadFile(); err != nil {
		// Config file is optional; only error if it exists but is invalid
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading config file: %w", err)
		}
	}

	cfg.loadEnv()
	cfg.loadFlags()

	// Expand ~ in paths
	cfg.ProjectsDir = expandHome(cfg.ProjectsDir)
	cfg.DataDir = expandHome(cfg.DataDir)

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	return cfg, nil
}

func (c *Config) loadFile() error {
	path := filepath.Join(c.DataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, c)
}

func (c *Config) loadEnv() {
	if v := os.Getenv("CLAWIDE_HOST"); v != "" {
		c.Host = v
	}
	if v := os.Getenv("CLAWIDE_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Port = p
		}
	}
	if v := os.Getenv("CLAWIDE_PROJECTS_DIR"); v != "" {
		c.ProjectsDir = v
	}
	if v := os.Getenv("CLAWIDE_MAX_SESSIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxSessions = n
		}
	}
	if v := os.Getenv("CLAWIDE_SCROLLBACK_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.ScrollbackSize = n
		}
	}
	if v := os.Getenv("CLAWIDE_CLAUDE_COMMAND"); v != "" {
		c.ClaudeCommand = v
	}
	if v := os.Getenv("CLAWIDE_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("CLAWIDE_DATA_DIR"); v != "" {
		c.DataDir = v
	}
}

func (c *Config) loadFlags() {
	flag.StringVar(&c.Host, "host", c.Host, "Listen host")
	flag.IntVar(&c.Port, "port", c.Port, "Listen port")
	flag.StringVar(&c.ProjectsDir, "projects-dir", c.ProjectsDir, "Projects root directory")
	flag.IntVar(&c.MaxSessions, "max-sessions", c.MaxSessions, "Maximum concurrent sessions")
	flag.StringVar(&c.ClaudeCommand, "claude-command", c.ClaudeCommand, "Claude CLI command")
	flag.StringVar(&c.LogLevel, "log-level", c.LogLevel, "Log level (debug, info, warn, error)")
	flag.StringVar(&c.DataDir, "data-dir", c.DataDir, "Data directory for state/config")
	flag.BoolVar(&c.Restart, "restart", false, "Kill existing instance and restart")
	flag.Parse()
}

func (c *Config) StateFilePath() string {
	return filepath.Join(c.DataDir, "state.json")
}

func (c *Config) PidFilePath() string {
	return filepath.Join(c.DataDir, "clawide.pid")
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
