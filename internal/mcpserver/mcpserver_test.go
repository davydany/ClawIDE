package mcpserver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestListServers_BothFiles(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.mcp.json")
	projectPath := filepath.Join(dir, "project.mcp.json")

	writeTestFile(t, globalPath, map[string]serverEntry{
		"global-server": {Command: "npx", Args: []string{"start"}},
	})
	writeTestFile(t, projectPath, map[string]serverEntry{
		"project-server": {Command: "node", Args: []string{"server.js"}},
	})

	servers, err := ListServers(globalPath, projectPath)
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}

	scopes := map[string]string{}
	for _, s := range servers {
		scopes[s.Name] = s.Scope
	}
	if scopes["global-server"] != "global" {
		t.Errorf("expected global scope for global-server")
	}
	if scopes["project-server"] != "project" {
		t.Errorf("expected project scope for project-server")
	}
}

func TestListServers_MissingFiles(t *testing.T) {
	servers, err := ListServers("/nonexistent/global.json", "/nonexistent/project.json")
	if err != nil {
		t.Fatalf("ListServers with missing files: %v", err)
	}
	if len(servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(servers))
	}
}

func TestListServers_EmptyPaths(t *testing.T) {
	servers, err := ListServers("", "")
	if err != nil {
		t.Fatalf("ListServers with empty paths: %v", err)
	}
	if len(servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(servers))
	}
}

func TestGetServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	writeTestFile(t, path, map[string]serverEntry{
		"my-server": {Command: "python", Args: []string{"-m", "mcp"}},
	})

	srv, err := GetServer(path, "my-server")
	if err != nil {
		t.Fatalf("GetServer: %v", err)
	}
	if srv.Command != "python" {
		t.Errorf("expected command 'python', got %q", srv.Command)
	}

	_, err = GetServer(path, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent server")
	}
}

func TestCreateServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")

	// Create on new file
	err := CreateServer(path, MCPServerConfig{
		Name:    "new-server",
		Command: "npx",
		Args:    []string{"-y", "pkg"},
		Env:     map[string]string{"KEY": "val"},
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	srv, err := GetServer(path, "new-server")
	if err != nil {
		t.Fatalf("GetServer after create: %v", err)
	}
	if srv.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", srv.Command)
	}
	if srv.Env["KEY"] != "val" {
		t.Errorf("expected env KEY=val")
	}

	// Duplicate should fail
	err = CreateServer(path, MCPServerConfig{Name: "new-server", Command: "x"})
	if err == nil {
		t.Error("expected error for duplicate server name")
	}
}

func TestUpdateServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	writeTestFile(t, path, map[string]serverEntry{
		"srv": {Command: "old-cmd", Args: []string{"a"}},
	})

	// Update fields
	err := UpdateServer(path, "srv", MCPServerConfig{
		Name:    "srv",
		Command: "new-cmd",
		Args:    []string{"b", "c"},
	})
	if err != nil {
		t.Fatalf("UpdateServer: %v", err)
	}
	srv, _ := GetServer(path, "srv")
	if srv.Command != "new-cmd" {
		t.Errorf("expected updated command")
	}
	if len(srv.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(srv.Args))
	}
}

func TestUpdateServer_Rename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	writeTestFile(t, path, map[string]serverEntry{
		"old-name": {Command: "cmd"},
	})

	err := UpdateServer(path, "old-name", MCPServerConfig{
		Name:    "new-name",
		Command: "cmd",
	})
	if err != nil {
		t.Fatalf("UpdateServer rename: %v", err)
	}

	_, err = GetServer(path, "old-name")
	if err == nil {
		t.Error("old name should not exist after rename")
	}
	srv, err := GetServer(path, "new-name")
	if err != nil {
		t.Fatalf("new name not found after rename: %v", err)
	}
	if srv.Command != "cmd" {
		t.Errorf("command should be preserved")
	}
}

func TestDeleteServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	writeTestFile(t, path, map[string]serverEntry{
		"srv-a": {Command: "a"},
		"srv-b": {Command: "b"},
	})

	err := DeleteServer(path, "srv-a")
	if err != nil {
		t.Fatalf("DeleteServer: %v", err)
	}

	_, err = GetServer(path, "srv-a")
	if err == nil {
		t.Error("expected srv-a to be deleted")
	}
	_, err = GetServer(path, "srv-b")
	if err != nil {
		t.Error("srv-b should still exist")
	}

	// Delete nonexistent
	err = DeleteServer(path, "nonexistent")
	if err == nil {
		t.Error("expected error deleting nonexistent server")
	}
}

func TestMoveServer(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.mcp.json")
	dstPath := filepath.Join(dir, "dst.mcp.json")

	writeTestFile(t, srcPath, map[string]serverEntry{
		"moving-srv": {Command: "cmd", Args: []string{"a"}, Env: map[string]string{"K": "V"}},
	})

	err := MoveServer(srcPath, dstPath, "moving-srv")
	if err != nil {
		t.Fatalf("MoveServer: %v", err)
	}

	// Should be gone from source
	_, err = GetServer(srcPath, "moving-srv")
	if err == nil {
		t.Error("server should be removed from source")
	}

	// Should exist in destination
	srv, err := GetServer(dstPath, "moving-srv")
	if err != nil {
		t.Fatalf("server not found in destination: %v", err)
	}
	if srv.Command != "cmd" || srv.Env["K"] != "V" {
		t.Error("server data should be preserved after move")
	}
}

func TestMoveServer_DuplicateInDst(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.mcp.json")
	dstPath := filepath.Join(dir, "dst.mcp.json")

	writeTestFile(t, srcPath, map[string]serverEntry{
		"srv": {Command: "a"},
	})
	writeTestFile(t, dstPath, map[string]serverEntry{
		"srv": {Command: "b"},
	})

	err := MoveServer(srcPath, dstPath, "srv")
	if err == nil {
		t.Error("expected error when server already exists in destination")
	}
}

func TestRoundTripPreservesUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")

	// Write file with an extra top-level field
	original := `{
  "mcpServers": {
    "srv": {"command": "cmd", "args": []}
  },
  "customField": {"nested": true}
}`
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	// Do a round-trip via CreateServer
	err := CreateServer(path, MCPServerConfig{Name: "new-srv", Command: "x", Args: []string{}})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	// Read back and verify customField is preserved
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}

	if _, ok := result["customField"]; !ok {
		t.Error("customField should be preserved after round-trip")
	}
	if _, ok := result["mcpServers"]; !ok {
		t.Error("mcpServers should still exist")
	}
}

// writeTestFile is a helper that creates a .mcp.json with the given servers.
func writeTestFile(t *testing.T, path string, servers map[string]serverEntry) {
	t.Helper()
	f := struct {
		MCPServers map[string]serverEntry `json:"mcpServers"`
	}{MCPServers: servers}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
