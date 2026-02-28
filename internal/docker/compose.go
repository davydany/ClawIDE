package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
)

// IsDockerRunning checks whether the Docker daemon is reachable by running
// `docker info` with a short timeout. Returns true if the command succeeds.
func IsDockerRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// Publisher represents a single port binding in the Docker Compose JSON output.
type Publisher struct {
	URL           string `json:"URL"`
	TargetPort    int    `json:"TargetPort"`
	PublishedPort int    `json:"PublishedPort"`
	Protocol      string `json:"Protocol"`
}

// Service represents a running Docker Compose service as reported by `docker compose ps`.
type Service struct {
	Name       string      `json:"Name"`
	Service    string      `json:"Service"`
	Status     string      `json:"Status"`
	State      string      `json:"State"`
	Health     string      `json:"Health"`
	Publishers []Publisher `json:"Publishers"`
}

// PS runs `docker compose ps --format json` in the given project directory
// and returns the list of services. When the command exits with a non-zero
// status (e.g. unhealthy services), it still attempts to parse whatever JSON
// was written to stdout so the caller gets service data alongside the error.
func PS(projectPath string) ([]Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "ps", "--format", "json")
	cmd.Dir = projectPath

	// Capture stderr separately so we can include it in error messages.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()

	// Parse whatever stdout we got, even if the command failed. Docker Compose
	// returns exit code 1 when services are unhealthy/restarting but still
	// writes valid NDJSON to stdout.
	services := parseNDJSON(out)

	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		// Return services AND the error — let the caller decide what to do.
		return services, fmt.Errorf("docker compose ps: %s", errMsg)
	}

	return services, nil
}

// parseNDJSON parses Docker Compose NDJSON output (one JSON object per line)
// into a slice of Service. It also handles the case where some versions emit
// a JSON array on a single line.
func parseNDJSON(data []byte) []Service {
	if len(data) == 0 {
		return nil
	}

	var services []Service
	scanner := bufio.NewScanner(bytes.NewReader(data))
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

	return services
}

// Up runs `docker compose up -d` in the given project directory.
func Up(projectPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "up", "-d")
	cmd.Dir = projectPath
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg != "" {
			return fmt.Errorf("docker compose up: %s", msg)
		}
		return fmt.Errorf("docker compose up: %w", err)
	}
	return nil
}

// Down runs `docker compose down` in the given project directory.
func Down(projectPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "down")
	cmd.Dir = projectPath
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg != "" {
			return fmt.Errorf("docker compose down: %s", msg)
		}
		return fmt.Errorf("docker compose down: %w", err)
	}
	return nil
}

// StartService starts a single service via `docker compose start <service>`.
func StartService(projectPath, service string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "start", service)
	cmd.Dir = projectPath
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg != "" {
			return fmt.Errorf("docker compose start %s: %s", service, msg)
		}
		return fmt.Errorf("docker compose start %s: %w", service, err)
	}
	return nil
}

// StopService stops a single service via `docker compose stop <service>`.
func StopService(projectPath, service string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "stop", service)
	cmd.Dir = projectPath
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg != "" {
			return fmt.Errorf("docker compose stop %s: %s", service, msg)
		}
		return fmt.Errorf("docker compose stop %s: %w", service, err)
	}
	return nil
}

// RestartService restarts a single service via `docker compose restart <service>`.
func RestartService(projectPath, service string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "restart", service)
	cmd.Dir = projectPath
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg != "" {
			return fmt.Errorf("docker compose restart %s: %s", service, msg)
		}
		return fmt.Errorf("docker compose restart %s: %w", service, err)
	}
	return nil
}

// LogsStream starts `docker compose logs -f <service>` and returns a ReadCloser
// that streams the combined stdout/stderr output. The caller is responsible for
// closing the returned reader, which will also terminate the underlying process.
// The provided context can be used to cancel the log stream. When tail > 0,
// only the last N lines are returned before streaming begins.
func LogsStream(ctx context.Context, projectPath, service string, tail int) (io.ReadCloser, error) {
	args := []string{"compose", "logs", "-f", "--no-log-prefix"}
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}
	args = append(args, service)
	cmd := exec.CommandContext(ctx, "docker", args...)
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

// formatPublishers converts the Publishers array into a human-readable ports
// string like "0.0.0.0:3001->5432/tcp, 0.0.0.0:3000->8000/tcp".
func formatPublishers(pubs []Publisher) string {
	if len(pubs) == 0 {
		return ""
	}
	var parts []string
	for _, p := range pubs {
		if p.PublishedPort == 0 {
			// Container port only, no host binding
			parts = append(parts, fmt.Sprintf("%d/%s", p.TargetPort, p.Protocol))
		} else if p.URL != "" {
			parts = append(parts, fmt.Sprintf("%s:%d->%d/%s", p.URL, p.PublishedPort, p.TargetPort, p.Protocol))
		} else {
			parts = append(parts, fmt.Sprintf("%d->%d/%s", p.PublishedPort, p.TargetPort, p.Protocol))
		}
	}
	return strings.Join(parts, ", ")
}

// ToDockerServices converts the internal Service slice to the model representation
// used by handlers and templates.
func ToDockerServices(services []Service) []model.DockerService {
	out := make([]model.DockerService, len(services))
	for i, s := range services {
		out[i] = model.DockerService{
			Name:    s.Name,
			Service: s.Service,
			Status:  s.Status,
			State:   s.State,
			Health:  s.Health,
			Ports:   formatPublishers(s.Publishers),
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
