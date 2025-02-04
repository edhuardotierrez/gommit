package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/edhuardotierrez/gommit/internal/colors"

	"github.com/henomis/lingoose/llm/anthropic"
	"github.com/henomis/lingoose/llm/ollama"
	"github.com/henomis/lingoose/llm/openai"
	"github.com/henomis/lingoose/thread"

	"github.com/edhuardotierrez/gommit/internal/git"
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
}

// GetAvailableModels returns a list of available models for a given provider
func GetAvailableModels(provider types.ProviderName) []string {
	switch provider {
	case types.ProviderOpenAI:
		return []string{
			"gpt-4o",
			"gpt-4o-mini",
		}
	case types.ProviderAnthropic:
		return []string{
			"claude-3-5-sonnet-latest",
			"claude-3-5-haiku-latest",
			"claude-3-haiku-20240229",
		}
	case types.ProviderOllama:
		return []string{
			"llama3",
			"mistral",
		}
	default:
		return []string{}
	}
}

// truncateDiff shortens the diff content while keeping the most relevant parts
func truncateDiff(diff string) string {
	if len(diff) <= maxDiffLength {
		return diff
	}

	// Take first and last parts of the diff to keep context
	halfLength := maxDiffLength / 2
	return diff[:halfLength] + "\n...[truncated]...\n" + diff[len(diff)-halfLength:]
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

// GenerateCommitMessage generates a commit message based on the staged changes
func GenerateCommitMessage(cfg *types.Config, changes []git.StagedChange, provider string, selectedProvider types.ProviderConfig) (string, error) {

	providerName := types.ProviderName(provider)

	// Prepare the changes summary with truncated diffs
	var summary strings.Builder
	for _, change := range changes {
		fmt.Fprintf(&summary, "File: %s (Status: %s)\n", change.Path, change.Status)
		truncatedDiff := truncateDiff(change.Diff)
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

	// Create a new thread with system and user messages
	userMessage := fmt.Sprintf("Please generate a commit message for the following changes (using '%s' as commit style):\n\n%s", style, summary.String())
	myThread := thread.New().
		AddMessage(thread.NewSystemMessage().AddContent(thread.NewTextContent(promptToUse))).
		AddMessage(thread.NewUserMessage().AddContent(
			thread.NewTextContent(userMessage),
		))

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
	var err error
	switch providerName {
	case types.ProviderOpenAI:
		_ = os.Setenv("OPENAI_API_KEY", selectedProvider.APIKey)
		llmClient := openai.New().
			WithModel(openai.Model(selectedProvider.Model)).
			WithTemperature(float32(selectedProvider.Temperature))
		err = llmClient.Generate(context.Background(), myThread)

	case types.ProviderAnthropic:
		_ = os.Setenv("ANTHROPIC_API_KEY", selectedProvider.APIKey)
		llmClient := anthropic.New().
			WithModel(selectedProvider.Model).
			WithTemperature(selectedProvider.Temperature)
		err = llmClient.Generate(context.Background(), myThread)

	case types.ProviderOllama:
		_ = os.Setenv("OLLAMA_API_KEY", selectedProvider.APIKey)
		_ = os.Setenv("OLLAMA_URI", selectedProvider.URI)
		llmClient := ollama.New().
			WithModel(selectedProvider.Model).
			WithTemperature(selectedProvider.Temperature).
			WithEndpoint(selectedProvider.URI)
		err = llmClient.Generate(context.Background(), myThread)

	default:
		return "", fmt.Errorf("unsupported LLM provider: %s", provider)
	}

	if err != nil {
		return "", fmt.Errorf("error generating commit message: %w", err)
	}

	if len(myThread.Messages) < 3 {
		return "", fmt.Errorf("no commit message generated by the assistant")
	}

	lastMessage := myThread.Messages[len(myThread.Messages)-1]
	if lastMessage == nil || len(lastMessage.Contents) == 0 {
		return "", fmt.Errorf("no commit message content found. check your provider configuration")
	}
	contents := lastMessage.Contents[0]
	messageString := contents.Data.(string)

	return messageString, nil
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
