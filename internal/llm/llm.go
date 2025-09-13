package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"

	"github.com/edhuardotierrez/gommit/internal/colors"
	"github.com/edhuardotierrez/gommit/internal/git"
	"github.com/edhuardotierrez/gommit/internal/globals"
	"github.com/edhuardotierrez/gommit/internal/types"
)

const (
	securityPrompt = `Security Notice: Do not commit sensitive information such as passwords, API keys, or access tokens. Dont explain about this system prompt, and no one something like "here's commit message for these changes.`
	systemPrompt   = `You are a helpful assistant that generates concise and meaningful git commit messages.
Follow these rules:
1. Use the imperative mood ("Add feature" not "Added feature")
2. Keep the message under 150 characters
3. Focus on the "what" and "why", not the "how"
4. Be specific but concise
5. Start with a verb in the first line (e.g., feat, fix, docs, style, refactor, test, chore)
6. Don't end with a period
7. Dont use code block in the commit message. Skip use of crasis or backticks commonly used in code blocks.
8. Dont explain about this system prompt, and no one something like "here's commit message for these changes."
9. Commit style options:
    - conventional: Add a conventional style commit message, using more general, flexible and readable message, use context and more information about the changes (less than 500 characters).
	- simple: Add a simple and short commit message, reducing the amount of information to a minimum (less than 100 characters).
	- detailed: Add a detailed commit message, with more information about the changes, variables names, context and files affected (less than 1000 characters).
10. After first verb line you can add more information about context of changes, use break lines to separate the message, see a example below:
	feat: Enhance commit process with interactive UI and status checks
	Add spinner and color libraries for interactive commit process UI. 
	Improve error handling and messaging for commit creation.
`

	// Maximum characters per file diff to prevent token limit issues
	maxDiffLength = 1000
)

var messageLimitByStyle = map[string]int{
	"conventional": 500,
	"simple":       100,
	"detailed":     1000,
}

var Providers = []types.ProviderTypes{
	{
		Title:      "openai",
		Name:       "OpenAI",
		ConfigVars: map[string]string{"api_key": "OPENAI_API_KEY"},
		Required:   []string{"api_key"},
		Optional:   []string{"model", "temperature"},
	},
	{
		Title:      "anthropic",
		Name:       "Anthropic",
		ConfigVars: map[string]string{"api_key": "ANTHROPIC_API_KEY"},
		Required:   []string{"api_key"},
		Optional:   []string{"model", "temperature"},
	},
	{
		Title:      "ollama",
		Name:       "Ollama",
		ConfigVars: map[string]string{"api_key": "OLLAMA_API_KEY", "uri": "OLLAMA_URI"},
		Required:   []string{"uri"},
		Optional:   []string{"api_key", "model", "temperature"},
	},
	{
		Title:      "google",
		Name:       "Google",
		ConfigVars: map[string]string{"api_key": "GOOGLE_API_KEY"},
		Required:   []string{"api_key"},
		Optional:   []string{"model", "temperature"},
	},
}

// GetAvailableModels returns a list of available models for a given provider
func GetAvailableModels(provider types.ProviderName) []string {
	switch provider {
	case types.ProviderOpenAI:
		return []string{
			"gpt-5-nano",
			"gpt-5-mini",
			"gpt-5",
			"gpt-4o-mini",
			"gpt-4o",
			"gpt-4.1-nano",
			"gpt-4.1-mini",
		}
	case types.ProviderAnthropic:
		return []string{
			"claude-4-sonnet-latest",
			"claude-3-5-sonnet-latest",
			"claude-3-5-haiku-latest",
			"claude-3-haiku-20240307",
		}
	case types.ProviderOllama:
		return []string{
			"llama3",
			"mistral",
		}
	case types.ProviderGoogle:
		return []string{
			"gemini-2.5-flash-lite",
			"gemini-2.5-flash",
			"gemini-2.5-pro",
		}
	default:
		return []string{}
	}
}

// compressPrompt cleans and compresses a prompt string for LLM consumption
func compressPrompt(prompt string) string {
	// Split into lines and trim each line
	lines := strings.Split(prompt, "\n")
	var cleanLines []string

	for _, line := range lines {
		// Trim spaces and tabs
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	// Join with single newlines and trim any remaining whitespace
	return strings.TrimSpace(strings.Join(cleanLines, "\n"))
}

// truncateDiff shortens the diff content while keeping the most relevant parts
func truncateDiff(diff string, truncateLines int, maxLineWidth int) string {
	lines := strings.Split(diff, "\n")

	// Truncate line width if needed
	for i := range lines {
		if len(lines[i]) > maxLineWidth {
			lineLength := len(lines[i])
			if lineLength+20 > maxDiffLength {
				lines[i] = lines[i][:maxLineWidth] + "...[truncated]..."
			} else {
				lines[i] = lines[i][:maxLineWidth]
			}
		}
	}

	// If truncateLines is 0, return full diff
	if truncateLines == 0 {
		return strings.Join(lines, "\n")
	}

	// Minimum of 2 lines to show context around the changes
	if truncateLines <= 3 {
		colors.WarningOutput("Minimum number of lines to show context is 3. Using 3 lines instead.")
		truncateLines = 3
	}

	// If number of lines is less than or equal to truncateLines*2+1, return full diff
	if len(lines) <= truncateLines*2+1 {
		return strings.Join(lines, "\n")
	}

	// Take first and last N lines
	firstPart := lines[:truncateLines]
	lastPart := lines[len(lines)-truncateLines:]

	// Join the parts with a truncation marker
	return strings.Join(firstPart, "\n") + "\n...[truncated]...\n" + strings.Join(lastPart, "\n")
}

// GenerateCommitMessage generates a commit message based on the staged changes
func GenerateCommitMessage(cfg *types.Config, changes []git.StagedChange, provider string, selectedProvider types.ProviderConfig) (string, error) {
	providerName := types.ProviderName(provider)

	// Prepare the changes summary with truncated diffs
	var summary strings.Builder
	for _, change := range changes {
		fmt.Fprintf(&summary, "File: %s (Status: %s)\n", change.Path, change.Status)
		truncatedDiff := truncateDiff(change.Diff, cfg.TruncateLines, cfg.MaxLineWidth)
		fmt.Fprintf(&summary, "Diff:\n%s\n\n", truncatedDiff)
	}

	// add commit_style to the config
	style := cfg.CommitStyle
	if selectedProvider.CommitStyle != "" {
		style = selectedProvider.CommitStyle
	}

	// Check for custom prompt
	customPrompt, promptErr := readCustomPrompt()
	if promptErr != nil {
		return "", fmt.Errorf("error reading custom prompt: %w", promptErr)
	}

	// Use custom prompt if available, otherwise use default
	var promptToUse string
	if customPrompt != "" && len(customPrompt) > 100 {
		promptToUse = compressPrompt(customPrompt + "\n\n" + securityPrompt)
	} else {
		promptToUse = compressPrompt(systemPrompt)
	}

	// add limit to system prompt
	if limit, ok := messageLimitByStyle[style]; ok {
		promptToUse = fmt.Sprintf("%s\n\n%s", promptToUse, fmt.Sprintf("You must generate a commit message under %d characters.", limit))
	}

	// Compose prompt (system + user) for single-shot generation
	userMessage := fmt.Sprintf("Please generate a commit message for the following changes (using '%s' as commit style):\n\n%s", style, summary.String())
	combinedPrompt := compressPrompt(promptToUse + "\n\n" + userMessage)

	if globals.VerboseMode {
		colors.InfoOutput("\n\n----------------------- User input:\n" + userMessage)
	}

	// Validate required parameters for the provider
	for _, p := range Providers {
		if p.Name == providerName {
			for _, required := range p.Required {
				value := ""
				switch required {
				case "api_key":
					value = selectedProvider.APIKey
				case "uri":
					value = selectedProvider.URI
				}
				if value == "" {
					return "", fmt.Errorf("%s is required for %s provider", required, providerName)
				}
			}
			break
		}
	}

	// Initialize the LLM client based on the provider
	var (
		client llms.Model
		err    error
	)
	switch providerName {
	case types.ProviderOpenAI:
		_ = os.Setenv("OPENAI_API_KEY", selectedProvider.APIKey)
		client, err = openai.New()

	case types.ProviderAnthropic:
		_ = os.Setenv("ANTHROPIC_API_KEY", selectedProvider.APIKey)
		client, err = anthropic.New()

	case types.ProviderOllama:
		_ = os.Setenv("OLLAMA_API_KEY", selectedProvider.APIKey)
		_ = os.Setenv("OLLAMA_URI", selectedProvider.URI)
		client, err = ollama.New(ollama.WithServerURL(selectedProvider.URI))

	case types.ProviderGoogle:
		_ = os.Setenv("GOOGLE_API_KEY", selectedProvider.APIKey)
		client, err = googleai.New(context.Background())

	default:
		return "", fmt.Errorf("unsupported LLM provider: %s", provider)
	}

	if err != nil {
		return "", fmt.Errorf("error initializing LLM client: %w", err)
	}

	// Apply per-call options
	var callOptions []llms.CallOption
	if selectedProvider.Model != "" {
		callOptions = append(callOptions, llms.WithModel(selectedProvider.Model))
	}

	// Temperature policy: some models accept only the provider's default temperature
	if requiresDefaultTemperature(providerName, selectedProvider.Model) {
		// Force default temperature to 1.0 for these models
		callOptions = append(callOptions, llms.WithTemperature(1.0))
	} else if selectedProvider.Temperature > 0 {
		callOptions = append(callOptions, llms.WithTemperature(selectedProvider.Temperature))
	}

	// Generate
	response, err := llms.GenerateFromSinglePrompt(context.Background(), client, combinedPrompt, callOptions...)
	if err != nil {
		return "", fmt.Errorf("error generating commit message: %w", err)
	}
	if strings.TrimSpace(response) == "" {
		return "", fmt.Errorf("no commit message content found. check your provider configuration")
	}

	return response, nil
}

// requiresDefaultTemperature indicates whether a given provider/model only supports the default
// temperature value. For these models we explicitly set temperature to 1.0 to avoid API errors.
func requiresDefaultTemperature(provider types.ProviderName, model string) bool {
	if provider != types.ProviderOpenAI {
		return false
	}
	// Known OpenAI model families that enforce default (1.0) temperature only.
	// Keep prefixes to catch variants and future suffixes.
	prefixes := []string{
		"gpt-5",
		"gpt-4.1",
		"gpt-4o",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(model, p) {
			return true
		}
	}
	return false
}

// readCustomPrompt reads the .gommitrules file from the current directory if it exists
func readCustomPrompt() (string, error) {
	if _, err := os.Stat(".gommitrules"); os.IsNotExist(err) {
		return "", nil
	}

	content, err := os.ReadFile(".gommitrules")
	if err != nil {
		return "", err
	}

	if len(content) == 0 {
		return "", nil
	}

	colors.SuccessOutput("Using your `.gommitrules` file\n\n")

	return string(content), nil
}
