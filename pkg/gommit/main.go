package gommit

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/edhuardotierrez/gommit/internal/config"
	"github.com/edhuardotierrez/gommit/internal/git"
	"github.com/edhuardotierrez/gommit/internal/llm"
	"github.com/edhuardotierrez/gommit/internal/setup"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

var (
	version = "dev" // This will be overridden during build

	// Define colored output
	errorOutput   = color.New(color.FgRed).PrintfFunc()
	infoOutput    = color.New(color.FgCyan).PrintfFunc()
	descOutput    = color.New(color.FgHiCyan).PrintfFunc()
	textOutput    = color.New(color.FgWhite).PrintfFunc()
	successOutput = color.New(color.FgGreen).PrintfFunc()
)

func Run() {
	// Add flags
	showVersion := flag.Bool("version", false, "Show version information")
	runConfig := flag.Bool("config", false, "Run configuration wizard")

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
		errorOutput("Error: invalid argument %q\n", flag.Arg(0))
		flag.Usage()
		os.Exit(1)
	}

	if *showVersion {
		fmt.Printf("gommit version %s", version)
		return
	}

	// Run configuration wizard if requested
	if *runConfig {
		_, err := setup.CreateConfigWizard(config.GetConfigPath())
		if err != nil {
			errorOutput("Error in configuration wizard: %v\n", err)
			os.Exit(1)
		}
		successOutput("\nConfiguration completed successfully!\n\n")
		return
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		errorOutput("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Check if we're in a git repository
	if !git.IsGitRepository() {
		errorOutput("Error: not a git repository\n")
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
		errorOutput("Error getting staged changes: %v\n", err)
		os.Exit(1)
	}

	if len(changes) == 0 {
		// Get list of modified but unstaged files
		unstagedFiles, err := git.GetUnstagedChanges()
		if err != nil {
			errorOutput("Error getting unstaged changes: %v\n", err)
			os.Exit(1)
		}

		errorOutput("\n❌ No staged changes found. Use 'git add' first.\n\n")

		if len(unstagedFiles) > 0 {
			descOutput("Modified files that could be staged:\n")
			descOutput("----------------------------------\n")

			// Show up to 10 unstaged files
			maxFiles := 10
			if len(unstagedFiles) < maxFiles {
				maxFiles = len(unstagedFiles)
			}

			for i := 0; i < maxFiles; i++ {
				textOutput("  • %s (%s)\n", unstagedFiles[i].Path, unstagedFiles[i].Status)
			}

			if len(unstagedFiles) > maxFiles {
				descOutput("\nAnd %d more files...\n", len(unstagedFiles)-maxFiles)
			}

			descOutput("\nTry: git add <file> to stage specific files\n")
			descOutput("  or: git add . to stage all files\n")
		}

		os.Exit(0)
	}

	selectedProvider := cfg.Providers[cfg.DefaultProvider]

	// Generate commit message using LLM
	s.Suffix = fmt.Sprintf(" Generating commit message using AI (%s)...", selectedProvider.Model)
	s.Start()
	message, err := llm.GenerateCommitMessage(cfg, changes)
	s.Stop()
	if err != nil {
		errorOutput("Error generating commit message: %v\n", err)
		os.Exit(1)
	}

	// Preview commit message and ask for confirmation
	randIcons := []string{"✍️", "✏️", "📝", "💡", "🧠"}
	title := fmt.Sprintf("\n%s Generated commit message (%s):\n", randIcons[rand.Intn(len(randIcons))], selectedProvider.Model)
	infoOutput(title)
	infoOutput(strings.Repeat("-", len(title)) + "\n")
	fmt.Println(message)
	infoOutput("\n---------------------------------------------------------------\n")

	labelConfirmation := "✨ Would you like to proceed with this commit message"
	infoOutput(labelConfirmation)
	infoOutput(strings.Repeat("-", len(labelConfirmation)))

	prompt := promptui.Prompt{
		Label:     labelConfirmation,
		IsConfirm: true,
	}

	if _, err := prompt.Run(); err != nil {
		infoOutput("\n🚫 Commit cancelled by user\n")
		os.Exit(0)
	}

	// Create the commit
	s.Suffix = " Creating git commit..."
	s.Start()
	err = git.Commit(message)
	s.Stop()
	if err != nil {
		errorOutput("❌ Error creating commit: %v\n\n", err)
		os.Exit(1)
	}

	successOutput("\n✅ Successfully created commit!\n\n")
}
