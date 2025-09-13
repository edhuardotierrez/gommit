package types

// ProviderConfig holds the configuration for a specific LLM provider
type ProviderConfig struct {
	APIKey      string  `json:"api_key,omitempty"`
	URI         string  `json:"uri,omitempty"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	CommitStyle string  `json:"commit_style,omitempty"`
}

// Config holds the application configuration
type Config struct {
	DefaultProvider string                    `json:"default_provider"`
	Providers       map[string]ProviderConfig `json:"providers"`
	MaxTokens       int                       `json:"max_tokens"`
	CommitStyle     string                    `json:"commit_style"`
	TruncateLines   int                       `json:"truncate_lines,omitempty"`
	MaxLineWidth    int                       `json:"max_line_width"`
}

// Default values for configuration
const (
	DefaultMaxTokens     = 500
	DefaultCommitStyle   = "conventional" // can be: conventional, simple, detailed
	DefaultTruncateLines = 1000           // default number of context lines in git diff
	DefaultMaxLineWidth  = 300
)

// ProviderTypes represents an LLM provider configuration
type ProviderTypes struct {
	Title      string
	Name       ProviderName
	ConfigVars map[string]string
	Required   []string
	Optional   []string
}

// ProviderName represents the name of an LLM provider
type ProviderName string

// Provider constants
const (
	ProviderAnthropic ProviderName = "anthropic"
	ProviderGoogle    ProviderName = "google"
	ProviderOpenAI    ProviderName = "openai"
	ProviderOllama    ProviderName = "ollama"
)
