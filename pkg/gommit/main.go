package gommit

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/edhuardotierrez/gommit/internal/globals"

	"github.com/edhuardotierrez/gommit/internal/colors"
	"github.com/edhuardotierrez/gommit/internal/config"
	"github.com/edhuardotierrez/gommit/internal/git"
	"github.com/edhuardotierrez/gommit/internal/llm"
	"github.com/edhuardotierrez/gommit/internal/setup"

	"github.com/briandowns/spinner"
	"github.com/manifoldco/promptui"
)

var (
	version = "dev" // This will be overridden during build
)

func Run() {
	// Add flags
	showVersion := flag.Bool("version", false, "Show version information")
	runConfig := flag.Bool("config", false, "Run configuration wizard")
	showVerbose := flag.Bool("verbose", false, "Show verbose output")

	// optional
	runWithProvider := flag.String("p", "", "Run with a specific provider (optional)")
	runWithModel := flag.String("m", "", "Run with a specific model (optional)")
	runWithTemperature := flag.String("t", "", "Run with a specific temperature (optional)")
	runWithStyle := flag.String("s", "", "Run with a specific commit style (optional)")
	runWithTruncateLines := flag.Int("l", 0, "Run with a specific number of truncate lines (optional)")
	runWithMaxLineWidth := flag.Int("w", 0, "Run with a specific max line width (optional)")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of gommit:\n")
		fmt.Fprintf(os.Stderr, "  gommit [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}

	// Parse and validate flags
	flag.Parse()

	// Check for invalid flags
	if flag.NArg() > 0 {
		colors.ErrorOutput("Error: invalid argument %q\n", flag.Arg(0))
		flag.Usage()
		os.Exit(1)
	}

	if *showVersion {
		fmt.Printf("gommit version %s", version)
		return
	}

	if *showVerbose {
		globals.VerboseMode = true
	}

	// Run configuration wizard if requested
	if *runConfig {
		_, err := setup.CreateConfigWizard(config.GetConfigPath())
		if err != nil {
			colors.ErrorOutput("Error in configuration wizard: %v\n", err)
			os.Exit(1)
		}
		colors.SuccessOutput("\nConfiguration completed successfully!\n\n")
		return
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		colors.ErrorOutput("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Check if we're in a git repository
	if !git.IsGitRepository() {
		colors.ErrorOutput("Error: not a git repository\n")
		os.Exit(1)
	}

	// Get staged changes
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " Analyzing git changes..."
	_ = s.Color("cyan")
	s.Start()

	changes, err := git.GetStagedChanges()
	s.Stop()
	if err != nil {
		colors.ErrorOutput("Error getting staged changes: %v\n", err)
		os.Exit(1)
	}

	if len(changes) == 0 {
		// Get list of modified but unstaged files
		unstagedFiles, err := git.GetUnstagedChanges()
		if err != nil {
			colors.ErrorOutput("Error getting unstaged changes: %v\n", err)
			os.Exit(1)
		}

		colors.ErrorOutput("\n❌ No staged changes found. Use 'git add' first.\n\n")

		if len(unstagedFiles) > 0 {
			colors.DescOutput("Modified files that could be staged:\n")
			colors.DescOutput("----------------------------------\n")

			// Show up to 10 unstaged files
			maxFiles := 10
			if len(unstagedFiles) < maxFiles {
				maxFiles = len(unstagedFiles)
			}

			for i := 0; i < maxFiles; i++ {
				colors.TextOutput("  • %s (%s)\n", unstagedFiles[i].Path, unstagedFiles[i].Status)
			}

			if len(unstagedFiles) > maxFiles {
				colors.DescOutput("\nAnd %d more files...\n", len(unstagedFiles)-maxFiles)
			}

			colors.DescOutput("\nTry: git add <file> to stage specific files\n")
			colors.DescOutput("  or: git add . to stage all files\n")
		}

		os.Exit(0)
	}

	var provider = cfg.DefaultProvider
	var overrides []string

	if *runWithProvider != "" {
		provider = *runWithProvider
		overrides = append(overrides, provider)
	}

	selectedConfig := cfg.Providers[provider]

	// Add model and temperature if provided
	if *runWithModel != "" {
		selectedConfig.Model = *runWithModel
		overrides = append(overrides, fmt.Sprintf("model(%s)", *runWithModel))
	}

	// if flagTemperature is not 0, set the temperature
	runWithTemperatureFloat, err := strconv.ParseFloat(*runWithTemperature, 64)
	if err == nil && runWithTemperatureFloat >= 0.0 {
		selectedConfig.Temperature = runWithTemperatureFloat
		overrides = append(overrides, fmt.Sprintf("temperature(%.2f)", runWithTemperatureFloat))
	}

	if *runWithStyle != "" {
		selectedConfig.CommitStyle = *runWithStyle
		overrides = append(overrides, fmt.Sprintf("style(%s)", *runWithStyle))
	}

	if *runWithTruncateLines > 0 {
		cfg.TruncateLines = *runWithTruncateLines
		overrides = append(overrides, fmt.Sprintf("truncate_lines(%d)", *runWithTruncateLines))
	}

	if *runWithMaxLineWidth > 0 {
		cfg.MaxLineWidth = *runWithMaxLineWidth
		overrides = append(overrides, fmt.Sprintf("max_line_width(%d)", *runWithMaxLineWidth))
	}

	if len(overrides) > 0 {
		colors.WarningOutput("⚠️ Overriding configuration: %s\n\n", strings.Join(overrides, ", "))
	}

	// Generate commit message using LLM
	s.Suffix = fmt.Sprintf(" Generating commit message using AI (%s)...", selectedConfig.Model)
	s.Start()
	message, err := llm.GenerateCommitMessage(cfg, changes, provider, selectedConfig)
	s.Stop()
	if err != nil {
		colors.ErrorOutput("Error generating commit message: %v\n", err)
		os.Exit(1)
	}

	// Preview commit message and ask for confirmation
	randIcons := []string{"✍️", "✏️", "📝", "💡", "🧠"}
	title := fmt.Sprintf("\n%s Generated commit message (%s):\n", randIcons[rand.Intn(len(randIcons))], selectedConfig.Model)
	colors.InfoOutput(title)
	colors.InfoOutput(strings.Repeat("-", len(title)) + "\n")
	fmt.Println(message)
	colors.InfoOutput("\n---------------------------------------------------------------\n")

	labelConfirmation := "✨ Would you like to proceed with this commit message"
	colors.InfoOutput(labelConfirmation)
	colors.InfoOutput(strings.Repeat("-", len(labelConfirmation)))

	prompt := promptui.Prompt{
		Label:     labelConfirmation,
		IsConfirm: true,
	}

	if _, err := prompt.Run(); err != nil {
		colors.InfoOutput("\n🚫 Commit cancelled by user\n")
		os.Exit(0)
	}

	// Create the commit
	s.Suffix = " Creating git commit..."
	s.Start()
	err = git.Commit(message)
	s.Stop()
	if err != nil {
		colors.ErrorOutput("❌ Error creating commit: %v\n\n", err)
		os.Exit(1)
	}

	colors.SuccessOutput("\n✅ Successfully created commit!\n\n")
}
