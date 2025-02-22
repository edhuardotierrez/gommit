package git

import (
	"bufio"
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
	Path   string
	Status string
	Diff   string
}

// getTopLevelGitPath returns the absolute path of the git repository root
func getTopLevelGitPath() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.Trim(string(output), "\n")
}

// GetStagedChanges returns a list of staged changes in the repository
func GetStagedChanges() ([]StagedChange, error) {
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

	// Get working directory (based where the `gommit`` command is run from)
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting working directory: %w", err)
	}

	rootPath := getTopLevelGitPath()
	relativeRootPath := strings.TrimPrefix(workingDir, rootPath+"/")

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		path := parts[1]

		relativePath := path
		if workingDir != "" {
			relativePath = strings.TrimPrefix(path, relativeRootPath+"/")
		}

		cmd = exec.Command("git", "--no-pager", "diff", "--staged", "--cached", "--", relativePath)
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

// GetUnstagedChanges returns a list of modified but unstaged files
func GetUnstagedChanges() ([]StagedChange, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error getting unstaged changes: %w", err)
	}

	var changes []StagedChange
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 4 {
			continue
		}

		// Check for unstaged changes (M in second column)
		statusCode := line[0:2]
		if statusCode[1] == 'M' || statusCode[1] == '?' || statusCode[1] == 'D' {
			path := strings.TrimSpace(line[3:])
			status := getStatusDescription(statusCode[1])
			changes = append(changes, StagedChange{
				Path:   path,
				Status: status,
			})
		}
	}

	return changes, nil
}

func getStatusDescription(code byte) string {
	switch code {
	case 'M':
		return "modified"
	case '?':
		return "untracked"
	case 'D':
		return "deleted"
	default:
		return "unknown"
	}
}
