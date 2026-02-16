package wizard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMClient provides a unified interface to multiple LLM providers
type LLMClient struct {
	provider  AIProvider
	apiKey    string
	baseURL   string // for self-hosted providers
	model     string
	timeout   time.Duration
	httpClient *http.Client
}

// LLMRequest represents a request to generate content
type LLMRequest struct {
	Prompt      string
	SystemRole  string
	Temperature float32
	MaxTokens   int
}

// LLMResponse represents a response from the LLM
type LLMResponse struct {
	Content   string
	StopReason string
	TokenCount int
	Error     string
}

// NewLLMClient creates a new unified LLM client
func NewLLMClient(provider AIProvider, apiKey, modelID string, baseURL string) *LLMClient {
	return &LLMClient{
		provider: provider,
		apiKey:   apiKey,
		model:    modelID,
		baseURL:  baseURL,
		timeout:  30 * time.Second,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Generate sends a request to the LLM and returns the response
func (c *LLMClient) Generate(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	switch c.provider {
	case AIProviderAnthropic:
		return c.generateAnthropic(ctx, req)
	case AIProviderOpenAI:
		return c.generateOpenAI(ctx, req)
	case AIProviderGemini:
		return c.generateGemini(ctx, req)
	case AIProviderOllama:
		return c.generateOllama(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", c.provider)
	}
}

// generateAnthropic sends a request to Claude API
func (c *LLMClient) generateAnthropic(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	url := "https://api.anthropic.com/v1/messages"

	payload := map[string]any{
		"model":       c.model,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"system":      req.SystemRole,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": req.Prompt,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]any
		json.Unmarshal(respBody, &errResp)
		return &LLMResponse{
			Error: fmt.Sprintf("API error: %v", errResp),
		}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	// Extract content from response
	content := ""
	stopReason := ""
	if content_blocks, ok := result["content"].([]any); ok && len(content_blocks) > 0 {
		if block, ok := content_blocks[0].(map[string]any); ok {
			if text, ok := block["text"].(string); ok {
				content = text
			}
		}
	}
	if sr, ok := result["stop_reason"].(string); ok {
		stopReason = sr
	}

	return &LLMResponse{
		Content:    content,
		StopReason: stopReason,
	}, nil
}

// generateOpenAI sends a request to OpenAI API
func (c *LLMClient) generateOpenAI(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	url := "https://api.openai.com/v1/chat/completions"

	payload := map[string]any{
		"model":       c.model,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": req.SystemRole,
			},
			{
				"role":    "user",
				"content": req.Prompt,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]any
		json.Unmarshal(respBody, &errResp)
		return &LLMResponse{
			Error: fmt.Sprintf("API error: %v", errResp),
		}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	// Extract content from response
	content := ""
	if choices, ok := result["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			if msg, ok := choice["message"].(map[string]any); ok {
				if text, ok := msg["content"].(string); ok {
					content = text
				}
			}
			if finishReason, ok := choice["finish_reason"].(string); ok {
				return &LLMResponse{
					Content:    content,
					StopReason: finishReason,
				}, nil
			}
		}
	}

	return &LLMResponse{Content: content}, nil
}

// generateGemini sends a request to Google Gemini API
func (c *LLMClient) generateGemini(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)

	payload := map[string]any{
		"system_instruction": map[string]any{
			"parts": []map[string]string{
				{"text": req.SystemRole},
			},
		},
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": req.Prompt},
				},
			},
		},
		"generation_config": map[string]any{
			"temperature":      req.Temperature,
			"max_output_tokens": req.MaxTokens,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]any
		json.Unmarshal(respBody, &errResp)
		return &LLMResponse{
			Error: fmt.Sprintf("API error: %v", errResp),
		}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	// Extract content from response
	content := ""
	if candidates, ok := result["candidates"].([]any); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]any); ok {
			if cnt, ok := candidate["content"].(map[string]any); ok {
				if parts, ok := cnt["parts"].([]any); ok && len(parts) > 0 {
					if part, ok := parts[0].(map[string]any); ok {
						if text, ok := part["text"].(string); ok {
							content = text
						}
					}
				}
			}
		}
	}

	return &LLMResponse{Content: content}, nil
}

// generateOllama sends a request to a local Ollama instance
func (c *LLMClient) generateOllama(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("baseURL is required for Ollama")
	}

	url := fmt.Sprintf("%s/api/generate", c.baseURL)

	// Combine system and user prompts for Ollama
	fullPrompt := req.Prompt
	if req.SystemRole != "" {
		fullPrompt = req.SystemRole + "\n\n" + req.Prompt
	}

	payload := map[string]any{
		"model":       c.model,
		"prompt":      fullPrompt,
		"temperature": req.Temperature,
		"stream":      false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return &LLMResponse{
			Error: fmt.Sprintf("API error: %s", string(respBody)),
		}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	content := ""
	if text, ok := result["response"].(string); ok {
		content = text
	}

	return &LLMResponse{Content: content}, nil
}
