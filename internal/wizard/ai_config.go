package wizard

import "fmt"

// AIProvider represents the AI provider type for code generation
type AIProvider string

const (
	AIProviderAnthropic AIProvider = "anthropic"
	AIProviderOpenAI    AIProvider = "openai"
	AIProviderGemini    AIProvider = "gemini"
	AIProviderOllama    AIProvider = "ollama"
)

// AIModel represents a specific AI model configuration
type AIModel struct {
	ID       string `json:"id"`
	Provider AIProvider `json:"provider"`
	Name     string `json:"name"`
	// For OpenAI/Gemini/Ollama: the model identifier
	ModelID string `json:"model_id"`
}

// AIConfig holds configuration for AI-powered code generation
type AIConfig struct {
	// Enabled controls whether AI generation is available
	Enabled bool `json:"enabled"`

	// Provider is the configured AI provider (anthropic, openai, gemini, ollama)
	Provider AIProvider `json:"provider"`

	// APIKey for the selected provider (encrypted at rest)
	APIKey string `json:"api_key"`

	// Optional: APISecret for providers that need it (e.g., Anthropic has no secret, but others might)
	APISecret string `json:"api_secret"`

	// Optional: BaseURL for self-hosted providers like Ollama
	BaseURL string `json:"base_url"`

	// Model specifies which model to use
	Model string `json:"model"`

	// ModelList holds available models for this provider
	AvailableModels []AIModel `json:"available_models"`

	// Temperature controls randomness (0.0 to 1.0)
	Temperature float32 `json:"temperature"`

	// MaxTokens limits response length
	MaxTokens int `json:"max_tokens"`

	// BudgetConfig for tracking API usage
	BudgetConfig *BudgetConfig `json:"budget_config"`
}

// BudgetConfig tracks API usage and costs
type BudgetConfig struct {
	// MonthlyLimit in USD
	MonthlyLimit float64 `json:"monthly_limit"`

	// CurrentSpend tracks spending in current month
	CurrentSpend float64 `json:"current_spend"`

	// WarningThresholdPercent (e.g., 80 = warn at 80% usage)
	WarningThresholdPercent int `json:"warning_threshold_percent"`

	// HardCap: if true, stop generation when limit reached
	HardCap bool `json:"hard_cap"`
}

// DefaultAIConfig returns sensible defaults for AI configuration
func DefaultAIConfig() *AIConfig {
	return &AIConfig{
		Enabled:     false,
		Provider:    AIProviderAnthropic,
		Temperature: 0.7,
		MaxTokens:   2048,
		BudgetConfig: &BudgetConfig{
			MonthlyLimit:            100,
			WarningThresholdPercent: 80,
			HardCap:                 false,
		},
		AvailableModels: []AIModel{
			{
				ID:       "claude-opus",
				Provider: AIProviderAnthropic,
				Name:     "Claude 3.5 Opus",
				ModelID:  "claude-opus-4-20250514",
			},
			{
				ID:       "claude-sonnet",
				Provider: AIProviderAnthropic,
				Name:     "Claude 3.5 Sonnet",
				ModelID:  "claude-sonnet-4-20250514",
			},
			{
				ID:       "claude-haiku",
				Provider: AIProviderAnthropic,
				Name:     "Claude 3.5 Haiku",
				ModelID:  "claude-haiku-3-20250307",
			},
		},
	}
}

// IsConfigured checks if AI config has all required fields for actual use
func (ac *AIConfig) IsConfigured() bool {
	return ac != nil && ac.Enabled && string(ac.Provider) != "" && ac.Model != "" && ac.APIKey != ""
}

// Validate checks if AI configuration is valid
func (ac *AIConfig) Validate() error {
	if !ac.Enabled {
		return nil // disabled config is valid
	}

	if ac.Provider == "" {
		return fmt.Errorf("AI provider is required when enabled")
	}

	if ac.Model == "" {
		return fmt.Errorf("AI model is required when enabled")
	}

	if ac.APIKey == "" {
		return fmt.Errorf("API key is required when enabled")
	}

	if ac.Temperature < 0.0 || ac.Temperature > 1.0 {
		return fmt.Errorf("temperature must be between 0.0 and 1.0, got %f", ac.Temperature)
	}

	if ac.MaxTokens < 256 || ac.MaxTokens > 128000 {
		return fmt.Errorf("max_tokens must be between 256 and 128000, got %d", ac.MaxTokens)
	}

	if ac.Provider == AIProviderOllama && ac.BaseURL == "" {
		return fmt.Errorf("base_url is required for Ollama provider")
	}

	return nil
}

// GetAvailableProviders returns all supported AI providers
func GetAvailableProviders() []AIProvider {
	return []AIProvider{
		AIProviderAnthropic,
		AIProviderOpenAI,
		AIProviderGemini,
		AIProviderOllama,
	}
}

// ProviderModels returns default models for each provider (February 2026 latest)
func ProviderModels(provider AIProvider) []AIModel {
	switch provider {
	case AIProviderAnthropic:
		return []AIModel{
			{
				ID:       "claude-opus",
				Provider: AIProviderAnthropic,
				Name:     "Claude 3.5 Opus (Latest, most powerful)",
				ModelID:  "claude-opus-4-20250514",
			},
			{
				ID:       "claude-sonnet",
				Provider: AIProviderAnthropic,
				Name:     "Claude 3.5 Sonnet (Recommended - balanced)",
				ModelID:  "claude-sonnet-4-20250514",
			},
			{
				ID:       "claude-haiku",
				Provider: AIProviderAnthropic,
				Name:     "Claude 3.5 Haiku (Fast, cost-effective)",
				ModelID:  "claude-haiku-3-20250307",
			},
		}
	case AIProviderOpenAI:
		return []AIModel{
			{
				ID:       "gpt-4o",
				Provider: AIProviderOpenAI,
				Name:     "GPT-4o (Latest, recommended)",
				ModelID:  "gpt-4o-2025-01-01",
			},
			{
				ID:       "gpt-4o-mini",
				Provider: AIProviderOpenAI,
				Name:     "GPT-4o Mini (Fast, cost-effective)",
				ModelID:  "gpt-4o-mini-2025-01-01",
			},
			{
				ID:       "gpt-4-turbo",
				Provider: AIProviderOpenAI,
				Name:     "GPT-4 Turbo (Legacy)",
				ModelID:  "gpt-4-turbo-2025-01-01",
			},
		}
	case AIProviderGemini:
		return []AIModel{
			{
				ID:       "gemini-2-flash",
				Provider: AIProviderGemini,
				Name:     "Gemini 2.0 Flash (Recommended, fastest)",
				ModelID:  "gemini-2.0-flash-exp",
			},
			{
				ID:       "gemini-2-pro",
				Provider: AIProviderGemini,
				Name:     "Gemini 2.0 Pro (Most capable)",
				ModelID:  "gemini-2.0-pro-exp",
			},
			{
				ID:       "gemini-1.5-pro",
				Provider: AIProviderGemini,
				Name:     "Gemini 1.5 Pro (Legacy)",
				ModelID:  "gemini-1.5-pro",
			},
		}
	case AIProviderOllama:
		return []AIModel{
			{
				ID:       "mistral-large",
				Provider: AIProviderOllama,
				Name:     "Mistral Large (Recommended)",
				ModelID:  "mistral-large",
			},
			{
				ID:       "llama2",
				Provider: AIProviderOllama,
				Name:     "Llama 2 (70B)",
				ModelID:  "llama2",
			},
			{
				ID:       "neural-chat",
				Provider: AIProviderOllama,
				Name:     "Neural Chat",
				ModelID:  "neural-chat",
			},
		}
	default:
		return []AIModel{}
	}
}
