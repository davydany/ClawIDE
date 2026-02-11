package docker

import (
	"bytes"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/davydany/ClawIDE/internal/model"
)

// Service represents a running Docker Compose service as reported by `docker compose ps`.
type Service struct {
	Name   string `json:"Name"`
	Status string `json:"Status"`
	State  string `json:"State"`
	Ports  string `json:"Publishers"`
}

// PS runs `docker compose ps --format json` in the given project directory
// and returns the list of services.
func PS(projectPath string) ([]Service, error) {
	cmd := exec.Command("docker", "compose", "ps", "--format", "json")
	cmd.Dir = projectPath

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker compose ps: %w", err)
	}

	if len(out) == 0 {
		return []Service{}, nil
	}

	// docker compose ps --format json outputs one JSON object per line (NDJSON),
	// not a JSON array. Parse each line independently.
	var services []Service
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Try parsing as a single object first.
		var svc Service
		if err := json.Unmarshal(line, &svc); err == nil {
			services = append(services, svc)
			continue
		}

		// Some versions emit a JSON array on a single line.
		var arr []Service
		if err := json.Unmarshal(line, &arr); err == nil {
			services = append(services, arr...)
			continue
		}
	}

	return services, scanner.Err()
}

// Up runs `docker compose up -d` in the given project directory.
func Up(projectPath string) error {
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}
	return nil
}

// Down runs `docker compose down` in the given project directory.
func Down(projectPath string) error {
	cmd := exec.Command("docker", "compose", "down")
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose down: %w", err)
	}
	return nil
}

// StartService starts a single service via `docker compose start <service>`.
func StartService(projectPath, service string) error {
	cmd := exec.Command("docker", "compose", "start", service)
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose start %s: %w", service, err)
	}
	return nil
}

// StopService stops a single service via `docker compose stop <service>`.
func StopService(projectPath, service string) error {
	cmd := exec.Command("docker", "compose", "stop", service)
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose stop %s: %w", service, err)
	}
	return nil
}

// RestartService restarts a single service via `docker compose restart <service>`.
func RestartService(projectPath, service string) error {
	cmd := exec.Command("docker", "compose", "restart", service)
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose restart %s: %w", service, err)
	}
	return nil
}

// LogsStream starts `docker compose logs -f <service>` and returns a ReadCloser
// that streams the combined stdout/stderr output. The caller is responsible for
// closing the returned reader, which will also terminate the underlying process.
// The provided context can be used to cancel the log stream.
func LogsStream(ctx context.Context, projectPath, service string) (io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, "docker", "compose", "logs", "-f", "--no-log-prefix", service)
	cmd.Dir = projectPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("docker compose logs stdout pipe: %w", err)
	}

	// Merge stderr into stdout so we get all output from a single reader.
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("docker compose logs start: %w", err)
	}

	// Wrap in a processReadCloser so that closing the reader also cleans up
	// the child process.
	return &processReadCloser{
		ReadCloser: stdout,
		cmd:        cmd,
	}, nil
}

// HasComposeFile checks whether the project directory contains a
// docker-compose.yml, docker-compose.yaml, compose.yml, or compose.yaml file.
func HasComposeFile(projectPath string) bool {
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}
	for _, name := range candidates {
		if _, err := os.Stat(filepath.Join(projectPath, name)); err == nil {
			return true
		}
	}
	return false
}

// ToDockerServices converts the internal Service slice to the model representation
// used by handlers and templates.
func ToDockerServices(services []Service) []model.DockerService {
	out := make([]model.DockerService, len(services))
	for i, s := range services {
		out[i] = model.DockerService{
			Name:   s.Name,
			Status: s.Status,
			State:  s.State,
			Ports:  s.Ports,
		}
	}
	return out
}

// processReadCloser wraps an io.ReadCloser and ensures the underlying
// exec.Cmd process is properly waited on when closed.
type processReadCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (p *processReadCloser) Close() error {
	_ = p.ReadCloser.Close()
	// Wait collects the child process; ignore the error since we are
	// intentionally killing it via context cancellation.
	_ = p.cmd.Wait()
	return nil
}
