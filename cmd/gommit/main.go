package main

import (
	"flag"
	"os"

	"github.com/edhuardotierrez/gommit/internal/config"
	"github.com/edhuardotierrez/gommit/internal/git"
	"github.com/edhuardotierrez/gommit/internal/llm"
	"github.com/edhuardotierrez/gommit/internal/setup"

	"github.com/manifoldco/promptui"
	log "github.com/sirupsen/logrus"
)

var (
	version = "dev" // This will be overridden during build
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	// Add flags
	showVersion := flag.Bool("version", false, "Show version information")
	runConfig := flag.Bool("config", false, "Run configuration wizard")
	flag.Parse()

	if *showVersion {
		log.Infof("gommit version %s", version)
		return
	}

	// Run configuration wizard if requested
	if *runConfig {
		_, err := setup.CreateConfigWizard(config.GetConfigPath())
		if err != nil {
			log.Warnf("Error in configuration wizard: %v", err)
			os.Exit(1)
		}
		log.Info("Configuration completed successfully!")
		return
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Warnf("Error loading configuration: %v", err)
		os.Exit(1)
	}

	// Check if we're in a git repository
	if !git.IsGitRepository() {
		log.Warnf("Error: not a git repository")
		os.Exit(1)
	}

	// Get staged changes
	changes, err := git.GetStagedChanges()
	if err != nil {
		log.Warnf("Error getting staged changes: %v", err)
		os.Exit(1)
	}

	if len(changes) == 0 {
		log.Warnf("No staged changes found. Use 'git add' first.")
		os.Exit(1)
	}

	// Generate commit message using LLM
	message, err := llm.GenerateCommitMessage(cfg, changes)
	if err != nil {
		log.Warnf("Error generating commit message: %v", err)
		os.Exit(1)
	}

	// Preview commit message and ask for confirmation
	log.Info("Generated commit message:")
	log.Info("------------------------")
	log.Info(message)
	log.Info("------------------------")

	prompt := promptui.Prompt{
		Label:     "Would you like to proceed with this commit message",
		IsConfirm: true,
	}

	if _, err := prompt.Run(); err != nil {
		log.Info("Commit cancelled by user")
		os.Exit(0)
	}

	// Create the commit
	if err := git.Commit(message); err != nil {
		log.Warnf("Error creating commit: %v", err)
		os.Exit(1)
	}

	log.Info("Successfully created commit!")
}
