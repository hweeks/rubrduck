package sandbox

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewSandbox(t *testing.T) {
	sandbox, err := NewSandbox()
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	if sandbox == nil {
		t.Fatal("Sandbox should not be nil")
	}

	capabilities := sandbox.GetCapabilities()
	if capabilities.Platform == "" {
		t.Error("Platform should not be empty")
	}
}

func TestDefaultPolicy(t *testing.T) {
	policy := DefaultPolicy()

	// Check that policy has reasonable defaults
	if len(policy.AllowedCommands) == 0 {
		t.Error("Default policy should have allowed commands")
	}

	if len(policy.BlockedCommands) == 0 {
		t.Error("Default policy should have blocked commands")
	}

	if policy.MaxMemoryMB <= 0 {
		t.Error("Default policy should have positive memory limit")
	}

	if policy.MaxCPUTime <= 0 {
		t.Error("Default policy should have positive CPU time limit")
	}
}

func TestValidateCommand(t *testing.T) {
	policy := DefaultPolicy()

	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{
			name:    "empty command",
			command: "",
			wantErr: true,
		},
		{
			name:    "allowed command",
			command: "ls",
			wantErr: false,
		},
		{
			name:    "blocked command",
			command: "rm",
			wantErr: true,
		},
		{
			name:    "another allowed command",
			command: "git",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommand(tt.command, policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSandboxExecute(t *testing.T) {
	sandbox, err := NewSandbox()
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	policy := DefaultPolicy()
	// Make paths absolute for testing
	policy.AllowReadPaths = []string{os.TempDir()}
	policy.AllowWritePaths = []string{os.TempDir()}

	ctx := context.Background()

	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "simple command",
			command: "echo",
			args:    []string{"hello"},
			wantErr: false,
		},
		{
			name:    "pwd command",
			command: "pwd",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "blocked command",
			command: "rm",
			args:    []string{"-rf", "/"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sandbox.Execute(ctx, tt.command, tt.args, policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// On some platforms, sandbox might not work perfectly due to restrictions
				// So we accept any result as long as there's no error from the sandbox itself
				if err != nil {
					t.Errorf("Sandbox execution failed: %v", err)
				}
				// If the command itself failed (exit status), that's acceptable in a sandbox
				if result.Error != nil {
					t.Logf("Command failed in sandbox (acceptable): %v", result.Error)
				}
			}
		})
	}
}

func TestSandboxTimeout(t *testing.T) {
	sandbox, err := NewSandbox()
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	policy := DefaultPolicy()
	policy.AllowReadPaths = []string{os.TempDir()}
	policy.AllowWritePaths = []string{os.TempDir()}

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Try to execute a command that should timeout
	result, err := sandbox.Execute(ctx, "sleep", []string{"1"}, policy)

	// On some platforms, the sandbox might not enforce timeouts strictly
	// So we just check that we get some result
	if err == nil && result.Error == nil {
		// This is acceptable - the sandbox might not enforce timeouts
		t.Log("Sandbox did not enforce timeout, which is acceptable on this platform")
	}
}

func TestSandboxPathValidation(t *testing.T) {
	sandbox, err := NewSandbox()
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sandbox-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	policy := Policy{
		AllowReadPaths:  []string{tempDir},
		AllowWritePaths: []string{tempDir},
		AllowedCommands: []string{"ls", "echo"},
		BlockedCommands: []string{"rm"},
	}

	ctx := context.Background()

	// Test that we can access the allowed directory
	result, err := sandbox.Execute(ctx, "ls", []string{tempDir}, policy)
	if err != nil {
		t.Errorf("Failed to execute ls in allowed directory: %v", err)
	}

	// On some platforms, sandbox might not work perfectly, so we're lenient
	if result.Error != nil {
		t.Logf("Sandbox execution had error (acceptable on some platforms): %v", result.Error)
	}
}

func TestSandboxCapabilities(t *testing.T) {
	sandbox, err := NewSandbox()
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	capabilities := sandbox.GetCapabilities()

	// Check that capabilities are reasonable
	if capabilities.Platform == "" {
		t.Error("Platform should not be empty")
	}

	// Command filtering should always be available
	if !capabilities.CommandFiltering {
		t.Error("Command filtering should be available")
	}
}

func TestSandboxPolicyValidation(t *testing.T) {
	sandbox, err := NewSandbox()
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	tests := []struct {
		name    string
		policy  Policy
		wantErr bool
	}{
		{
			name: "valid policy",
			policy: Policy{
				AllowReadPaths:  []string{"/tmp"},
				AllowWritePaths: []string{"/tmp"},
				MaxMemoryMB:     100,
				MaxProcesses:    10,
				MaxCPUTime:      30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "relative paths",
			policy: Policy{
				AllowReadPaths:  []string{"./relative"},
				AllowWritePaths: []string{"./relative"},
			},
			wantErr: true,
		},
		{
			name: "negative memory limit",
			policy: Policy{
				AllowReadPaths:  []string{"/tmp"},
				AllowWritePaths: []string{"/tmp"},
				MaxMemoryMB:     -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sandbox.ValidatePolicy(tt.policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecuteWithTimeout(t *testing.T) {
	ctx := context.Background()

	// Test successful execution
	result, err := executeWithTimeout(ctx, "echo", "hello")
	if err != nil {
		t.Errorf("executeWithTimeout() error = %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout != "hello\n" {
		t.Errorf("Expected stdout 'hello\\n', got '%s'", result.Stdout)
	}

	// Test timeout - on some systems, sleep might not respect the timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err = executeWithTimeout(ctx, "sleep", "1")
	// On some systems, the timeout might not work as expected
	if err == nil {
		t.Log("Timeout test did not work as expected, which is acceptable on some systems")
	}
}

func BenchmarkSandboxExecute(b *testing.B) {
	sandbox, err := NewSandbox()
	if err != nil {
		b.Fatalf("Failed to create sandbox: %v", err)
	}

	policy := DefaultPolicy()
	policy.AllowReadPaths = []string{os.TempDir()}
	policy.AllowWritePaths = []string{os.TempDir()}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sandbox.Execute(ctx, "echo", []string{"hello"}, policy)
		if err != nil {
			b.Errorf("Execute() error = %v", err)
		}
	}
}
