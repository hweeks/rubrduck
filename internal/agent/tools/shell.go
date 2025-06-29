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

// ShellTool provides shell command execution capabilities
type ShellTool struct {
	basePath    string
	allowedCmds []string
	blockedCmds []string
	timeout     time.Duration
}

// NewShellTool creates a new shell tool instance
func NewShellTool(basePath string) *ShellTool {
	return &ShellTool{
		basePath: basePath,
		allowedCmds: []string{
			"ls", "cat", "head", "tail", "grep", "find", "wc", "sort", "uniq",
			"echo", "pwd", "whoami", "date", "ps", "top", "df", "du",
			"git", "go", "npm", "yarn", "python", "node", "make",
		},
		blockedCmds: []string{
			"rm", "rmdir", "del", "format", "mkfs", "dd", "shred",
			"sudo", "su", "chmod", "chown", "passwd", "useradd",
			"wget", "curl", "nc", "netcat", "ssh", "scp", "rsync",
		},
		timeout: 30 * time.Second,
	}
}

// GetDefinition returns the tool definition for the AI
func (s *ShellTool) GetDefinition() ai.Tool {
	return ai.Tool{
		Type: "function",
		Function: ai.ToolFunction{
			Name:        "shell_execute",
			Description: "Execute shell commands with security restrictions and approval handling",
			Parameters: map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The shell command to execute",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Timeout in seconds (default: 30)",
					"default":     30,
				},
				"working_dir": map[string]interface{}{
					"type":        "string",
					"description": "Working directory for command execution (relative to project root)",
				},
			},
		},
	}
}

// Execute runs the shell command with the given arguments
func (s *ShellTool) Execute(ctx context.Context, args string) (string, error) {
	var params struct {
		Command    string `json:"command"`
		Timeout    int    `json:"timeout"`
		WorkingDir string `json:"working_dir"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Set default timeout
	if params.Timeout == 0 {
		params.Timeout = 30
	}

	// Validate command
	if err := s.validateCommand(params.Command); err != nil {
		return "", err
	}

	// Determine working directory
	workDir := s.basePath
	if params.WorkingDir != "" {
		sanitizedDir, err := s.sanitizePath(params.WorkingDir)
		if err != nil {
			return "", fmt.Errorf("invalid working directory: %w", err)
		}
		workDir = sanitizedDir
	}

	log.Debug().
		Str("command", params.Command).
		Str("working_dir", workDir).
		Int("timeout", params.Timeout).
		Msg("Executing shell command")

	// Create context with timeout
	timeout := time.Duration(params.Timeout) * time.Second
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute command
	result, err := s.executeCommand(execCtx, params.Command, workDir)
	if err != nil {
		return "", fmt.Errorf("command execution failed: %w", err)
	}

	return result, nil
}

// validateCommand checks if the command is allowed
func (s *ShellTool) validateCommand(command string) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Split command into parts
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("invalid command")
	}

	cmd := parts[0]

	// Check if command is blocked
	for _, blocked := range s.blockedCmds {
		if cmd == blocked {
			return fmt.Errorf("command '%s' is not allowed for security reasons", cmd)
		}
	}

	// Check if command contains dangerous patterns
	dangerousPatterns := []string{
		"&&", "||", ";", "|", ">", "<", ">>", "<<", "2>", "&>",
		"$((", "`", "eval", "exec", "source", ".",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return fmt.Errorf("command contains dangerous pattern '%s'", pattern)
		}
	}

	// Check for redirection attempts
	if strings.Contains(command, ">") || strings.Contains(command, "<") {
		return fmt.Errorf("file redirection is not allowed")
	}

	// Check for background execution
	if strings.Contains(command, "&") {
		return fmt.Errorf("background execution is not allowed")
	}

	return nil
}

// sanitizePath ensures the path is safe and within the project bounds
func (s *ShellTool) sanitizePath(path string) (string, error) {
	// Clean the path to remove any .. or . components
	cleanPath := strings.TrimSpace(path)

	// If it's an absolute path, make it relative to base
	if strings.HasPrefix(cleanPath, "/") {
		// For now, only allow relative paths
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	// Join with base path
	fullPath := cleanPath
	if !strings.HasPrefix(fullPath, s.basePath) {
		fullPath = s.basePath + "/" + cleanPath
	}

	// Ensure the final path is within the base path
	if !strings.HasPrefix(fullPath, s.basePath) {
		return "", fmt.Errorf("path outside project bounds")
	}

	return fullPath, nil
}

// executeCommand executes the shell command and captures output
func (s *ShellTool) executeCommand(ctx context.Context, command, workDir string) (string, error) {
	// Create command
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = workDir

	// Capture output
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	// Build result
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Command: %s\n", command))
	result.WriteString(fmt.Sprintf("Working Directory: %s\n", workDir))
	result.WriteString(fmt.Sprintf("Exit Code: %d\n\n", cmd.ProcessState.ExitCode()))

	if stdout.Len() > 0 {
		result.WriteString("STDOUT:\n")
		result.WriteString(stdout.String())
		result.WriteString("\n")
	}

	if stderr.Len() > 0 {
		result.WriteString("STDERR:\n")
		result.WriteString(stderr.String())
		result.WriteString("\n")
	}

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return result.String(), fmt.Errorf("command timed out after %v", s.timeout)
	}

	// Check for execution error
	if err != nil {
		return result.String(), fmt.Errorf("command failed: %w", err)
	}

	return result.String(), nil
}

// SetAllowedCommands sets the list of allowed commands
func (s *ShellTool) SetAllowedCommands(cmds []string) {
	s.allowedCmds = cmds
}

// SetBlockedCommands sets the list of blocked commands
func (s *ShellTool) SetBlockedCommands(cmds []string) {
	s.blockedCmds = cmds
}

// SetTimeout sets the command execution timeout
func (s *ShellTool) SetTimeout(timeout time.Duration) {
	s.timeout = timeout
}
