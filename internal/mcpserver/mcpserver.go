package mcpserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MCPServerConfig represents a single MCP server entry in .mcp.json.
type MCPServerConfig struct {
	Name      string            `json:"name"`
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env,omitempty"`
	AutoStart bool              `json:"autoStart,omitempty"`
	Scope     string            `json:"scope"` // runtime only: "global" or "project"
}

// serverEntry is the on-disk representation of a single MCP server in .mcp.json.
type serverEntry struct {
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env,omitempty"`
	AutoStart bool              `json:"autoStart,omitempty"`
}

// mcpFile is the on-disk representation of .mcp.json.
// We use json.RawMessage to preserve unknown top-level fields during round-trips.
type mcpFile struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
	extra      map[string]json.RawMessage // non-mcpServers top-level keys
}

// GlobalMCPFilePath returns the path to the global .mcp.json.
func GlobalMCPFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", ".mcp.json")
}

// ProjectMCPFilePath returns the path to a project's .mcp.json.
func ProjectMCPFilePath(projectPath string) string {
	return filepath.Join(projectPath, ".mcp.json")
}

// ListServers reads both global and project .mcp.json files and returns all servers.
func ListServers(globalPath, projectPath string) ([]MCPServerConfig, error) {
	var all []MCPServerConfig

	if globalPath != "" {
		servers, err := readServers(globalPath, "global")
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("reading global MCP config: %w", err)
		}
		all = append(all, servers...)
	}

	if projectPath != "" {
		servers, err := readServers(projectPath, "project")
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("reading project MCP config: %w", err)
		}
		all = append(all, servers...)
	}

	return all, nil
}

// HasServer checks if a server with the given name exists in a .mcp.json file.
func HasServer(filePath, name string) bool {
	f, err := readMCPFile(filePath)
	if err != nil {
		return false
	}
	_, exists := f.MCPServers[name]
	return exists
}

// GetServer reads a single server from a .mcp.json file.
func GetServer(filePath, name string) (*MCPServerConfig, error) {
	servers, err := readServers(filePath, "")
	if err != nil {
		return nil, err
	}

	for i := range servers {
		if servers[i].Name == name {
			return &servers[i], nil
		}
	}
	return nil, fmt.Errorf("server %q not found", name)
}

// CreateServer adds a new MCP server entry to a .mcp.json file.
func CreateServer(filePath string, server MCPServerConfig) error {
	f, err := readMCPFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if f == nil {
		f = &mcpFile{MCPServers: make(map[string]json.RawMessage)}
	}

	if _, exists := f.MCPServers[server.Name]; exists {
		return fmt.Errorf("server %q already exists", server.Name)
	}

	entry := serverEntry{
		Command:   server.Command,
		Args:      server.Args,
		Env:       server.Env,
		AutoStart: server.AutoStart,
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling server entry: %w", err)
	}

	f.MCPServers[server.Name] = raw
	return writeMCPFile(filePath, f)
}

// UpdateServer updates an existing MCP server entry, supporting rename.
func UpdateServer(filePath, oldName string, server MCPServerConfig) error {
	f, err := readMCPFile(filePath)
	if err != nil {
		return err
	}

	if _, exists := f.MCPServers[oldName]; !exists {
		return fmt.Errorf("server %q not found", oldName)
	}

	// If renaming, check target doesn't exist
	newName := server.Name
	if newName == "" {
		newName = oldName
	}
	if newName != oldName {
		if _, exists := f.MCPServers[newName]; exists {
			return fmt.Errorf("server %q already exists", newName)
		}
		delete(f.MCPServers, oldName)
	}

	entry := serverEntry{
		Command:   server.Command,
		Args:      server.Args,
		Env:       server.Env,
		AutoStart: server.AutoStart,
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling server entry: %w", err)
	}

	f.MCPServers[newName] = raw
	return writeMCPFile(filePath, f)
}

// DeleteServer removes an MCP server entry from a .mcp.json file.
func DeleteServer(filePath, name string) error {
	f, err := readMCPFile(filePath)
	if err != nil {
		return err
	}

	if _, exists := f.MCPServers[name]; !exists {
		return fmt.Errorf("server %q not found", name)
	}

	delete(f.MCPServers, name)
	return writeMCPFile(filePath, f)
}

// MoveServer moves an MCP server from one .mcp.json file to another.
func MoveServer(srcPath, dstPath, name string) error {
	srcFile, err := readMCPFile(srcPath)
	if err != nil {
		return fmt.Errorf("reading source: %w", err)
	}

	raw, exists := srcFile.MCPServers[name]
	if !exists {
		return fmt.Errorf("server %q not found in source", name)
	}

	dstFile, err := readMCPFile(dstPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading destination: %w", err)
	}
	if dstFile == nil {
		dstFile = &mcpFile{MCPServers: make(map[string]json.RawMessage)}
	}

	if _, exists := dstFile.MCPServers[name]; exists {
		return fmt.Errorf("server %q already exists in destination", name)
	}

	dstFile.MCPServers[name] = raw
	if err := writeMCPFile(dstPath, dstFile); err != nil {
		return fmt.Errorf("writing destination: %w", err)
	}

	delete(srcFile.MCPServers, name)
	if err := writeMCPFile(srcPath, srcFile); err != nil {
		return fmt.Errorf("writing source: %w", err)
	}

	return nil
}

// readServers reads all servers from a .mcp.json file and tags them with scope.
func readServers(filePath, scope string) ([]MCPServerConfig, error) {
	f, err := readMCPFile(filePath)
	if err != nil {
		return nil, err
	}

	var servers []MCPServerConfig
	for name, raw := range f.MCPServers {
		var entry serverEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue // skip malformed entries
		}
		servers = append(servers, MCPServerConfig{
			Name:      name,
			Command:   entry.Command,
			Args:      entry.Args,
			Env:       entry.Env,
			AutoStart: entry.AutoStart,
			Scope:     scope,
		})
	}
	return servers, nil
}

// readMCPFile reads and parses a .mcp.json file, preserving unknown fields.
func readMCPFile(path string) (*mcpFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse top-level as generic map to capture all keys
	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal(data, &topLevel); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	f := &mcpFile{
		MCPServers: make(map[string]json.RawMessage),
		extra:      make(map[string]json.RawMessage),
	}

	// Extract mcpServers
	if raw, ok := topLevel["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &f.MCPServers); err != nil {
			return nil, fmt.Errorf("parsing mcpServers in %s: %w", path, err)
		}
	}

	// Preserve all other top-level keys
	for k, v := range topLevel {
		if k != "mcpServers" {
			f.extra[k] = v
		}
	}

	return f, nil
}

// writeMCPFile writes the mcpFile back to disk, preserving unknown fields.
func writeMCPFile(path string, f *mcpFile) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Build top-level map with preserved extra fields
	topLevel := make(map[string]interface{})
	for k, v := range f.extra {
		topLevel[k] = v
	}
	topLevel["mcpServers"] = f.MCPServers

	data, err := json.MarshalIndent(topLevel, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}

	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
