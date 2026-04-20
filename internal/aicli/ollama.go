package aicli

import (
	"context"
	"fmt"
	"strings"
)

// OllamaProvider shells out to the `ollama` CLI via:
//
//	ollama run <model> <prompt>
//
// Stdout is plain text. Ollama also supports --format json, but for a simple Q&A we don't need
// structured output; returning raw stdout matches how a human would read the response.
type OllamaProvider struct {
	registry *Registry
}

// NewOllamaProvider constructs the provider.
func NewOllamaProvider(r *Registry) *OllamaProvider {
	return &OllamaProvider{registry: r}
}

func (p *OllamaProvider) ID() string          { return "ollama" }
func (p *OllamaProvider) DisplayName() string { return "Ollama (local)" }
func (p *OllamaProvider) Binary() string      { return "ollama" }

// AvailableModels is intentionally small — users can install any ollama model, but the UI only
// presents these well-known ones. Can be extended later by calling `ollama list` at startup.
func (p *OllamaProvider) AvailableModels() []ModelInfo {
	return []ModelInfo{
		{ID: "llama3.1", DisplayName: "Llama 3.1"},
		{ID: "llama3.2", DisplayName: "Llama 3.2"},
		{ID: "mistral", DisplayName: "Mistral"},
		{ID: "qwen2.5-coder", DisplayName: "Qwen 2.5 Coder"},
	}
}

func (p *OllamaProvider) SupportsStreaming() bool { return false }
func (p *OllamaProvider) RunStreaming(_ context.Context, _ Request, _ func(StreamChunk)) error {
	return fmt.Errorf("ollama does not support streaming yet")
}

func (p *OllamaProvider) Run(ctx context.Context, req Request) (Response, error) {
	if err := ValidateModel(p, req.Model); err != nil {
		return Response{}, err
	}
	args := []string{"run", req.Model, req.Prompt}
	res, err := runSubprocess(ctx, p.registry, req, p.Binary(), args)
	if err != nil {
		return Response{}, err
	}
	text := strings.TrimSpace(res.Stdout)
	if text == "" {
		return Response{}, fmt.Errorf("ollama returned empty output")
	}
	return Response{
		Text:       text,
		RawOutput:  truncate(res.Stdout, 64*1024),
		Provider:   p.ID(),
		Model:      req.Model,
		DurationMs: res.Duration.Milliseconds(),
	}, nil
}
