package editor

import (
	"testing"
)

func TestAllEditors(t *testing.T) {
	editors := AllEditors()
	if len(editors) == 0 {
		t.Fatal("AllEditors() returned empty list")
	}

	// Verify known editors exist
	ids := make(map[string]bool)
	for _, e := range editors {
		ids[e.ID] = true
		if e.Name == "" {
			t.Errorf("editor %q has empty name", e.ID)
		}
		if len(e.CLICommands) == 0 {
			t.Errorf("editor %q has no CLI commands", e.ID)
		}
	}

	for _, expected := range []string{"vscode", "cursor", "vim", "neovim", "emacs"} {
		if !ids[expected] {
			t.Errorf("expected editor %q not found in AllEditors()", expected)
		}
	}
}

func TestGetEditor(t *testing.T) {
	e := GetEditor("vscode")
	if e == nil {
		t.Fatal("GetEditor(\"vscode\") returned nil")
	}
	if e.Name != "VS Code" {
		t.Errorf("expected VS Code, got %q", e.Name)
	}
	if e.Terminal {
		t.Error("vscode should not be a terminal editor")
	}

	e = GetEditor("vim")
	if e == nil {
		t.Fatal("GetEditor(\"vim\") returned nil")
	}
	if !e.Terminal {
		t.Error("vim should be a terminal editor")
	}

	e = GetEditor("nonexistent")
	if e != nil {
		t.Error("GetEditor(\"nonexistent\") should return nil")
	}
}

func TestGetEditorName(t *testing.T) {
	if name := GetEditorName("vscode"); name != "VS Code" {
		t.Errorf("expected \"VS Code\", got %q", name)
	}
	if name := GetEditorName("nonexistent"); name != "" {
		t.Errorf("expected empty string for unknown editor, got %q", name)
	}
}

func TestDetectAvailable(t *testing.T) {
	editors := DetectAvailable()
	if len(editors) == 0 {
		t.Fatal("DetectAvailable() returned empty list")
	}
	for _, e := range editors {
		if e.Installed && e.CLI == "" {
			t.Errorf("editor %q is installed but CLI path is empty", e.ID)
		}
		if !e.Installed && e.CLI != "" {
			t.Errorf("editor %q is not installed but has CLI path %q", e.ID, e.CLI)
		}
	}
}

func TestOpenEditor_UnknownEditor(t *testing.T) {
	err := OpenEditor("nonexistent", "/tmp")
	if err == nil {
		t.Error("OpenEditor with unknown editor should return error")
	}
}

func TestOpenEditor_TerminalEditor(t *testing.T) {
	err := OpenEditor("vim", "/tmp")
	if err == nil {
		t.Error("OpenEditor with terminal editor should return error")
	}
}

func TestOpenEditor_InvalidDirectory(t *testing.T) {
	err := OpenEditor("vscode", "--malicious-flag")
	if err == nil {
		t.Error("OpenEditor with flag-like directory should return error")
	}
}

func TestOpenFileExplorer_InvalidDirectory(t *testing.T) {
	err := OpenFileExplorer("--malicious-flag")
	if err == nil {
		t.Error("OpenFileExplorer with flag-like directory should return error")
	}
}

func TestOpenFileExplorer_ValidDirectory(t *testing.T) {
	// /tmp exists on macOS/Linux; this should succeed (opens Finder/xdg-open)
	err := OpenFileExplorer("/tmp")
	if err != nil {
		t.Errorf("OpenFileExplorer(\"/tmp\") returned unexpected error: %v", err)
	}
}
