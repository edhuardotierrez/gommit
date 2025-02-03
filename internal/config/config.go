package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/edhuardotierrez/gommit/internal/setup"
	"github.com/edhuardotierrez/gommit/internal/types"
)

// GetConfigPath returns the path to the configuration file
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "gommit.json" // fallback to current directory
	}
	return filepath.Join(homeDir, "gommit.json")
}

var sampleConfigMessage = `
{
	"default_provider": "openai",
	"providers": {
		"openai": {
			"api_key": "sk-proj-1234567890",
			"model": "gpt-4o-mini",
			"temperature": 0.7
		}
	}
}
`

// ProviderConfig holds the configuration for a specific LLM provider
type ProviderConfig struct {
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature,omitempty"`
}

// Config holds the application configuration
type Config struct {
	DefaultProvider string                    `json:"default_provider"`
	Providers       map[string]ProviderConfig `json:"providers"`
}

// Load reads the configuration file from the user's home directory
func Load() (*types.Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, "gommit.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			_, _ = setup.CreateConfigWizard(configPath)
			fmt.Printf("\n🚀 You're all set! Run 'gommit' to start using gommit.\n")
			os.Exit(0)
		}
		return nil, fmt.Errorf("could not read config file at %s: %w\n%s", configPath, err, sampleConfigMessage)
	}

	var config types.Config
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

	if providerConfig.Temperature == 0 {
		providerConfig.Temperature = 0.7 // Set default temperature if not specified
	}

	if providerConfig.Temperature < 0 || providerConfig.Temperature > 1 {
		return nil, fmt.Errorf("temperature must be between 0 and 1 for provider %s", config.DefaultProvider)
	}

	config.Providers[config.DefaultProvider] = providerConfig
	return &config, nil
}
