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

func TestNormalizeBuild(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, ""},
		{"string", ".", "."},
		{"map with context only", map[string]any{"context": "./app"}, "./app"},
		{"map with context and dockerfile", map[string]any{"context": "./app", "dockerfile": "Dockerfile.prod"}, "./app/Dockerfile.prod"},
		{"unsupported type", 42, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeBuild(tt.in))
		})
	}
}

func TestNormalizeEnvironment(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want []string
	}{
		{"nil", nil, nil},
		{"string list", []any{"FOO=bar", "BAZ=qux"}, []string{"BAZ=qux", "FOO=bar"}},
		{"key-value map", map[string]any{"DB_HOST": "localhost", "DB_PORT": 5432}, []string{"DB_HOST=localhost", "DB_PORT=5432"}},
		{"unsupported type", 42, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeEnvironment(tt.in))
		})
	}
}

func TestNormalizeDependsOn(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want []string
	}{
		{"nil", nil, nil},
		{"string list", []any{"db", "redis"}, []string{"db", "redis"}},
		{"map with conditions", map[string]any{"db": map[string]any{"condition": "service_healthy"}, "redis": nil}, []string{"db", "redis"}},
		{"unsupported type", 42, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeDependsOn(tt.in))
		})
	}
}

func TestNormalizeCommand(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, ""},
		{"string", "python manage.py runserver", "python manage.py runserver"},
		{"string list", []any{"python", "manage.py", "runserver"}, "python manage.py runserver"},
		{"unsupported type", 42, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeCommand(tt.in))
		})
	}
}

func TestExtractServiceDetails(t *testing.T) {
	t.Run("full config", func(t *testing.T) {
		cfg := &model.ComposeConfig{
			Services: map[string]model.ComposeService{
				"web": {
					Image:         "nginx:latest",
					Ports:         []string{"8080:80", "443:443/tcp"},
					Volumes:       []string{"./html:/usr/share/nginx/html"},
					Environment:   []any{"NGINX_HOST=localhost"},
					DependsOn:     []any{"api"},
					Command:       "nginx -g 'daemon off;'",
					ContainerName: "my-web",
					Restart:       "always",
				},
				"api": {
					Build:       map[string]any{"context": ".", "dockerfile": "Dockerfile"},
					Ports:       []string{"3000:3000"},
					Environment: map[string]any{"DB_HOST": "db", "DB_PORT": 5432},
				},
			},
		}

		details := ExtractServiceDetails(cfg)
		require.Len(t, details, 2)

		// Sorted by name: api first, then web
		assert.Equal(t, "api", details[0].Name)
		assert.Equal(t, "./Dockerfile", details[0].Build)
		assert.Len(t, details[0].Ports, 1)
		assert.Equal(t, "3000", details[0].Ports[0].HostPort)
		assert.Equal(t, []string{"DB_HOST=db", "DB_PORT=5432"}, details[0].Environment)

		assert.Equal(t, "web", details[1].Name)
		assert.Equal(t, "nginx:latest", details[1].Image)
		assert.Len(t, details[1].Ports, 2)
		assert.Equal(t, "my-web", details[1].ContainerName)
		assert.Equal(t, "always", details[1].Restart)
		assert.Equal(t, "nginx -g 'daemon off;'", details[1].Command)
		assert.Equal(t, []string{"api"}, details[1].DependsOn)
	})

	t.Run("nil config", func(t *testing.T) {
		assert.Nil(t, ExtractServiceDetails(nil))
	})

	t.Run("empty services", func(t *testing.T) {
		cfg := &model.ComposeConfig{Services: map[string]model.ComposeService{}}
		details := ExtractServiceDetails(cfg)
		assert.Empty(t, details)
	})

	t.Run("partial fields produce empty slices not nil", func(t *testing.T) {
		cfg := &model.ComposeConfig{
			Services: map[string]model.ComposeService{
				"minimal": {Image: "alpine"},
			},
		}
		details := ExtractServiceDetails(cfg)
		require.Len(t, details, 1)
		assert.Equal(t, "minimal", details[0].Name)
		assert.NotNil(t, details[0].Ports)
		assert.NotNil(t, details[0].Volumes)
		assert.NotNil(t, details[0].Environment)
		assert.NotNil(t, details[0].DependsOn)
		assert.Empty(t, details[0].Ports)
		assert.Empty(t, details[0].Volumes)
		assert.Empty(t, details[0].Environment)
		assert.Empty(t, details[0].DependsOn)
	})
}
