package aicli

import (
	"context"
	"encoding/json"
	"fmt"
)

// ClaudeProvider shells out to the `claude` CLI via:
//
//	claude -p --output-format json --model <alias> <prompt>
//
// Stdout is a single JSON object containing (among other fields): result (string), is_error (bool),
// session_id (string). The shape was verified by running the CLI locally and the captured sample
// lives in testdata/claude_sample.json.
type ClaudeProvider struct {
	registry *Registry
}

// NewClaudeProvider constructs the provider. The registry reference lets us share the Executor.
func NewClaudeProvider(r *Registry) *ClaudeProvider {
	return &ClaudeProvider{registry: r}
}

func (p *ClaudeProvider) ID() string          { return "claude" }
func (p *ClaudeProvider) DisplayName() string { return "Claude Code" }
func (p *ClaudeProvider) Binary() string      { return "claude" }

// AvailableModels lists the stable model aliases. The CLI also accepts full model IDs, but aliases
// are forward-compatible — when a new sonnet ships, `sonnet` keeps working without a code change.
func (p *ClaudeProvider) AvailableModels() []ModelInfo {
	return []ModelInfo{
		{ID: "sonnet", DisplayName: "Sonnet (Recommended)"},
		{ID: "opus", DisplayName: "Opus (Most capable)"},
		{ID: "haiku", DisplayName: "Haiku (Fast, cost-effective)"},
	}
}

// Run executes claude and extracts the answer text from its JSON output.
func (p *ClaudeProvider) Run(ctx context.Context, req Request) (Response, error) {
	if err := ValidateModel(p, req.Model); err != nil {
		return Response{}, err
	}
	args := []string{
		"-p",
		"--output-format", "json",
		"--model", req.Model,
		req.Prompt,
	}
	res, err := runSubprocess(ctx, p.registry, req, p.Binary(), args)
	if err != nil {
		return Response{}, err
	}

	text, rawErr := parseClaudeOutput(res.Stdout)
	if rawErr != nil {
		return Response{}, fmt.Errorf("claude output parse error: %w (stdout prefix: %s)", rawErr, truncate(res.Stdout, 512))
	}
	return Response{
		Text:       text,
		RawOutput:  truncate(res.Stdout, 64*1024),
		Provider:   p.ID(),
		Model:      req.Model,
		DurationMs: res.Duration.Milliseconds(),
	}, nil
}

// claudeOutput is the subset of claude's --output-format json response we care about.
type claudeOutput struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	IsError   bool   `json:"is_error"`
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
}

// parseClaudeOutput extracts the answer text. If is_error is true, returns an error whose message
// is the .result field (which claude uses for error descriptions in this shape).
func parseClaudeOutput(stdout string) (string, error) {
	if stdout == "" {
		return "", fmt.Errorf("empty stdout")
	}
	var out claudeOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		return "", fmt.Errorf("not valid JSON: %w", err)
	}
	if out.IsError {
		msg := out.Result
		if msg == "" {
			msg = "unknown claude error"
		}
		return "", fmt.Errorf("claude returned is_error=true: %s", msg)
	}
	if out.Result == "" {
		return "", fmt.Errorf("claude output had no .result field")
	}
	return out.Result, nil
}
