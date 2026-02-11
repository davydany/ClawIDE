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
			{Name: "web", Status: "running", State: "running", Ports: "0.0.0.0:8080->80/tcp"},
			{Name: "db", Status: "running", State: "running", Ports: "0.0.0.0:5432->5432/tcp"},
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
}
