package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davydany/ClawIDE/internal/model"
	"gopkg.in/yaml.v3"
)

// PortMapping represents a single host:container port binding extracted from a
// Docker Compose service definition.
type PortMapping struct {
	Service       string `json:"service"`
	HostPort      string `json:"host_port"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol"`
}

// ParseComposeFile reads and parses the Docker Compose file from the given
// project directory. It checks for docker-compose.yml, docker-compose.yaml,
// compose.yml, and compose.yaml in that order, returning the first one found.
func ParseComposeFile(projectPath string) (*model.ComposeConfig, error) {
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	var data []byte
	var readErr error
	for _, name := range candidates {
		data, readErr = os.ReadFile(filepath.Join(projectPath, name))
		if readErr == nil {
			break
		}
	}
	if readErr != nil {
		return nil, fmt.Errorf("no compose file found in %s: %w", projectPath, readErr)
	}

	var cfg model.ComposeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing compose file: %w", err)
	}

	return &cfg, nil
}

// ExtractPorts extracts all port mappings from a parsed ComposeConfig.
// It handles the short syntax formats commonly used in compose files:
//   - "8080:80"          -> host 8080, container 80, tcp
//   - "8080:80/udp"      -> host 8080, container 80, udp
//   - "80"               -> host unspecified, container 80, tcp
//   - "127.0.0.1:8080:80" -> host 8080, container 80, tcp (IP binding ignored)
func ExtractPorts(cfg *model.ComposeConfig) []PortMapping {
	var mappings []PortMapping

	for svcName, svc := range cfg.Services {
		for _, portStr := range svc.Ports {
			mapping := parsePortString(svcName, portStr)
			mappings = append(mappings, mapping)
		}
	}

	return mappings
}

// parsePortString parses a single Docker Compose port string into a PortMapping.
func parsePortString(service, raw string) PortMapping {
	pm := PortMapping{
		Service:  service,
		Protocol: "tcp",
	}

	// Extract protocol suffix if present (e.g. "8080:80/udp").
	portPart := raw
	if idx := strings.LastIndex(raw, "/"); idx != -1 {
		pm.Protocol = raw[idx+1:]
		portPart = raw[:idx]
	}

	// Split on ":" to handle the various formats.
	parts := strings.Split(portPart, ":")

	switch len(parts) {
	case 1:
		// "80" - container port only
		pm.ContainerPort = parts[0]
	case 2:
		// "8080:80" - host:container
		pm.HostPort = parts[0]
		pm.ContainerPort = parts[1]
	case 3:
		// "127.0.0.1:8080:80" - ip:host:container (ip binding is ignored)
		pm.HostPort = parts[1]
		pm.ContainerPort = parts[2]
	}

	return pm
}
