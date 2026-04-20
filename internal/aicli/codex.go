package aicli

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// CodexProvider shells out to the OpenAI `codex` CLI via:
//
//	codex exec --skip-git-repo-check -s read-only -m <model> -o <tmpfile> <prompt>
//
// Codex writes the final assistant message to the file named by -o so we don't have to parse its
// JSONL event stream. `--skip-git-repo-check` lets global-scope tasks run even when WorkDir isn't
// a git repo. `-s read-only` denies file mutations since an "Ask AI" is a read-only question.
//
// Note: codex's non-interactive subcommand is `exec`, NOT `-p` (which is `--profile` in codex).
// This tripped us up initially — verified by running `codex --help` and `codex exec --help`.
type CodexProvider struct {
	registry *Registry
}

// NewCodexProvider constructs the provider.
func NewCodexProvider(r *Registry) *CodexProvider {
	return &CodexProvider{registry: r}
}

func (p *CodexProvider) ID() string          { return "codex" }
func (p *CodexProvider) DisplayName() string { return "OpenAI Codex" }
func (p *CodexProvider) Binary() string      { return "codex" }

// AvailableModels is best-guess for the current codex CLI release. Exact model strings may drift;
// users can always bypass by editing this list and rebuilding, and a future enhancement could
// fetch this list from `codex models` if codex adds such a command.
func (p *CodexProvider) AvailableModels() []ModelInfo {
	return []ModelInfo{
		{ID: "gpt-5-codex", DisplayName: "GPT-5 Codex (Recommended)"},
		{ID: "gpt-5", DisplayName: "GPT-5"},
		{ID: "o4-mini", DisplayName: "o4-mini (Fast)"},
	}
}

func (p *CodexProvider) SupportsStreaming() bool { return false }
func (p *CodexProvider) RunStreaming(_ context.Context, _ Request, _ func(StreamChunk)) error {
	return fmt.Errorf("codex does not support streaming")
}

func (p *CodexProvider) Run(ctx context.Context, req Request) (Response, error) {
	if err := ValidateModel(p, req.Model); err != nil {
		return Response{}, err
	}
	// Create a tempfile for codex's -o output. Codex writes the final assistant message to this
	// file. We create + close + pass the path + read + delete.
	tmp, err := os.CreateTemp("", "clawide-codex-*.txt")
	if err != nil {
		return Response{}, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	args := []string{
		"exec",
		"--skip-git-repo-check",
		"-s", "read-only",
		"-m", req.Model,
		"-o", tmpPath,
		req.Prompt,
	}
	res, err := runSubprocess(ctx, p.registry, req, p.Binary(), args)
	if err != nil {
		return Response{}, err
	}

	data, readErr := os.ReadFile(tmpPath)
	if readErr != nil {
		// Fall back to stdout if the output file is missing — covers cases where codex exits
		// without writing the file (e.g. a very short prompt that gets blocked by policy).
		return Response{}, fmt.Errorf("codex output file unreadable: %w (stdout: %s)", readErr, truncate(res.Stdout, 512))
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return Response{}, fmt.Errorf("codex output file was empty (stdout: %s)", truncate(res.Stdout, 512))
	}
	return Response{
		Text:       text,
		RawOutput:  truncate(res.Stdout, 64*1024),
		Provider:   p.ID(),
		Model:      req.Model,
		DurationMs: res.Duration.Milliseconds(),
	}, nil
}
