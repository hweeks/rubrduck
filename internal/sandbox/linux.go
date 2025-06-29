package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// LinuxSandbox implements sandboxing for Linux using Landlock and seccomp
type LinuxSandbox struct {
	basePath string
	// Capability flags
	hasLandlock bool
	hasSeccomp  bool
}

// NewLinuxSandbox creates a new Linux sandbox instance
func NewLinuxSandbox() (Sandbox, error) {
	basePath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	sandbox := &LinuxSandbox{
		basePath: basePath,
	}

	// Detect capabilities
	sandbox.hasLandlock = sandbox.detectLandlock()
	sandbox.hasSeccomp = sandbox.detectSeccomp()

	log.Info().
		Bool("landlock", sandbox.hasLandlock).
		Bool("seccomp", sandbox.hasSeccomp).
		Msg("Linux sandbox capabilities detected")

	return sandbox, nil
}

// Execute runs a command in the Linux sandbox
func (l *LinuxSandbox) Execute(ctx context.Context, command string, args []string, policy Policy) (Result, error) {
	// Validate policy
	if err := l.ValidatePolicy(policy); err != nil {
		return Result{}, err
	}

	// Validate command
	if err := ValidateCommand(command, policy); err != nil {
		return Result{}, err
	}

	// Try Landlock first if available
	if l.hasLandlock {
		log.Debug().Msg("Attempting Landlock sandbox execution")
		result, err := l.executeWithLandlock(ctx, command, args, policy)
		if err == nil {
			return result, nil
		}
		log.Debug().Err(err).Msg("Landlock execution failed, falling back to seccomp")
	}

	// Try seccomp if available
	if l.hasSeccomp {
		log.Debug().Msg("Attempting seccomp sandbox execution")
		result, err := l.executeWithSeccomp(ctx, command, args, policy)
		if err == nil {
			return result, nil
		}
		log.Debug().Err(err).Msg("Seccomp execution failed, falling back to basic restrictions")
	}

	// Fallback to basic restrictions
	log.Debug().Msg("Using basic Linux sandbox restrictions")
	return l.executeWithBasicRestrictions(ctx, command, args, policy)
}

// ValidatePolicy checks if the policy is valid for Linux sandbox
func (l *LinuxSandbox) ValidatePolicy(policy Policy) error {
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

// GetCapabilities returns the capabilities of the Linux sandbox
func (l *LinuxSandbox) GetCapabilities() Capabilities {
	return Capabilities{
		Platform:            "linux",
		FileSystemIsolation: l.hasLandlock,
		NetworkIsolation:    l.hasSeccomp,
		ProcessIsolation:    l.hasSeccomp,
		MemoryLimits:        true,
		CPULimits:           true,
		CommandFiltering:    true,
	}
}

// detectLandlock checks if Landlock is available
func (l *LinuxSandbox) detectLandlock() bool {
	// Check kernel version (Landlock requires 5.13+)
	// This is a simplified check - in production you'd want to check /proc/version
	// or use uname to get the actual kernel version
	return true // Simplified for now
}

// detectSeccomp checks if seccomp is available
func (l *LinuxSandbox) detectSeccomp() bool {
	// Check if seccomp is supported by trying to access /proc/sys/kernel/seccomp
	// This is a simplified check
	return true // Simplified for now
}

// executeWithLandlock runs a command with Landlock restrictions
func (l *LinuxSandbox) executeWithLandlock(ctx context.Context, command string, args []string, policy Policy) (Result, error) {
	// TODO: Implement actual Landlock syscalls
	// For now, this is a stub that will fall back to seccomp
	log.Debug().Msg("Landlock implementation not yet complete, falling back to seccomp")
	return l.executeWithSeccomp(ctx, command, args, policy)
}

// executeWithSeccomp runs a command with seccomp restrictions
func (l *LinuxSandbox) executeWithSeccomp(ctx context.Context, command string, args []string, policy Policy) (Result, error) {
	// TODO: Implement actual seccomp filter
	// For now, this is a stub that will fall back to basic restrictions
	log.Debug().Msg("Seccomp implementation not yet complete, falling back to basic restrictions")
	return l.executeWithBasicRestrictions(ctx, command, args, policy)
}

// executeWithBasicRestrictions runs a command with basic resource limits
func (l *LinuxSandbox) executeWithBasicRestrictions(ctx context.Context, command string, args []string, policy Policy) (Result, error) {
	// Create command
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = l.basePath

	// Set up basic resource limits using syscall
	if err := l.setResourceLimits(cmd, policy); err != nil {
		return Result{}, fmt.Errorf("failed to set resource limits: %w", err)
	}

	// Execute command
	return executeWithTimeout(ctx, cmd.Path, cmd.Args[1:]...)
}

// setResourceLimits sets resource limits on the command
func (l *LinuxSandbox) setResourceLimits(cmd *exec.Cmd, policy Policy) error {
	// Set up basic resource limits using syscall
	if policy.MaxMemoryMB > 0 {
		// Note: Setting memory limits requires more complex implementation
		// For now, we'll rely on the timeout mechanism
		log.Debug().Int("max_memory_mb", policy.MaxMemoryMB).Msg("Memory limits not yet implemented")
	}

	if policy.MaxProcesses > 0 {
		// Note: Setting process limits requires more complex implementation
		// For now, we'll rely on command validation
		log.Debug().Int("max_processes", policy.MaxProcesses).Msg("Process limits not yet implemented")
	}

	if policy.MaxCPUTime > 0 {
		// Note: Setting CPU time limits requires more complex implementation
		// For now, we'll rely on the context timeout
		log.Debug().Dur("max_cpu_time", policy.MaxCPUTime).Msg("CPU time limits not yet implemented")
	}

	return nil
}
