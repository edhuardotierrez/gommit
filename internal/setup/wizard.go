package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/edhuardotierrez/gommit/internal/colors"
	"github.com/edhuardotierrez/gommit/internal/llm"
	"github.com/edhuardotierrez/gommit/internal/types"
	"github.com/manifoldco/promptui"
)

// CreateConfigWizard creates a new config file with user input
func CreateConfigWizard(configPath string) (*types.Config, error) {
	fmt.Println("\n🎉 Welcome to gommit configuration wizard! 🎉")
	colors.InfoOutput("The configuration file will be created at %s\n", configPath)
	colors.InfoOutput("This wizard will help you set up your gommit configuration.\n\n")

	// Ask if user wants to use the wizard
	prompt := promptui.Prompt{
		Label:     "Would you like to configure gommit using the wizard",
		IsConfirm: true,
	}

	if _, err := prompt.Run(); err != nil {
		return nil, fmt.Errorf("wizard cancelled by user")
	}

	// Select LLM provider
	providerSelect := promptui.Select{
		Label: "Select your preferred LLM provider",
		Items: func() []string {
			var items []string
			for _, p := range llm.Providers {
				items = append(items, p.Title)
			}
			return items
		}(),
	}

	_, provider, err := providerSelect.Run()
	if err != nil {
		return nil, fmt.Errorf("provider selection failed: %w", err)
	}
	// Find provider config
	var providerConfig types.ProviderTypes
	for _, p := range llm.Providers {
		if p.Title == provider {
			providerConfig = p
			break
		}
	}

	var apiKey string
	if slices.Contains(providerConfig.Required, "api_key") {
		apiKeyPrompt := promptui.Prompt{
			Label: fmt.Sprintf("Enter your %s API key", provider),
			Validate: func(input string) error {
				if len(input) < 1 {
					return fmt.Errorf("API key cannot be empty")
				}
				return nil
			},
			Mask: '*',
		}
		apiKey, err = apiKeyPrompt.Run()

		if err != nil {
			return nil, fmt.Errorf("API key input failed: %w", err)
		}
	}

	// Check if URI is required
	uri := ""
	if slices.Contains(providerConfig.Required, "uri") {
		uriPrompt := promptui.Prompt{
			Label: fmt.Sprintf("Enter %s URI", provider),
			Validate: func(input string) error {
				if len(input) < 1 {
					return fmt.Errorf("URI cannot be empty")
				}
				return nil
			},
		}
		uri, err = uriPrompt.Run()
		if err != nil {
			return nil, fmt.Errorf("URI input failed: %w", err)
		}
	}

	// Select model for the provider
	models := llm.GetAvailableModels(types.ProviderName(provider))
	modelSelect := promptui.Select{
		Label: fmt.Sprintf("Select %s model", provider),
		Items: models,
	}

	_, model, err := modelSelect.Run()
	if err != nil {
		return nil, fmt.Errorf("model selection failed: %w", err)
	}

	// Add temperature prompt
	temperaturePrompt := promptui.Prompt{
		Label:     "Enter temperature (0.0-1.0, default: 0.7, press enter to skip)",
		Default:   "0.7",
		AllowEdit: true,
		Validate: func(input string) error {
			if input == "" {
				return nil
			}
			var temp float64
			_, err := fmt.Sscanf(input, "%f", &temp)
			if err != nil || temp < 0 || temp > 1 {
				return fmt.Errorf("please enter a number between 0 and 1")
			}
			return nil
		},
	}

	temperatureStr, err := temperaturePrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("temperature input failed: %w", err)
	}

	temperature := 0.7 // default temperature
	if temperatureStr != "" {
		fmt.Sscanf(temperatureStr, "%f", &temperature)
	}

	// Select commit style
	commitStyleSelect := promptui.Select{
		Label: "Select commit message style",
		Items: []string{"conventional", "simple", "detailed"},
		Size:  3,
	}

	_, commitStyle, err := commitStyleSelect.Run()
	if err != nil {
		return nil, fmt.Errorf("commit style selection failed: %w", err)
	}

	// Ask for max tokens (optional)
	maxTokensPrompt := promptui.Prompt{
		Label:     fmt.Sprintf("Max tokens for responses (default: %d, press enter to skip)", types.DefaultMaxTokens),
		Default:   fmt.Sprintf("%d", types.DefaultMaxTokens),
		AllowEdit: true,
		Validate: func(input string) error {
			if input == "" {
				return nil
			}
			var tokens int
			_, err := fmt.Sscanf(input, "%d", &tokens)
			if err != nil || tokens < 1 {
				return fmt.Errorf("please enter a valid positive number")
			}
			return nil
		},
	}

	maxTokensStr, err := maxTokensPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("max tokens input failed: %w", err)
	}

	maxTokens := types.DefaultMaxTokens
	if maxTokensStr != "" {
		fmt.Sscanf(maxTokensStr, "%d", &maxTokens)
	}

	// Create initial config
	cfg := &types.Config{
		DefaultProvider: provider,
		Providers: map[string]types.ProviderConfig{
			provider: {
				APIKey:      apiKey,
				Model:       model,
				Temperature: temperature,
				URI:         uri,
			},
		},
		MaxTokens:   maxTokens,
		CommitStyle: commitStyle,
	}

	// Show final configuration
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("could not marshal config: %w", err)
	}

	fmt.Printf("\nConfiguration Preview:\n%s\n\n", string(data))

	// Confirm configuration
	confirmPrompt := promptui.Prompt{
		Label:     "Would you like to save this configuration",
		IsConfirm: true,
	}

	if _, err := confirmPrompt.Run(); err != nil {
		return nil, fmt.Errorf("configuration cancelled by user")
	}

	// Save configuration
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return nil, fmt.Errorf("could not write config file: %w", err)
	}

	fmt.Printf("\n✅ Configuration file created at %s\n", configPath)

	return cfg, nil
}
