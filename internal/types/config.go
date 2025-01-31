package types

// ProviderConfig holds the configuration for a specific LLM provider
type ProviderConfig struct {
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

// Config holds the application configuration
type Config struct {
	DefaultProvider string                    `json:"default_provider"`
	Providers       map[string]ProviderConfig `json:"providers"`
	MaxTokens       int                       `json:"max_tokens,omitempty"`
	CommitStyle     string                    `json:"commit_style,omitempty"`
}

// Default values for configuration
const (
	DefaultMaxTokens   = 150
	DefaultCommitStyle = "conventional" // can be: conventional, simple, detailed
)

// ProviderTypes represents an LLM provider configuration
type ProviderTypes struct {
	Title     string
	Name      ProviderName
	EnvVarKey string
}

// ProviderName represents the name of an LLM provider
type ProviderName string

// Provider constants
const (
	ProviderOpenAI    ProviderName = "openai"
	ProviderAnthropic ProviderName = "anthropic"
	ProviderCohere    ProviderName = "cohere"
	ProviderOllama    ProviderName = "ollama"
) 