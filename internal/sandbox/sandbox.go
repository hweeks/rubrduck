package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Policy defines the security policy for sandboxed execution
type Policy struct {
	// File system access
	AllowReadPaths  []string `json:"allow_read_paths"`
	AllowWritePaths []string `json:"allow_write_paths"`
	BlockPaths      []string `json:"block_paths"`

	// Network access
	AllowNetwork bool     `json:"allow_network"`
	AllowedHosts []string `json:"allowed_hosts"`

	// Process restrictions
	MaxProcesses int           `json:"max_processes"`
	MaxMemoryMB  int           `json:"max_memory_mb"`
	MaxCPUTime   time.Duration `json:"max_cpu_time"`

	// Command restrictions
	AllowedCommands []string `json:"allowed_commands"`
	BlockedCommands []string `json:"blocked_commands"`

	// Environment
	AllowedEnvVars []string `json:"allowed_env_vars"`
	BlockedEnvVars []string `json:"blocked_env_vars"`
}

// Result represents the result of a sandboxed execution
type Result struct {
	ExitCode   int           `json:"exit_code"`
	Stdout     string        `json:"stdout"`
	Stderr     string        `json:"stderr"`
	Duration   time.Duration `json:"duration"`
	MemoryUsed int64         `json:"memory_used"`
	Error      error         `json:"error,omitempty"`
}

// Sandbox defines the interface for sandbox implementations
type Sandbox interface {
	// Execute runs a command in the sandbox
	Execute(ctx context.Context, command string, args []string, policy Policy) (Result, error)

	// ValidatePolicy checks if a policy is valid for this sandbox
	ValidatePolicy(policy Policy) error

	// GetCapabilities returns the capabilities of this sandbox
	GetCapabilities() Capabilities
}

// Capabilities describes what a sandbox implementation can do
type Capabilities struct {
	Platform            string `json:"platform"`
	FileSystemIsolation bool   `json:"file_system_isolation"`
	NetworkIsolation    bool   `json:"network_isolation"`
	ProcessIsolation    bool   `json:"process_isolation"`
	MemoryLimits        bool   `json:"memory_limits"`
	CPULimits           bool   `json:"cpu_limits"`
	CommandFiltering    bool   `json:"command_filtering"`
}

// NewSandbox creates a new sandbox instance for the current platform
func NewSandbox() (Sandbox, error) {
	switch runtime.GOOS {
	case "darwin":
		return NewDarwinSandbox()
	case "linux":
		return NewLinuxSandbox()
	default:
		return NewFallbackSandbox()
	}
}

// DefaultPolicy returns a safe default policy
func DefaultPolicy() Policy {
	return Policy{
		AllowReadPaths:  []string{"./"},
		AllowWritePaths: []string{"./"},
		BlockPaths:      []string{"/etc", "/var", "/usr", "/bin", "/sbin", "/System"},
		AllowNetwork:    false,
		MaxProcesses:    10,
		MaxMemoryMB:     512,
		MaxCPUTime:      30 * time.Second,
		AllowedCommands: []string{
			"ls", "cat", "head", "tail", "grep", "find", "wc", "sort", "uniq",
			"echo", "pwd", "whoami", "date", "ps", "git", "go", "npm", "yarn", "python", "node", "make",
		},
		BlockedCommands: []string{
			"rm", "rmdir", "del", "format", "mkfs", "dd", "shred",
			"sudo", "su", "chmod", "chown", "passwd", "useradd",
			"wget", "curl", "nc", "netcat", "ssh", "scp", "rsync",
		},
		AllowedEnvVars: []string{"PATH", "HOME", "USER", "PWD", "LANG", "LC_ALL"},
		BlockedEnvVars: []string{"SUDO_ASKPASS", "SSH_AUTH_SOCK", "GPG_AGENT_INFO"},
	}
}

// ValidateCommand checks if a command is allowed according to the policy
func ValidateCommand(command string, policy Policy) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Check blocked commands
	for _, blocked := range policy.BlockedCommands {
		if command == blocked {
			return fmt.Errorf("command '%s' is blocked by policy", blocked)
		}
	}

	// If allowed commands are specified, check if command is in the list
	if len(policy.AllowedCommands) > 0 {
		allowed := false
		for _, allowedCmd := range policy.AllowedCommands {
			if command == allowedCmd {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("command '%s' is not in allowed commands list", command)
		}
	}

	return nil
}

// executeWithTimeout runs a command with a timeout
func executeWithTimeout(ctx context.Context, name string, args ...string) (Result, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	result := Result{
		ExitCode: cmd.ProcessState.ExitCode(),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	if err != nil {
		result.Error = err
	}

	return result, nil
}
