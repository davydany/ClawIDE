package portdetect

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposePort represents a port mapping extracted from a docker-compose.yml file.
type ComposePort struct {
	Service       string `json:"service"`
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
}

// composeFile is a minimal representation of a docker-compose.yml for port extraction.
type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Ports []string `yaml:"ports"`
}

// ExtractComposePorts reads a docker-compose.yml (or docker-compose.yaml) from
// projectPath and returns all declared port mappings.
func ExtractComposePorts(projectPath string) ([]ComposePort, error) {
	data, err := readComposeFile(projectPath)
	if err != nil {
		return nil, err
	}

	var cf composeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parse docker-compose: %w", err)
	}

	var ports []ComposePort
	for serviceName, svc := range cf.Services {
		for _, portSpec := range svc.Ports {
			parsed, err := parsePortSpec(serviceName, portSpec)
			if err != nil {
				// Skip malformed port specs rather than failing the whole call.
				continue
			}
			ports = append(ports, parsed...)
		}
	}

	return ports, nil
}

// readComposeFile tries docker-compose.yml then docker-compose.yaml in the
// given directory.
func readComposeFile(dir string) ([]byte, error) {
	candidates := []string{
		filepath.Join(dir, "docker-compose.yml"),
		filepath.Join(dir, "docker-compose.yaml"),
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
	}

	return nil, fmt.Errorf("no docker-compose.yml found in %s", dir)
}

// parsePortSpec parses a Docker Compose port string and returns one or more
// ComposePort entries. Supported formats:
//
//	"8080:80"               -> host 8080, container 80
//	"3000:3000"             -> host 3000, container 3000
//	"127.0.0.1:5432:5432"  -> host 5432, container 5432 (IP prefix stripped)
//	"8080"                  -> host 8080, container 8080 (single port)
//	"8080-8082:80-82"      -> host 8080->80, 8081->81, 8082->82 (range)
func parsePortSpec(service, spec string) ([]ComposePort, error) {
	// Strip protocol suffix if present, e.g. "8080:80/tcp"
	if idx := strings.Index(spec, "/"); idx != -1 {
		spec = spec[:idx]
	}

	parts := strings.Split(spec, ":")

	var hostPart, containerPart string

	switch len(parts) {
	case 1:
		// "8080" -- single port, same for host and container
		hostPart = parts[0]
		containerPart = parts[0]
	case 2:
		// "8080:80"
		hostPart = parts[0]
		containerPart = parts[1]
	case 3:
		// "127.0.0.1:5432:5432" -- IP:host:container
		hostPart = parts[1]
		containerPart = parts[2]
	default:
		return nil, fmt.Errorf("unsupported port format: %s", spec)
	}

	// Handle port ranges like "8080-8082:80-82"
	if strings.Contains(hostPart, "-") && strings.Contains(containerPart, "-") {
		return parsePortRange(service, hostPart, containerPart)
	}

	hostPort, err := strconv.Atoi(hostPart)
	if err != nil {
		return nil, fmt.Errorf("invalid host port %q: %w", hostPart, err)
	}

	containerPort, err := strconv.Atoi(containerPart)
	if err != nil {
		return nil, fmt.Errorf("invalid container port %q: %w", containerPart, err)
	}

	return []ComposePort{{
		Service:       service,
		HostPort:      hostPort,
		ContainerPort: containerPort,
	}}, nil
}

// parsePortRange expands a port range spec like "8080-8082" / "80-82" into
// individual ComposePort entries.
func parsePortRange(service, hostRange, containerRange string) ([]ComposePort, error) {
	hostStart, hostEnd, err := splitRange(hostRange)
	if err != nil {
		return nil, fmt.Errorf("invalid host port range %q: %w", hostRange, err)
	}

	containerStart, containerEnd, err := splitRange(containerRange)
	if err != nil {
		return nil, fmt.Errorf("invalid container port range %q: %w", containerRange, err)
	}

	hostCount := hostEnd - hostStart + 1
	containerCount := containerEnd - containerStart + 1
	if hostCount != containerCount {
		return nil, fmt.Errorf("port range mismatch: host has %d ports, container has %d", hostCount, containerCount)
	}

	var ports []ComposePort
	for i := 0; i < hostCount; i++ {
		ports = append(ports, ComposePort{
			Service:       service,
			HostPort:      hostStart + i,
			ContainerPort: containerStart + i,
		})
	}

	return ports, nil
}

// splitRange splits "8080-8082" into (8080, 8082).
func splitRange(s string) (int, int, error) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("not a range: %s", s)
	}

	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	if end < start {
		return 0, 0, fmt.Errorf("invalid range: %d > %d", start, end)
	}

	return start, end, nil
}
