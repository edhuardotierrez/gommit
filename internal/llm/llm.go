package llm

import (
	"context"
	"fmt"
	"github.com/henomis/lingoose/llm/anthropic"
	"github.com/henomis/lingoose/llm/cohere"
	"github.com/henomis/lingoose/llm/ollama"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"

	"github.com/edhuardotierrez/gommit/internal/git"
	"github.com/edhuardotierrez/gommit/internal/types"
	"github.com/henomis/lingoose/llm/openai"
	"github.com/henomis/lingoose/thread"
)

const systemPrompt = `You are a helpful assistant that generates concise and meaningful git commit messages.
Follow these rules:
1. Use the imperative mood ("Add feature" not "Added feature")
2. Keep the message under 72 characters
3. Focus on the "what" and "why", not the "how"
4. Be specific but concise
5. Start with a verb (e.g., Add, Fix, Update, Refactor, etc.)
6. Don't end with a period`

var Providers = []types.ProviderTypes{
	{
		Title:     "openai",
		Name:      "OpenAI",
		EnvVarKey: "OPENAI_API_KEY",
	},
	{
		Title:     "anthropic",
		Name:      "Anthropic",
		EnvVarKey: "ANTHROPIC_API_KEY",
	},
	{
		Title:     "ollama",
		Name:      "Ollama",
		EnvVarKey: "OLLAMA_API_KEY",
	},
}

// GetAvailableModels returns a list of available models for a given provider
func GetAvailableModels(provider types.ProviderName) []string {
	switch provider {
	case types.ProviderOpenAI:
		return []string{
			"gpt-4",
			"gpt-4-turbo-preview",
			"gpt-3.5-turbo",
		}
	case types.ProviderAnthropic:
		return []string{
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-2.1",
		}
	case types.ProviderCohere:
		return []string{
			"command",
			"command-light",
			"command-nightly",
		}
	case types.ProviderOllama:
		return []string{
			"llama2",
			"mistral",
			"codellama",
		}
	default:
		return []string{}
	}
}

// GenerateCommitMessage generates a commit message based on the staged changes
func GenerateCommitMessage(cfg *types.Config, changes []git.StagedChange) (string, error) {
	// Prepare the changes summary
	var summary strings.Builder
	for _, change := range changes {
		fmt.Fprintf(&summary, "File: %s (Status: %s)\n", change.Path, change.Status)
		fmt.Fprintf(&summary, "Diff:\n%s\n\n", change.Diff)
	}

	// Create a new thread with system and user messages
	myThread := thread.New().
		AddMessage(thread.NewSystemMessage().AddContent(thread.NewTextContent(systemPrompt))).
		AddMessage(thread.NewUserMessage().AddContent(
			thread.NewTextContent(fmt.Sprintf("Please generate a commit message for the following changes:\n\n%s", summary.String())),
		))

	// Get the provider config
	providerConfig, ok := cfg.Providers[cfg.DefaultProvider]
	if !ok {
		return "", fmt.Errorf("provider configuration not found for: %s", cfg.DefaultProvider)
	}

	// Validate API key
	if providerConfig.APIKey == "" {
		return "", fmt.Errorf("API key not configured for provider: %s", cfg.DefaultProvider)
	}

	// Initialize the LLM client based on the provider
	var err error
	switch types.ProviderName(cfg.DefaultProvider) {
	case types.ProviderOpenAI:
		_ = os.Setenv("OPENAI_API_KEY", providerConfig.APIKey)
		llmClient := openai.New().
			WithModel(openai.Model(providerConfig.Model))
		err = llmClient.Generate(context.Background(), myThread)

	case types.ProviderAnthropic:
		_ = os.Setenv("ANTHROPIC_API_KEY", providerConfig.APIKey)
		llmClient := anthropic.New().
			WithModel(providerConfig.Model)
		err = llmClient.Generate(context.Background(), myThread)

	case types.ProviderCohere:
		_ = os.Setenv("COHERE_API_KEY", providerConfig.APIKey)
		llmClient := cohere.New().
			WithModel(cohere.Model(providerConfig.Model))
		err = llmClient.Generate(context.Background(), myThread)

	case types.ProviderOllama:
		_ = os.Setenv("OLLAMA_API_KEY", providerConfig.APIKey)
		llmClient := ollama.New().
			WithModel(providerConfig.Model)
		err = llmClient.Generate(context.Background(), myThread)

	default:
		return "", fmt.Errorf("unsupported LLM provider: %s", cfg.DefaultProvider)
	}

	if err != nil {
		return "", fmt.Errorf("error generating commit message: %w", err)
	}

	// Get the last message from the thread (assistant's response)
	log.Println(myThread)
	//if len(messages) < 3 { // system + user + assistant
	//	return "", fmt.Errorf("no commit message generated")
	//}

	return "", nil
	//return strings.TrimSpace(messages[len(messages)-1].GetContent()[0].GetText()), nil
}
