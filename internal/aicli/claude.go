package aicli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
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

func (p *ClaudeProvider) SupportsStreaming() bool { return true }

// RunStreaming uses `claude -p --bare --verbose --output-format stream-json` to stream JSONL
// events. Each line is parsed: "assistant" messages with text content produce chunks, and the
// final "result" message produces the Done chunk.
func (p *ClaudeProvider) RunStreaming(ctx context.Context, req Request, onChunk func(StreamChunk)) error {
	if err := ValidateModel(p, req.Model); err != nil {
		return err
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{
		"-p", "--bare", "--verbose",
		"--output-format", "stream-json",
		"--model", req.Model,
		req.Prompt,
	}
	cmd := exec.CommandContext(ctx, p.Binary(), args...)
	cmd.Dir = req.WorkDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	var fullResult string
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 256*1024), 256*1024) // large buffer for long lines
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		text, result, isDone := parseStreamLine(line)
		if text != "" {
			onChunk(StreamChunk{Text: text})
		}
		if isDone {
			fullResult = result
		}
	}

	if err := cmd.Wait(); err != nil {
		onChunk(StreamChunk{Error: fmt.Sprintf("claude exited with error: %v", err), Done: true})
		return err
	}

	if fullResult == "" {
		fullResult = "(no result)"
	}
	onChunk(StreamChunk{Text: fullResult, Done: true})
	return nil
}

// parseStreamLine extracts displayable text from a single JSONL line of stream-json output.
// Returns (textChunk, fullResult, isDone).
func parseStreamLine(line string) (string, string, bool) {
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return "", "", false
	}
	typ, _ := obj["type"].(string)

	switch typ {
	case "assistant":
		// Extract text from message.content[].text where type=="text"
		msg, _ := obj["message"].(map[string]any)
		if msg == nil {
			return "", "", false
		}
		content, _ := msg["content"].([]any)
		for _, c := range content {
			block, _ := c.(map[string]any)
			if block == nil {
				continue
			}
			if blockType, _ := block["type"].(string); blockType == "text" {
				text, _ := block["text"].(string)
				if text != "" {
					return text, "", false
				}
			}
		}
	case "result":
		result, _ := obj["result"].(string)
		isError, _ := obj["is_error"].(bool)
		if isError {
			return "", "", false
		}
		return "", result, true
	}
	return "", "", false
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
