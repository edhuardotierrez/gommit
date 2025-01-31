package main

import (
	"flag"
	"os"

	"github.com/edhuardotierrez/gommit/internal/config"
	"github.com/edhuardotierrez/gommit/internal/git"
	"github.com/edhuardotierrez/gommit/internal/llm"

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

	// Add version flag
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		log.Infof("gommit version %s", version)
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

	// Create the commit
	if err := git.Commit(message); err != nil {
		log.Warnf("Error creating commit: %v", err)
		os.Exit(1)
	}

	log.Info("Successfully created commit with message:")
	log.Info("%s", message)
}
