package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestHasComposeFile(t *testing.T) {
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	for _, name := range candidates {
		t.Run("present_"+name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, name), []byte("services: {}"), 0644)
			assert.True(t, HasComposeFile(dir))
		})
	}

	t.Run("absent", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, HasComposeFile(dir))
	})
}

func TestToDockerServices(t *testing.T) {
	t.Run("converts correctly", func(t *testing.T) {
		services := []Service{
			{
				Name: "web", Status: "running", State: "running",
				Publishers: []Publisher{{URL: "0.0.0.0", TargetPort: 80, PublishedPort: 8080, Protocol: "tcp"}},
			},
			{
				Name: "db", Status: "running", State: "running",
				Publishers: []Publisher{{URL: "0.0.0.0", TargetPort: 5432, PublishedPort: 5432, Protocol: "tcp"}},
			},
		}

		result := ToDockerServices(services)
		assert.Len(t, result, 2)
		assert.Equal(t, model.DockerService{
			Name:   "web",
			Status: "running",
			State:  "running",
			Ports:  "0.0.0.0:8080->80/tcp",
		}, result[0])
		assert.Equal(t, model.DockerService{
			Name:   "db",
			Status: "running",
			State:  "running",
			Ports:  "0.0.0.0:5432->5432/tcp",
		}, result[1])
	})

	t.Run("empty input", func(t *testing.T) {
		result := ToDockerServices([]Service{})
		assert.Empty(t, result)
	})

	t.Run("formats publishers with no host binding", func(t *testing.T) {
		services := []Service{
			{
				Name: "worker", Status: "running", State: "running",
				Publishers: []Publisher{{URL: "", TargetPort: 8000, PublishedPort: 0, Protocol: "tcp"}},
			},
		}
		result := ToDockerServices(services)
		assert.Equal(t, "8000/tcp", result[0].Ports)
	})

	t.Run("formats publishers with health", func(t *testing.T) {
		services := []Service{
			{
				Name: "pg", Status: "Up 5 minutes (healthy)", State: "running", Health: "healthy",
				Publishers: []Publisher{{URL: "0.0.0.0", TargetPort: 5432, PublishedPort: 3001, Protocol: "tcp"}},
			},
		}
		result := ToDockerServices(services)
		assert.Equal(t, "healthy", result[0].Health)
		assert.Equal(t, "0.0.0.0:3001->5432/tcp", result[0].Ports)
	})
}
