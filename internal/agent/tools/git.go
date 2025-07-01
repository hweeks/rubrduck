package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/hammie/rubrduck/internal/ai"
	"github.com/rs/zerolog/log"
)

// GitTool provides Git operations
type GitTool struct {
	basePath string
	timeout  time.Duration
}

// NewGitTool creates a new git tool instance
func NewGitTool(basePath string) *GitTool {
	return &GitTool{
		basePath: basePath,
		timeout:  30 * time.Second,
	}
}

// GetDefinition returns the tool definition for the AI
func (g *GitTool) GetDefinition() ai.Tool {
	return ai.Tool{
		Type: "function",
		Function: ai.ToolFunction{
			Name:        "git_operations",
			Description: "Perform Git operations including status, diff, commit, and branch management",
			Parameters: map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"status", "diff", "commit", "branch", "log", "remote"},
					"description": "The Git operation to perform",
				},
				"args": map[string]interface{}{
					"type":        "string",
					"description": "Additional arguments for the operation (e.g., commit message, branch name)",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Specific file to operate on (for diff, status, etc.)",
				},
				"max_lines": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of lines to return (for log, diff, etc.)",
					"default":     100,
				},
			},
		},
	}
}

// Execute runs the git operation with the given arguments
func (g *GitTool) Execute(ctx context.Context, args string) (string, error) {
	var params struct {
		Operation string `json:"operation"`
		Args      string `json:"args"`
		File      string `json:"file"`
		MaxLines  int    `json:"max_lines"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Set default max lines
	if params.MaxLines == 0 {
		params.MaxLines = 100
	}

	// Validate operation
	if params.Operation == "" {
		return "", fmt.Errorf("operation is required")
	}

	log.Debug().
		Str("operation", params.Operation).
		Str("args", params.Args).
		Str("file", params.File).
		Int("max_lines", params.MaxLines).
		Msg("Executing git operation")

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	switch params.Operation {
	case "status":
		return g.gitStatus(execCtx, params.File)
	case "diff":
		return g.gitDiff(execCtx, params.File, params.MaxLines)
	case "commit":
		return g.gitCommit(execCtx, params.Args)
	case "branch":
		return g.gitBranch(execCtx, params.Args)
	case "log":
		return g.gitLog(execCtx, params.MaxLines)
	case "remote":
		return g.gitRemote(execCtx)
	default:
		return "", fmt.Errorf("unknown git operation: %s", params.Operation)
	}
}

// gitStatus shows the current git status
func (g *GitTool) gitStatus(ctx context.Context, file string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	if file != "" {
		cmd.Args = append(cmd.Args, "--", file)
	}
	cmd.Dir = g.basePath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git status failed: %w", err)
	}

	if len(output) == 0 {
		return "Working directory is clean", nil
	}

	// Parse porcelain output for better formatting
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result strings.Builder
	result.WriteString("Git Status:\n\n")

	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		status := line[:2]
		filename := line[3:]

		var statusDesc string
		switch status {
		case "M ":
			statusDesc = "Modified"
		case " M":
			statusDesc = "Modified (staged)"
		case "A ":
			statusDesc = "Added"
		case "D ":
			statusDesc = "Deleted"
		case "R ":
			statusDesc = "Renamed"
		case "C ":
			statusDesc = "Copied"
		case "??":
			statusDesc = "Untracked"
		default:
			statusDesc = "Unknown"
		}

		result.WriteString(fmt.Sprintf("%s: %s\n", statusDesc, filename))
	}

	return result.String(), nil
}

// gitDiff shows the diff of changes
func (g *GitTool) gitDiff(ctx context.Context, file string, maxLines int) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff")
	if file != "" {
		cmd.Args = append(cmd.Args, "--", file)
	}
	cmd.Dir = g.basePath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	if len(output) == 0 {
		return "No changes to show", nil
	}

	// Limit output lines
	lines := strings.Split(string(output), "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, fmt.Sprintf("\n... (showing first %d lines)", maxLines))
	}

	return strings.Join(lines, "\n"), nil
}

// gitCommit commits changes
func (g *GitTool) gitCommit(ctx context.Context, message string) (string, error) {
	if message == "" {
		return "", fmt.Errorf("commit message is required")
	}

	// First check if there are changes to commit
	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	statusCmd.Dir = g.basePath

	statusOutput, err := statusCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to check git status: %w", err)
	}

	if len(statusOutput) == 0 {
		return "No changes to commit", nil
	}

	// Add all changes
	addCmd := exec.CommandContext(ctx, "git", "add", ".")
	addCmd.Dir = g.basePath

	if err := addCmd.Run(); err != nil {
		return "", fmt.Errorf("git add failed: %w", err)
	}

	// Commit changes
	commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	commitCmd.Dir = g.basePath

	output, err := commitCmd.Output()
	if err != nil {
		return "", fmt.Errorf("git commit failed: %w", err)
	}

	return fmt.Sprintf("Successfully committed changes:\n%s", string(output)), nil
}

// gitBranch manages branches
func (g *GitTool) gitBranch(ctx context.Context, args string) (string, error) {
	if args == "" {
		// List branches
		cmd := exec.CommandContext(ctx, "git", "branch", "-a")
		cmd.Dir = g.basePath

		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git branch failed: %w", err)
		}

		trimmed := strings.TrimSpace(string(output))
		// If no branches are listed (e.g., fresh repo with no commits), get the current branch name manually
		if trimmed == "" {
			nameCmd := exec.CommandContext(ctx, "git", "symbolic-ref", "--short", "HEAD")
			nameCmd.Dir = g.basePath
			nameOut, nameErr := nameCmd.Output()
			if nameErr == nil {
				trimmed = "* " + strings.TrimSpace(string(nameOut))
			}
		}

		lines := strings.Split(trimmed, "\n")
		var result strings.Builder
		result.WriteString("Branches:\n\n")

		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			// Remove leading spaces and asterisk
			branch := strings.TrimSpace(strings.TrimPrefix(line, "* "))
			result.WriteString(fmt.Sprintf("- %s\n", branch))
		}

		return result.String(), nil
	}

	// Parse branch operation
	parts := strings.Fields(args)
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid branch operation")
	}

	switch parts[0] {
	case "create", "new":
		if len(parts) < 2 {
			return "", fmt.Errorf("branch name required")
		}
		return g.createBranch(ctx, parts[1])
	case "switch", "checkout":
		if len(parts) < 2 {
			return "", fmt.Errorf("branch name required")
		}
		return g.switchBranch(ctx, parts[1])
	case "delete":
		if len(parts) < 2 {
			return "", fmt.Errorf("branch name required")
		}
		return g.deleteBranch(ctx, parts[1])
	default:
		return "", fmt.Errorf("unknown branch operation: %s", parts[0])
	}
}

// createBranch creates a new branch
func (g *GitTool) createBranch(ctx context.Context, branchName string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "checkout", "-b", branchName)
	cmd.Dir = g.basePath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to create branch: %w", err)
	}

	return fmt.Sprintf("Successfully created and switched to branch '%s':\n%s", branchName, string(output)), nil
}

// switchBranch switches to an existing branch
func (g *GitTool) switchBranch(ctx context.Context, branchName string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "checkout", branchName)
	cmd.Dir = g.basePath

	output, err := cmd.Output()
	if err != nil {
		// If checkout failed, attempt to create the branch (useful for empty repos)
		createCmd := exec.CommandContext(ctx, "git", "checkout", "-b", branchName)
		createCmd.Dir = g.basePath
		createOut, createErr := createCmd.Output()
		if createErr != nil {
			return "", fmt.Errorf("failed to switch branch: %w", err)
		}
		return fmt.Sprintf("Successfully created and switched to branch '%s':\n%s", branchName, string(createOut)), nil
	}

	return fmt.Sprintf("Successfully switched to branch '%s':\n%s", branchName, string(output)), nil
}

// deleteBranch deletes a branch
func (g *GitTool) deleteBranch(ctx context.Context, branchName string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "-d", branchName)
	cmd.Dir = g.basePath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to delete branch: %w", err)
	}

	return fmt.Sprintf("Successfully deleted branch '%s':\n%s", branchName, string(output)), nil
}

// gitLog shows commit history
func (g *GitTool) gitLog(ctx context.Context, maxLines int) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "--oneline", "--graph", "--decorate")
	cmd.Dir = g.basePath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git log failed: %w", err)
	}

	if len(output) == 0 {
		return "No commits found", nil
	}

	// Limit output lines
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, fmt.Sprintf("\n... (showing first %d commits)", maxLines))
	}

	var result strings.Builder
	result.WriteString("Recent Commits:\n\n")
	result.WriteString(strings.Join(lines, "\n"))

	return result.String(), nil
}

// gitRemote shows remote repository information
func (g *GitTool) gitRemote(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "-v")
	cmd.Dir = g.basePath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git remote failed: %w", err)
	}

	if len(output) == 0 {
		return "No remote repositories configured", nil
	}

	var result strings.Builder
	result.WriteString("Remote Repositories:\n\n")
	result.WriteString(string(output))

	return result.String(), nil
}

// SetTimeout sets the command execution timeout
func (g *GitTool) SetTimeout(timeout time.Duration) {
	g.timeout = timeout
}
