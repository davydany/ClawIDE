package editor

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Editor represents a code editor that can be launched from ClawIDE.
type Editor struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	CLICommands []string `json:"cli_commands"` // ordered fallback list
	Terminal    bool     `json:"terminal"`      // true for vim/neovim/emacs
}

// AvailableEditor extends Editor with detection status.
type AvailableEditor struct {
	Editor
	Installed bool   `json:"installed"`
	CLI       string `json:"cli,omitempty"` // resolved command that was found
}

// allEditors is the canonical list of supported editors.
var allEditors = []Editor{
	{ID: "vscode", Name: "VS Code", CLICommands: []string{"code"}, Terminal: false},
	{ID: "cursor", Name: "Cursor", CLICommands: []string{"cursor"}, Terminal: false},
	{ID: "zed", Name: "Zed", CLICommands: []string{"zed"}, Terminal: false},
	{ID: "sublime", Name: "Sublime Text", CLICommands: []string{"subl"}, Terminal: false},
	{ID: "vim", Name: "Vim", CLICommands: []string{"vim"}, Terminal: true},
	{ID: "neovim", Name: "Neovim", CLICommands: []string{"nvim"}, Terminal: true},
	{ID: "emacs", Name: "Emacs", CLICommands: []string{"emacs"}, Terminal: true},
	{ID: "intellij", Name: "IntelliJ IDEA", CLICommands: []string{"idea"}, Terminal: false},
	{ID: "webstorm", Name: "WebStorm", CLICommands: []string{"webstorm"}, Terminal: false},
	{ID: "fleet", Name: "Fleet", CLICommands: []string{"fleet"}, Terminal: false},
	{ID: "nova", Name: "Nova", CLICommands: []string{"nova"}, Terminal: false},
}

// AllEditors returns the full list of supported editors.
func AllEditors() []Editor {
	out := make([]Editor, len(allEditors))
	copy(out, allEditors)
	return out
}

// GetEditor returns a single editor by ID, or nil if not found.
func GetEditor(id string) *Editor {
	for _, e := range allEditors {
		if e.ID == id {
			cp := e
			return &cp
		}
	}
	return nil
}

// GetEditorName returns the display name for an editor ID, or "" if unknown.
func GetEditorName(id string) string {
	e := GetEditor(id)
	if e == nil {
		return ""
	}
	return e.Name
}

// DetectAvailable checks which editors are installed on the system.
func DetectAvailable() []AvailableEditor {
	result := make([]AvailableEditor, 0, len(allEditors))
	for _, e := range allEditors {
		ae := AvailableEditor{Editor: e}
		for _, cmd := range e.CLICommands {
			if path, err := exec.LookPath(cmd); err == nil {
				ae.Installed = true
				ae.CLI = path
				break
			}
		}
		result = append(result, ae)
	}
	return result
}

// OpenEditor launches the given editor in the specified directory.
// It returns an error if the editor is unknown, is a terminal editor, or
// the CLI command cannot be found.
func OpenEditor(id, directory string) error {
	e := GetEditor(id)
	if e == nil {
		return fmt.Errorf("unknown editor: %s", id)
	}
	if e.Terminal {
		return fmt.Errorf("terminal editor %q cannot be launched via exec", e.Name)
	}

	// Sanitize directory: reject anything that looks like a flag.
	if strings.HasPrefix(directory, "-") {
		return fmt.Errorf("invalid directory path")
	}

	for _, cmd := range e.CLICommands {
		path, err := exec.LookPath(cmd)
		if err != nil {
			continue
		}
		proc := exec.Command(path, directory)
		if err := proc.Start(); err != nil {
			return fmt.Errorf("failed to start %s: %w", cmd, err)
		}
		// Fire-and-forget: release the process so it doesn't block.
		go proc.Wait()
		return nil
	}

	return fmt.Errorf("no CLI command found for %s", e.Name)
}

// OpenFileExplorer opens the given directory in the OS file explorer.
// macOS: Finder (open), Windows: Explorer (explorer.exe), Linux: xdg-open.
func OpenFileExplorer(directory string) error {
	if strings.HasPrefix(directory, "-") {
		return fmt.Errorf("invalid directory path")
	}

	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{directory}
	case "windows":
		cmd = "explorer.exe"
		args = []string{directory}
	default: // linux, freebsd, etc.
		cmd = "xdg-open"
		args = []string{directory}
	}

	path, err := exec.LookPath(cmd)
	if err != nil {
		return fmt.Errorf("%s not found: %w", cmd, err)
	}

	proc := exec.Command(path, args...)
	if err := proc.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", cmd, err)
	}
	go proc.Wait()
	return nil
}
