package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// ComposeServiceDetail is a JSON-friendly normalized representation of a
// Docker Compose service with all fields as clean types.
type ComposeServiceDetail struct {
	Name          string       `json:"name"`
	Image         string       `json:"image,omitempty"`
	Build         string       `json:"build,omitempty"`
	Ports         []PortDetail `json:"ports"`
	Volumes       []string     `json:"volumes"`
	Environment   []string     `json:"environment"`
	DependsOn     []string     `json:"depends_on"`
	Command       string       `json:"command,omitempty"`
	ContainerName string       `json:"container_name,omitempty"`
	Restart       string       `json:"restart,omitempty"`
}

// PortDetail represents a single port binding with host, container, and protocol.
type PortDetail struct {
	HostPort      string `json:"host_port"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol"`
}

// ExtractServiceDetails converts a parsed ComposeConfig into a sorted slice
// of ComposeServiceDetail with all fields normalized to clean types.
func ExtractServiceDetails(cfg *model.ComposeConfig) []ComposeServiceDetail {
	if cfg == nil {
		return nil
	}

	details := make([]ComposeServiceDetail, 0, len(cfg.Services))
	for name, svc := range cfg.Services {
		d := ComposeServiceDetail{
			Name:          name,
			Image:         svc.Image,
			Build:         normalizeBuild(svc.Build),
			Volumes:       svc.Volumes,
			Environment:   normalizeEnvironment(svc.Environment),
			DependsOn:     normalizeDependsOn(svc.DependsOn),
			Command:       normalizeCommand(svc.Command),
			ContainerName: svc.ContainerName,
			Restart:       svc.Restart,
		}

		// Normalize ports
		d.Ports = make([]PortDetail, 0, len(svc.Ports))
		for _, portStr := range svc.Ports {
			pm := parsePortString(name, portStr)
			d.Ports = append(d.Ports, PortDetail{
				HostPort:      pm.HostPort,
				ContainerPort: pm.ContainerPort,
				Protocol:      pm.Protocol,
			})
		}

		// Ensure nil slices become empty slices for clean JSON
		if d.Volumes == nil {
			d.Volumes = []string{}
		}
		if d.Environment == nil {
			d.Environment = []string{}
		}
		if d.DependsOn == nil {
			d.DependsOn = []string{}
		}

		details = append(details, d)
	}

	sort.Slice(details, func(i, j int) bool {
		return details[i].Name < details[j].Name
	})

	return details
}

// normalizeBuild converts the YAML build field (nil, string, or map) to a
// single string. Maps with "context" and optional "dockerfile" are joined.
func normalizeBuild(v any) string {
	if v == nil {
		return ""
	}
	switch b := v.(type) {
	case string:
		return b
	case map[string]any:
		ctx, _ := b["context"].(string)
		df, _ := b["dockerfile"].(string)
		if df != "" {
			return ctx + "/" + df
		}
		return ctx
	}
	return ""
}

// normalizeEnvironment converts the YAML environment field (nil, string list,
// or key-value map) to a sorted []string of "KEY=VALUE" entries.
func normalizeEnvironment(v any) []string {
	if v == nil {
		return nil
	}
	switch e := v.(type) {
	case []any:
		out := make([]string, 0, len(e))
		for _, item := range e {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		sort.Strings(out)
		return out
	case map[string]any:
		out := make([]string, 0, len(e))
		for k, val := range e {
			out = append(out, fmt.Sprintf("%s=%v", k, val))
		}
		sort.Strings(out)
		return out
	}
	return nil
}

// normalizeDependsOn converts the YAML depends_on field (nil, string list,
// or map with conditions) to a sorted []string of service names.
func normalizeDependsOn(v any) []string {
	if v == nil {
		return nil
	}
	switch d := v.(type) {
	case []any:
		out := make([]string, 0, len(d))
		for _, item := range d {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		sort.Strings(out)
		return out
	case map[string]any:
		out := make([]string, 0, len(d))
		for k := range d {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	return nil
}

// normalizeCommand converts the YAML command field (nil, string, or string
// list) to a single string.
func normalizeCommand(v any) string {
	if v == nil {
		return ""
	}
	switch c := v.(type) {
	case string:
		return c
	case []any:
		parts := make([]string, 0, len(c))
		for _, item := range c {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}
