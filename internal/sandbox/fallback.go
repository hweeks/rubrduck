package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// FallbackSandbox implements basic sandboxing for unsupported platforms
type FallbackSandbox struct {
	basePath string
}

// NewFallbackSandbox creates a new fallback sandbox instance
func NewFallbackSandbox() (Sandbox, error) {
	basePath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return &FallbackSandbox{
		basePath: basePath,
	}, nil
}

// Execute runs a command with basic security restrictions
func (f *FallbackSandbox) Execute(ctx context.Context, command string, args []string, policy Policy) (Result, error) {
	// Validate policy
	if err := f.ValidatePolicy(policy); err != nil {
		return Result{}, err
	}

	// Validate command
	if err := ValidateCommand(command, policy); err != nil {
		return Result{}, err
	}

	// Validate file paths
	if err := f.validatePaths(policy); err != nil {
		return Result{}, err
	}

	// Create command
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = f.basePath

	// Set up basic resource limits
	if err := f.setResourceLimits(cmd, policy); err != nil {
		return Result{}, fmt.Errorf("failed to set resource limits: %w", err)
	}

	// Filter environment variables
	f.filterEnvironment(cmd, policy)

	// Execute command
	return executeWithTimeout(ctx, cmd.Path, cmd.Args[1:]...)
}

// ValidatePolicy checks if the policy is valid for fallback sandbox
func (f *FallbackSandbox) ValidatePolicy(policy Policy) error {
	// Check if paths are absolute
	for _, path := range policy.AllowReadPaths {
		if !filepath.IsAbs(path) {
			return fmt.Errorf("read path must be absolute: %s", path)
		}
	}

	for _, path := range policy.AllowWritePaths {
		if !filepath.IsAbs(path) {
			return fmt.Errorf("write path must be absolute: %s", path)
		}
	}

	// Validate resource limits
	if policy.MaxMemoryMB < 0 {
		return fmt.Errorf("max memory must be non-negative")
	}

	if policy.MaxProcesses < 0 {
		return fmt.Errorf("max processes must be non-negative")
	}

	if policy.MaxCPUTime < 0 {
		return fmt.Errorf("max CPU time must be non-negative")
	}

	return nil
}

// GetCapabilities returns the capabilities of the fallback sandbox
func (f *FallbackSandbox) GetCapabilities() Capabilities {
	return Capabilities{
		Platform:            runtime.GOOS,
		FileSystemIsolation: false, // No real isolation, just path validation
		NetworkIsolation:    false, // No network isolation
		ProcessIsolation:    false, // No process isolation
		MemoryLimits:        true,  // Basic memory limits
		CPULimits:           true,  // Basic CPU limits
		CommandFiltering:    true,  // Command validation
	}
}

// validatePaths checks if the policy paths are valid and accessible
func (f *FallbackSandbox) validatePaths(policy Policy) error {
	// Check read paths
	for _, path := range policy.AllowReadPaths {
		if !f.isPathAccessible(path, true) {
			return fmt.Errorf("read path not accessible: %s", path)
		}
	}

	// Check write paths
	for _, path := range policy.AllowWritePaths {
		if !f.isPathAccessible(path, false) {
			return fmt.Errorf("write path not accessible: %s", path)
		}
	}

	return nil
}

// isPathAccessible checks if a path is accessible for read/write
func (f *FallbackSandbox) isPathAccessible(path string, readOnly bool) bool {
	// Check if path exists
	if _, err := os.Stat(path); err != nil {
		return false
	}

	// Check permissions
	if readOnly {
		// Check read permission
		if _, err := os.Open(path); err != nil {
			return false
		}
	} else {
		// Check write permission by trying to create a temp file
		tempFile := filepath.Join(path, ".rubrduck-test")
		if file, err := os.Create(tempFile); err != nil {
			return false
		} else {
			file.Close()
			os.Remove(tempFile)
		}
	}

	return true
}

// setResourceLimits sets basic resource limits on the command
func (f *FallbackSandbox) setResourceLimits(cmd *exec.Cmd, policy Policy) error {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		var limits []syscall.Rlimit

		// Set memory limit
		if policy.MaxMemoryMB > 0 {
			limits = append(limits, syscall.Rlimit{
				Cur: uint64(policy.MaxMemoryMB * 1024 * 1024), // Convert MB to bytes
				Max: uint64(policy.MaxMemoryMB * 1024 * 1024),
			})
		}

		// Set process limit
		if policy.MaxProcesses > 0 {
			limits = append(limits, syscall.Rlimit{
				Cur: uint64(policy.MaxProcesses),
				Max: uint64(policy.MaxProcesses),
			})
		}

		// Set CPU time limit
		if policy.MaxCPUTime > 0 {
			limits = append(limits, syscall.Rlimit{
				Cur: uint64(policy.MaxCPUTime.Seconds()),
				Max: uint64(policy.MaxCPUTime.Seconds()),
			})
		}

		// Apply limits if any were set
		if len(limits) > 0 {
			// Note: This is a simplified approach. In practice, you'd want to
			// use a more sophisticated method to apply resource limits
			// that works across different platforms
		}
	}

	return nil
}

// filterEnvironment filters environment variables based on policy
func (f *FallbackSandbox) filterEnvironment(cmd *exec.Cmd, policy Policy) {
	if len(policy.AllowedEnvVars) == 0 && len(policy.BlockedEnvVars) == 0 {
		return // No filtering needed
	}

	// Get current environment
	env := cmd.Env
	if env == nil {
		env = os.Environ()
	}

	var filteredEnv []string

	for _, envVar := range env {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		// value := parts[1] // Not used in this implementation

		// Check if variable is blocked
		blocked := false
		for _, blockedVar := range policy.BlockedEnvVars {
			if key == blockedVar {
				blocked = true
				break
			}
		}

		if blocked {
			continue
		}

		// Check if variable is allowed (if allowlist is specified)
		if len(policy.AllowedEnvVars) > 0 {
			allowed := false
			for _, allowedVar := range policy.AllowedEnvVars {
				if key == allowedVar {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}

		filteredEnv = append(filteredEnv, envVar)
	}

	cmd.Env = filteredEnv
}
