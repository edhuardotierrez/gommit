package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"

	"github.com/edhuardotierrez/gommit/internal/colors"
	"github.com/edhuardotierrez/gommit/internal/llm"
	"github.com/edhuardotierrez/gommit/internal/types"
)

// --- helpers: configuration IO ---

func readConfigFromPath(configPath string) (*types.Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}
	var cfg types.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config: %w", err)
	}
	return &cfg, nil
}

func writeConfigToPath(configPath string, cfg *types.Config) error {
	out, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, out, 0600); err != nil {
		return fmt.Errorf("could not write config: %w", err)
	}
	return nil
}

// --- helpers: providers ---

func sortedProviderTitles() []string {
	titles := make([]string, 0, len(llm.Providers))
	for _, p := range llm.Providers {
		titles = append(titles, p.Title)
	}
	slices.Sort(titles)
	return titles
}

func providerDisplayList(cfg *types.Config, titles []string) []string {
	display := make([]string, 0, len(titles))
	for _, t := range titles {
		markers := make([]string, 0, 2)
		if _, ok := cfg.Providers[t]; ok {
			markers = append(markers, "configured")
		}
		if cfg.DefaultProvider == t {
			markers = append(markers, "default")
		}
		if len(markers) > 0 {
			display = append(display, fmt.Sprintf("%s [%s]", t, strings.Join(markers, "][")))
		} else {
			display = append(display, t)
		}
	}
	return display
}

func findProviderMetaByTitle(title string) (types.ProviderTypes, bool) {
	for _, p := range llm.Providers {
		if p.Title == title {
			return p, true
		}
	}
	return types.ProviderTypes{}, false
}

// --- helpers: prompts ---

func selectIndex(label string, items []string) (int, error) {
	s := promptui.Select{Label: label, Items: items, Size: len(items)}
	idx, _, err := s.Run()
	if err != nil {
		return 0, err
	}
	return idx, nil
}

func chooseModelForProvider(providerTitle, current string) (string, bool, error) {
	models := llm.GetAvailableModels(types.ProviderName(providerTitle))
	if len(models) == 0 {
		return current, false, nil
	}
	display := make([]string, 0, len(models)+1)
	display = append(display, fmt.Sprintf("(keep current) %s", current))
	display = append(display, models...)
	s := promptui.Select{Label: "Select model", Items: display}
	_, choice, err := s.Run()
	if err != nil {
		return current, false, err
	}
	if strings.HasPrefix(choice, "(keep current)") {
		return current, false, nil
	}
	return choice, true, nil
}

// --- helpers: editor ---

func ensureConfigPresenceWithDefaults(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		empty := &types.Config{DefaultProvider: "openai", Providers: map[string]types.ProviderConfig{"openai": {Model: "gpt-4o-mini", Temperature: 0.7}}}
		data, _ := json.MarshalIndent(empty, "", "    ")
		if writeErr := os.WriteFile(configPath, data, 0600); writeErr != nil {
			return fmt.Errorf("could not create default config: %w", writeErr)
		}
	}
	return nil
}

func resolveEditorCommand() (string, []string, error) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	candidates := []string{}
	if editor != "" {
		candidates = append(candidates, editor)
	}
	candidates = append(candidates, "nvim", "vim", "vi", "nano")
	for _, c := range candidates {
		parts := strings.Fields(c)
		bin := parts[0]
		if _, err := exec.LookPath(bin); err == nil {
			return parts[0], parts[1:], nil
		}
	}
	return "", nil, fmt.Errorf("no editor found; set $VISUAL or $EDITOR, or install vim/nano")
}

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

	temperature := 1.0 // default temperature
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

// EditConfigInEditor opens the config file in the user's editor (VISUAL/EDITOR) or falls back to common editors.
func EditConfigInEditor(configPath string) error {
	// Ensure file exists
	if err := ensureConfigPresenceWithDefaults(configPath); err != nil {
		return err
	}

	cmdName, args, err := resolveEditorCommand()
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdName, append(args, configPath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// EditProviderWizard lets the user choose a provider from the config and edit fields.
func EditProviderWizard(configPath string) error {
	// Load existing config
	cfg, err := readConfigFromPath(configPath)
	if err != nil {
		return err
	}

	if len(cfg.Providers) == 0 {
		return fmt.Errorf("no providers found in config")
	}

	// Build full list from available providers and highlight already configured/default ones
	providerTitles := sortedProviderTitles()
	orderedDisplay := providerDisplayList(cfg, providerTitles)

	idx, err := selectIndex("Select provider to edit", orderedDisplay)
	if err != nil {
		return fmt.Errorf("provider selection failed: %w", err)
	}

	selected := providerTitles[idx]
	pc := cfg.Providers[selected]

	// Find provider meta
	providerMeta, _ := findProviderMetaByTitle(selected)

	// API Key (masked). Leave empty to keep unchanged.
	apiKeyPrompt := promptui.Prompt{
		Label:     "New api_key (leave blank to keep)",
		Mask:      '*',
		AllowEdit: true,
	}
	if newKey, keyErr := apiKeyPrompt.Run(); keyErr == nil {
		if strings.TrimSpace(newKey) != "" {
			pc.APIKey = newKey
		}
	} else if !errors.Is(keyErr, promptui.ErrInterrupt) {
		return fmt.Errorf("api_key input failed: %w", keyErr)
	}

	// Model
	if model, changed, err := chooseModelForProvider(selected, pc.Model); err != nil {
		return fmt.Errorf("model selection failed: %w", err)
	} else if changed {
		pc.Model = model
	}

	// Temperature
	tempPrompt := promptui.Prompt{
		Label:     fmt.Sprintf("New temperature (0-1, current: %.2f, blank to keep)", pc.Temperature),
		AllowEdit: true,
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return nil
			}
			v, err := strconv.ParseFloat(input, 64)
			if err != nil || v < 0 || v > 1 {
				return fmt.Errorf("enter number between 0 and 1")
			}
			return nil
		},
	}
	if newTempStr, tErr := tempPrompt.Run(); tErr == nil {
		if strings.TrimSpace(newTempStr) != "" {
			v, _ := strconv.ParseFloat(newTempStr, 64)
			pc.Temperature = v
		}
	} else if !errors.Is(tErr, promptui.ErrInterrupt) {
		return fmt.Errorf("temperature input failed: %w", tErr)
	}

	// URI (optional or required by provider)
	requiresURI := slices.Contains(providerMeta.Required, "uri")
	if pc.URI != "" || requiresURI {
		uriPrompt := promptui.Prompt{
			Label:     fmt.Sprintf("New uri (current: %s, blank to keep)", pc.URI),
			AllowEdit: true,
		}
		if newURI, uErr := uriPrompt.Run(); uErr == nil {
			if strings.TrimSpace(newURI) != "" {
				pc.URI = newURI
			}
		} else if !errors.Is(uErr, promptui.ErrInterrupt) {
			return fmt.Errorf("uri input failed: %w", uErr)
		}
	}

	// Save back
	cfg.Providers[selected] = pc
	if err := writeConfigToPath(configPath, cfg); err != nil {
		return err
	}

	colors.SuccessOutput("Updated provider '%s'\n", selected)
	return nil
}

// EditDefaultsWizard lets the user change non-provider settings: default_provider, max_tokens, commit_style, max_line_width.
func EditDefaultsWizard(configPath string) error {
	// Load existing config
	cfg, err := readConfigFromPath(configPath)
	if err != nil {
		return err
	}

	// Default provider selection from available providers
	providerTitles := sortedProviderTitles()
	display := make([]string, 0, len(providerTitles))
	for _, name := range providerTitles {
		if name == cfg.DefaultProvider {
			display = append(display, fmt.Sprintf("%s [current]", name))
		} else {
			display = append(display, name)
		}
	}
	idx, err := selectIndex("Select default provider", display)
	if err != nil {
		return fmt.Errorf("default provider selection failed: %w", err)
	}
	cfg.DefaultProvider = providerTitles[idx]

	// Commit style
	styles := []string{"conventional", "simple", "detailed"}
	styleDisplay := make([]string, 0, len(styles))
	for _, s := range styles {
		if s == cfg.CommitStyle {
			styleDisplay = append(styleDisplay, fmt.Sprintf("%s [current]", s))
		} else {
			styleDisplay = append(styleDisplay, s)
		}
	}
	sidx, err := selectIndex("Select commit style", styleDisplay)
	if err != nil {
		return fmt.Errorf("commit style selection failed: %w", err)
	}
	cfg.CommitStyle = styles[sidx]

	// Max tokens
	maxTokensPrompt := promptui.Prompt{
		Label:     fmt.Sprintf("Max tokens (current: %d, blank to keep)", cfg.MaxTokens),
		AllowEdit: true,
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return nil
			}
			v, err := strconv.Atoi(strings.TrimSpace(input))
			if err != nil || v < 1 {
				return fmt.Errorf("enter a positive integer")
			}
			return nil
		},
	}
	if val, e := maxTokensPrompt.Run(); e == nil {
		if strings.TrimSpace(val) != "" {
			v, _ := strconv.Atoi(strings.TrimSpace(val))
			cfg.MaxTokens = v
		}
	} else if !errors.Is(e, promptui.ErrInterrupt) {
		return fmt.Errorf("max tokens input failed: %w", e)
	}

	// Max line width
	maxWidthPrompt := promptui.Prompt{
		Label:     fmt.Sprintf("Max line width (current: %d, blank to keep)", cfg.MaxLineWidth),
		AllowEdit: true,
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return nil
			}
			v, err := strconv.Atoi(strings.TrimSpace(input))
			if err != nil || v < 20 {
				return fmt.Errorf("enter an integer >= 20")
			}
			return nil
		},
	}
	if val, e := maxWidthPrompt.Run(); e == nil {
		if strings.TrimSpace(val) != "" {
			v, _ := strconv.Atoi(strings.TrimSpace(val))
			cfg.MaxLineWidth = v
		}
	} else if !errors.Is(e, promptui.ErrInterrupt) {
		return fmt.Errorf("max line width input failed: %w", e)
	}

	// Save back
	return writeConfigToPath(configPath, cfg)
}
