package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IsGitRepository checks if the current directory is a git repository
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}

// StagedChange represents a staged file change
type StagedChange struct {
	Path     string
	Status   string
	Diff     string
}

// GetStagedChanges returns a list of staged changes
func GetStagedChanges() ([]StagedChange, error) {
	// Get staged files
	cmd := exec.Command("git", "diff", "--cached", "--name-status")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error getting staged files: %w", err)
	}

	var changes []StagedChange
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return changes, nil
	}

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		path := parts[1]

		// Get diff for the file
		cmd = exec.Command("git", "diff", "--cached", "--", path)
		diff, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("error getting diff for %s: %w", path, err)
		}

		changes = append(changes, StagedChange{
			Path:   path,
			Status: status,
			Diff:   string(diff),
		})
	}

	return changes, nil
}

// Commit creates a new commit with the given message
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error creating commit: %w\n%s", err, stderr.String())
	}

	return nil
} 