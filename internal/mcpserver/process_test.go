package mcpserver

import (
	"strings"
	"testing"
	"time"
)

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(3)
	rb.Append("a")
	rb.Append("b")
	rb.Append("c")

	lines := rb.Lines()
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
		t.Errorf("unexpected lines: %v", lines)
	}

	// Overflow
	rb.Append("d")
	lines = rb.Lines()
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines after overflow, got %d", len(lines))
	}
	if lines[0] != "b" || lines[1] != "c" || lines[2] != "d" {
		t.Errorf("expected oldest evicted: %v", lines)
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer(5)
	rb.Append("x")
	rb.Append("y")
	rb.Clear()
	if len(rb.Lines()) != 0 {
		t.Error("expected empty after clear")
	}
}

func TestProcessManager_StartStop(t *testing.T) {
	pm := NewProcessManager()

	config := MCPServerConfig{
		Name:    "test-echo",
		Command: "echo",
		Args:    []string{"hello", "world"},
	}

	err := pm.Start("project", "test-echo", config)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for the short-lived process to finish
	time.Sleep(200 * time.Millisecond)

	info := pm.GetStatus("project", "test-echo")
	if info.Status != "stopped" && info.Status != "error" {
		t.Errorf("expected stopped or error after echo exits, got %q", info.Status)
	}

	logs := pm.GetLogs("project", "test-echo")
	if len(logs) == 0 {
		t.Error("expected captured log output")
	} else if !strings.Contains(logs[0], "hello world") {
		t.Errorf("expected 'hello world' in logs, got %q", logs[0])
	}
}

func TestProcessManager_StartLongRunning(t *testing.T) {
	pm := NewProcessManager()

	config := MCPServerConfig{
		Name:    "test-sleep",
		Command: "sleep",
		Args:    []string{"30"},
	}

	err := pm.Start("global", "test-sleep", config)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Should be running
	time.Sleep(100 * time.Millisecond)
	info := pm.GetStatus("global", "test-sleep")
	if info.Status != "running" {
		t.Errorf("expected running, got %q", info.Status)
	}

	// Stop
	err = pm.Stop("global", "test-sleep")
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	info = pm.GetStatus("global", "test-sleep")
	if info.Status == "running" {
		t.Error("expected not running after stop")
	}
}

func TestProcessManager_DoubleStart(t *testing.T) {
	pm := NewProcessManager()

	config := MCPServerConfig{
		Name:    "test-sleep",
		Command: "sleep",
		Args:    []string{"30"},
	}

	err := pm.Start("project", "test-sleep", config)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pm.Stop("project", "test-sleep")

	time.Sleep(100 * time.Millisecond)

	err = pm.Start("project", "test-sleep", config)
	if err == nil {
		t.Error("expected error starting already-running process")
	}
}

func TestProcessManager_Restart(t *testing.T) {
	pm := NewProcessManager()

	config := MCPServerConfig{
		Name:    "test-sleep",
		Command: "sleep",
		Args:    []string{"30"},
	}

	err := pm.Start("project", "test-sleep", config)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	err = pm.Restart("project", "test-sleep", config)
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	info := pm.GetStatus("project", "test-sleep")
	if info.Status != "running" {
		t.Errorf("expected running after restart, got %q", info.Status)
	}

	pm.Stop("project", "test-sleep")
}

func TestProcessManager_StatusAll(t *testing.T) {
	pm := NewProcessManager()

	pm.Start("project", "a", MCPServerConfig{Command: "sleep", Args: []string{"30"}})
	pm.Start("global", "b", MCPServerConfig{Command: "echo", Args: []string{"hi"}})
	defer pm.StopAll()

	time.Sleep(200 * time.Millisecond)

	all := pm.StatusAll()
	if len(all) != 2 {
		t.Errorf("expected 2 entries, got %d", len(all))
	}
}

func TestProcessManager_StopAll(t *testing.T) {
	pm := NewProcessManager()

	pm.Start("project", "a", MCPServerConfig{Command: "sleep", Args: []string{"30"}})
	pm.Start("project", "b", MCPServerConfig{Command: "sleep", Args: []string{"30"}})
	time.Sleep(100 * time.Millisecond)

	pm.StopAll()
	time.Sleep(200 * time.Millisecond)

	all := pm.StatusAll()
	for key, status := range all {
		if status == "running" {
			t.Errorf("process %s still running after StopAll", key)
		}
	}
}

func TestProcessManager_NoCommand(t *testing.T) {
	pm := NewProcessManager()
	err := pm.Start("project", "empty", MCPServerConfig{})
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestProcessManager_GetLogsUnknown(t *testing.T) {
	pm := NewProcessManager()
	logs := pm.GetLogs("project", "nonexistent")
	if logs != nil {
		t.Error("expected nil logs for unknown process")
	}
}
