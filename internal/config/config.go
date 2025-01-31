package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/edhuardotierrez/gommit/internal/llm"
	"github.com/manifoldco/promptui"
)

var sampleConfigMessage = `
{
	"default_provider": "openai",
	"providers": {
		"openai": {
			"api_key": "sk-proj-1234567890",
			"model": "gpt-4o-mini"
		}
	}
}
`

// ProviderConfig holds the configuration for a specific LLM provider
type ProviderConfig struct {
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

// Config holds the application configuration
type Config struct {
	DefaultProvider string                    `json:"default_provider"`
	Providers       map[string]ProviderConfig `json:"providers"`
}

// Load reads the configuration file from the user's home directory
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, "gommit.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return createConfigWizard(configPath)
		}
		return nil, fmt.Errorf("could not read config file at %s: %w\n%s", configPath, err, sampleConfigMessage)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	if config.DefaultProvider == "" {
		config.DefaultProvider = "openai"
	}

	providerConfig, ok := config.Providers[config.DefaultProvider]
	if !ok {
		return nil, fmt.Errorf("default provider %s not found in config", config.DefaultProvider)
	}

	if providerConfig.APIKey == "" {
		return nil, fmt.Errorf("api_key is required for provider %s", config.DefaultProvider)
	}

	return &config, nil
}

// createConfigWizard creates a new config file with user input
func createConfigWizard(configPath string) (*Config, error) {
	fmt.Println("\n🎉 Welcome to gommit configuration wizard! 🎉")
	fmt.Println("This wizard will help you set up your gommit configuration.\n")

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

	// Get API key for the selected provider
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

	apiKey, err := apiKeyPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("API key input failed: %w", err)
	}

	// Select model for the provider
	models := llm.GetAvailableModels(llm.ProviderName(provider))
	modelSelect := promptui.Select{
		Label: fmt.Sprintf("Select %s model", provider),
		Items: models,
	}

	_, model, err := modelSelect.Run()
	if err != nil {
		return nil, fmt.Errorf("model selection failed: %w", err)
	}

	// Create initial config
	config := &Config{
		DefaultProvider: provider,
		Providers: map[string]ProviderConfig{
			provider: {
				APIKey: apiKey,
				Model:  model,
			},
		},
	}

	// Show final configuration
	data, err := json.MarshalIndent(config, "", "    ")
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
	return config, nil
}
