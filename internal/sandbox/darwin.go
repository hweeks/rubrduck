package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// DarwinSandbox implements sandboxing for macOS using sandbox-exec
type DarwinSandbox struct {
	basePath string
}

// NewDarwinSandbox creates a new macOS sandbox instance
func NewDarwinSandbox() (Sandbox, error) {
	basePath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return &DarwinSandbox{
		basePath: basePath,
	}, nil
}

// Execute runs a command in the macOS sandbox
func (d *DarwinSandbox) Execute(ctx context.Context, command string, args []string, policy Policy) (Result, error) {
	// Validate policy
	if err := d.ValidatePolicy(policy); err != nil {
		return Result{}, err
	}

	// Validate command
	if err := ValidateCommand(command, policy); err != nil {
		return Result{}, err
	}

	// Create sandbox profile
	profile, err := d.createSandboxProfile(policy)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create sandbox profile: %w", err)
	}

	// Write profile to temporary file
	profileFile, err := os.CreateTemp("", "rubrduck-sandbox-*.sb")
	if err != nil {
		return Result{}, fmt.Errorf("failed to create sandbox profile file: %w", err)
	}
	defer os.Remove(profileFile.Name())

	if _, err := profileFile.WriteString(profile); err != nil {
		profileFile.Close()
		return Result{}, fmt.Errorf("failed to write sandbox profile: %w", err)
	}
	profileFile.Close()

	// Build sandbox-exec command
	sandboxArgs := []string{"-f", profileFile.Name(), command}
	sandboxArgs = append(sandboxArgs, args...)

	// Execute with sandbox-exec
	return executeWithTimeout(ctx, "sandbox-exec", sandboxArgs...)
}

// ValidatePolicy checks if the policy is valid for macOS sandbox
func (d *DarwinSandbox) ValidatePolicy(policy Policy) error {
	// Check if sandbox-exec is available
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		return fmt.Errorf("sandbox-exec not available: %w", err)
	}

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

// GetCapabilities returns the capabilities of the macOS sandbox
func (d *DarwinSandbox) GetCapabilities() Capabilities {
	return Capabilities{
		Platform:            "darwin",
		FileSystemIsolation: true,
		NetworkIsolation:    true,
		ProcessIsolation:    true,
		MemoryLimits:        false, // macOS sandbox doesn't provide memory limits
		CPULimits:           false, // macOS sandbox doesn't provide CPU limits
		CommandFiltering:    true,
	}
}

// createSandboxProfile generates a sandbox profile for sandbox-exec
func (d *DarwinSandbox) createSandboxProfile(policy Policy) (string, error) {
	const profileTemplate = `(version 1)
(deny default)

; Allow file system access for read paths
{{range .AllowReadPaths}}
(allow file-read* (subpath "{{.}}"))
{{end}}

; Allow file system access for write paths
{{range .AllowWritePaths}}
(allow file-write* (subpath "{{.}}"))
{{end}}

; Block access to sensitive paths
{{range .BlockPaths}}
(deny file* (subpath "{{.}}"))
{{end}}

; Network access
{{if .AllowNetwork}}
(allow network*)
{{else}}
(deny network*)
{{end}}

; Allow environment variables
{{range .AllowedEnvVars}}
(allow environment-variable "{{.}}")
{{end}}

; Block environment variables
{{range .BlockedEnvVars}}
(deny environment-variable "{{.}}")
{{end}}
`
	tmpl, err := template.New("sandbox").Parse(profileTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse sandbox template: %w", err)
	}

	var buf strings.Builder
	data := struct {
		Policy
	}{
		Policy: policy,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute sandbox template: %w", err)
	}

	return buf.String(), nil
}
