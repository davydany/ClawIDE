// Package aicli provides a pluggable abstraction over local AI CLI tools (claude, codex, ollama,
// gemini, etc.) so features like the task manager's "Ask AI" can shell out to whichever CLI the
// user has installed and authenticated. Each provider implements argv building and output parsing
// for its specific CLI; subprocess execution, timeouts, and error handling are shared via the
// wizard.Executor so providers don't reinvent that plumbing.
//
// This is deliberately separate from internal/wizard/llm_client.go, which makes HTTP calls to
// provider APIs with a stored API key. The CLI path inherits the user's local auth transparently
// and is what we want for user-triggered prompts in a project directory.
package aicli

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"sync"
	"time"

	"github.com/davydany/ClawIDE/internal/wizard"
)

// Request is what a caller hands a provider when asking a question. Every field is validated by
// the handler before reaching a provider, so providers can trust the values.
type Request struct {
	Prompt  string
	Model   string // must be one of the provider's AvailableModels
	WorkDir string // cwd for the subprocess; empty means "current process cwd"
	Timeout time.Duration
}

// Response is what a provider returns after a successful invocation.
type Response struct {
	Text       string        // the extracted answer text, ready to store as a comment body
	RawOutput  string        // full stdout for debugging, up to 64 KB
	Provider   string        // echoes provider ID
	Model      string        // echoes model used
	DurationMs int64
}

// ModelInfo is a single entry in a provider's model list.
type ModelInfo struct {
	ID          string `json:"id"`           // passed to the CLI's --model flag
	DisplayName string `json:"display_name"` // shown in the UI
}

// CLIProvider is the contract every AI CLI integration must satisfy.
type CLIProvider interface {
	// ID is a stable string used in API requests and stored in comment author fields.
	// Must match the regex [a-z][a-z0-9-]*.
	ID() string

	// DisplayName is shown to humans.
	DisplayName() string

	// Binary is the exec.LookPath name (e.g. "claude", "codex").
	Binary() string

	// AvailableModels returns the list of model IDs the provider is willing to forward. The
	// first entry is the recommended default.
	AvailableModels() []ModelInfo

	// Run executes the provider for the given request and returns the parsed response. Callers
	// always go through this method — they never build argv or exec.Command themselves.
	Run(ctx context.Context, req Request) (Response, error)
}

// IsInstalled reports whether a provider's binary is reachable via exec.LookPath. Providers that
// don't expose a Binary() (e.g. stubs) should embed a binary name that's guaranteed missing so
// this returns false. Stored in the registry so we don't re-stat PATH on every API call.
func IsInstalled(p CLIProvider) bool {
	_, err := exec.LookPath(p.Binary())
	return err == nil
}

// Registry holds registered CLI providers. It's built once at server startup and shared across
// handlers. All methods are safe for concurrent use.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]CLIProvider
	installed map[string]bool // cached result of exec.LookPath at Register time
	executor  *wizard.Executor
}

// NewRegistry returns an empty registry that will use the given Executor for subprocess calls.
// Passing nil creates one with a 120-second default timeout.
func NewRegistry(exec *wizard.Executor) *Registry {
	if exec == nil {
		exec = wizard.NewExecutor(120 * time.Second)
	}
	return &Registry{
		providers: make(map[string]CLIProvider),
		installed: make(map[string]bool),
		executor:  exec,
	}
}

// Executor exposes the shared executor so provider implementations can pass it their own argv.
func (r *Registry) Executor() *wizard.Executor {
	return r.executor
}

// Register adds a provider to the registry and probes for its binary. Duplicate IDs overwrite.
func (r *Registry) Register(p CLIProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.ID()] = p
	r.installed[p.ID()] = IsInstalled(p)
}

// Get returns a provider by ID. Second return is false if not registered.
func (r *Registry) Get(id string) (CLIProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	return p, ok
}

// IsInstalled returns the cached install status for the given provider ID.
func (r *Registry) IsInstalled(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.installed[id]
}

// List returns all registered providers sorted by ID. Used to populate the /api/ai/providers
// response.
func (r *Registry) List() []CLIProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]CLIProvider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

// ValidateModel is a helper for providers: returns nil if the requested model is in the allow
// list, otherwise returns a descriptive error. Providers call this at the start of Run().
func ValidateModel(p CLIProvider, model string) error {
	if model == "" {
		return fmt.Errorf("model is required")
	}
	for _, m := range p.AvailableModels() {
		if m.ID == model {
			return nil
		}
	}
	return fmt.Errorf("model %q is not available for provider %q", model, p.ID())
}

// runSubprocess is the shared subprocess helper every provider uses. It wraps wizard.Executor so
// timeout, stdout/stderr capture, and non-zero exit handling are identical across providers.
func runSubprocess(ctx context.Context, reg *Registry, req Request, binary string, args []string) (wizard.CommandResult, error) {
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	res := reg.executor.Run(ctx, req.WorkDir, binary, args...)
	if res.Err != nil {
		return res, fmt.Errorf("%s failed (exit %d): %s", binary, res.ExitCode, truncate(res.Stderr, 2048))
	}
	return res, nil
}

// truncate clips a string to at most n bytes and appends an ellipsis marker if it was clipped.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…(truncated)"
}
