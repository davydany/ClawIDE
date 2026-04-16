package aicli

import (
	"context"
	"fmt"
)

// GeminiProvider is a stub. Google's `gemini` CLI wasn't installed in the dev environment when
// this was built, so the exact flags and output format haven't been verified. The provider is
// registered so the UI can show it as "not available" rather than silently missing, and it's
// structured so flipping the stub into a real implementation only requires filling in Run().
type GeminiProvider struct {
	registry *Registry
}

func NewGeminiProvider(r *Registry) *GeminiProvider {
	return &GeminiProvider{registry: r}
}

func (p *GeminiProvider) ID() string          { return "gemini" }
func (p *GeminiProvider) DisplayName() string { return "Gemini (stub)" }
func (p *GeminiProvider) Binary() string      { return "gemini" }

func (p *GeminiProvider) AvailableModels() []ModelInfo {
	return []ModelInfo{
		{ID: "gemini-2.5-pro", DisplayName: "Gemini 2.5 Pro"},
		{ID: "gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash"},
	}
}

func (p *GeminiProvider) Run(ctx context.Context, req Request) (Response, error) {
	return Response{}, fmt.Errorf("gemini CLI provider is not yet implemented — flags and output format need to be verified against a real install")
}

// RegisterDefaults registers the built-in providers (claude, codex, ollama, gemini) on the given
// registry. Called once at server startup from cmd/clawide/main.go.
func RegisterDefaults(r *Registry) {
	r.Register(NewClaudeProvider(r))
	r.Register(NewCodexProvider(r))
	r.Register(NewOllamaProvider(r))
	r.Register(NewGeminiProvider(r))
}
