package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseComposeFile(t *testing.T) {
	t.Run("valid YAML", func(t *testing.T) {
		dir := t.TempDir()
		content := `
services:
  web:
    image: nginx
    ports:
      - "8080:80"
  db:
    image: postgres
    ports:
      - "5432:5432"
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(content), 0644))

		cfg, err := ParseComposeFile(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Len(t, cfg.Services, 2)
		assert.Contains(t, cfg.Services, "web")
		assert.Contains(t, cfg.Services, "db")
	})

	t.Run("tries all candidate filenames", func(t *testing.T) {
		candidates := []string{
			"docker-compose.yml",
			"docker-compose.yaml",
			"compose.yml",
			"compose.yaml",
		}

		for _, name := range candidates {
			t.Run(name, func(t *testing.T) {
				dir := t.TempDir()
				content := `
services:
  app:
    image: alpine
`
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))

				cfg, err := ParseComposeFile(dir)
				require.NoError(t, err)
				require.NotNil(t, cfg)
				assert.Len(t, cfg.Services, 1)
			})
		}
	})

	t.Run("no file found", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ParseComposeFile(dir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no compose file found")
	})

	t.Run("invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(":::invalid"), 0644))

		_, err := ParseComposeFile(dir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parsing compose file")
	})
}

func TestExtractPorts(t *testing.T) {
	t.Run("multiple services", func(t *testing.T) {
		cfg := &model.ComposeConfig{
			Services: map[string]model.ComposeService{
				"web": {Ports: []string{"8080:80", "443:443/tcp"}},
				"db":  {Ports: []string{"5432:5432"}},
			},
		}

		mappings := ExtractPorts(cfg)
		assert.Len(t, mappings, 3)
	})

	t.Run("empty services map", func(t *testing.T) {
		cfg := &model.ComposeConfig{
			Services: map[string]model.ComposeService{},
		}

		mappings := ExtractPorts(cfg)
		assert.Empty(t, mappings)
	})
}

func TestParsePortString(t *testing.T) {
	tests := []struct {
		name          string
		service       string
		raw           string
		wantHost      string
		wantContainer string
		wantProtocol  string
	}{
		{
			name:          "container only",
			service:       "web",
			raw:           "80",
			wantHost:      "",
			wantContainer: "80",
			wantProtocol:  "tcp",
		},
		{
			name:          "host:container",
			service:       "web",
			raw:           "8080:80",
			wantHost:      "8080",
			wantContainer: "80",
			wantProtocol:  "tcp",
		},
		{
			name:          "with protocol",
			service:       "web",
			raw:           "8080:80/udp",
			wantHost:      "8080",
			wantContainer: "80",
			wantProtocol:  "udp",
		},
		{
			name:          "IP:host:container",
			service:       "db",
			raw:           "127.0.0.1:8080:80",
			wantHost:      "8080",
			wantContainer: "80",
			wantProtocol:  "tcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePortString(tt.service, tt.raw)
			assert.Equal(t, tt.service, got.Service)
			assert.Equal(t, tt.wantHost, got.HostPort)
			assert.Equal(t, tt.wantContainer, got.ContainerPort)
			assert.Equal(t, tt.wantProtocol, got.Protocol)
		})
	}
}
