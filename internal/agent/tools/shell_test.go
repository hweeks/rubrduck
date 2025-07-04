package tools

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hammie/rubrduck/internal/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipOnDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping sandboxed shell test on macOS due to sandbox-exec limitations.")
	}
}

func absTempPolicy(tempDir string) sandbox.Policy {
	policy := sandbox.DefaultPolicy()
	policy.AllowReadPaths = []string{
		tempDir,
		"/bin",
		"/usr/bin",
		"/usr/lib",
		"/System/Library",
		"/private/tmp",
		"/dev",
		"/var",
		"/sbin",
		"/usr/sbin",
		"/Library",
		"/private/var",
		"/private/etc",
		"/Applications",
		"/tmp",
	}
	policy.AllowWritePaths = []string{tempDir}
	// Add additional commands needed for testing
	policy.AllowedCommands = append(policy.AllowedCommands, "sleep")
	return policy
}

func TestShellTool_ExecuteSimpleCommand(t *testing.T) {
	skipOnDarwin(t)
	tempDir := t.TempDir()
	policy := absTempPolicy(tempDir)
	shellTool := NewShellTool(tempDir, policy)

	// Test simple command
	args := `{"command": "echo hello world"}`
	result, err := shellTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "hello world")
	assert.Contains(t, result, "Exit Code: 0")
}

func TestShellTool_ExecuteWithWorkingDir(t *testing.T) {
	skipOnDarwin(t)
	tempDir := t.TempDir()
	policy := absTempPolicy(tempDir)
	shellTool := NewShellTool(tempDir, policy)

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// Test command with working directory
	args := `{"command": "pwd", "working_dir": "subdir"}`
	result, err := shellTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "subdir")
}

func TestShellTool_CommandValidation(t *testing.T) {
	skipOnDarwin(t)
	tempDir := t.TempDir()
	policy := absTempPolicy(tempDir)
	shellTool := NewShellTool(tempDir, policy)

	// Test blocked commands
	blockedCommands := []string{
		"rm -rf /",
		"sudo ls",
		"chmod 777 /etc/passwd",
		"wget http://example.com",
		"curl http://example.com",
	}

	for _, cmd := range blockedCommands {
		args := `{"command": "` + cmd + `"}`
		_, err := shellTool.Execute(context.Background(), args)
		assert.Error(t, err, "Command should be blocked: %s", cmd)
		assert.Contains(t, err.Error(), "not allowed")
	}

	// Test dangerous patterns
	dangerousPatterns := []string{
		"ls && rm -rf /",
		"echo test; rm -rf /",
		"ls | grep test",
		"echo test > /etc/passwd",
		"echo test >> /etc/passwd",
		"eval 'rm -rf /'",
	}

	for _, cmd := range dangerousPatterns {
		args := `{"command": "` + cmd + `"}`
		_, err := shellTool.Execute(context.Background(), args)
		assert.Error(t, err, "Command should be blocked: %s", cmd)
		assert.Contains(t, err.Error(), "dangerous pattern")
	}

	// Test background execution separately
	args := `{"command": "echo test &"}`
	_, err := shellTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "background execution is not allowed")
}

func TestShellTool_Timeout(t *testing.T) {
	skipOnDarwin(t)
	tempDir := t.TempDir()
	policy := absTempPolicy(tempDir)
	shellTool := NewShellTool(tempDir, policy)

	// Test command timeout
	args := `{"command": "sleep 10", "timeout": 1}`
	_, err := shellTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command execution failed")
}

func TestShellTool_InvalidArguments(t *testing.T) {
	skipOnDarwin(t)
	tempDir := t.TempDir()
	policy := absTempPolicy(tempDir)
	shellTool := NewShellTool(tempDir, policy)

	// Test invalid JSON
	_, err := shellTool.Execute(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid arguments")

	// Test empty command
	args := `{"command": ""}`
	_, err = shellTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command cannot be empty")

	// Test invalid working directory
	args = `{"command": "ls", "working_dir": "/etc"}`
	_, err = shellTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "absolute paths are not allowed")
}

func TestShellTool_GetDefinition(t *testing.T) {
	skipOnDarwin(t)
	policy := absTempPolicy("/tmp")
	shellTool := NewShellTool("/tmp", policy)
	def := shellTool.GetDefinition()

	assert.Equal(t, "function", def.Type)
	assert.Equal(t, "shell_execute", def.Function.Name)
	assert.Contains(t, def.Function.Description, "shell commands")

	// Check parameters
	params := def.Function.Parameters["properties"].(map[string]interface{})
	assert.NotNil(t, params["command"])
	assert.NotNil(t, params["timeout"])
	assert.NotNil(t, params["working_dir"])
}

func TestShellTool_CommandOutput(t *testing.T) {
	skipOnDarwin(t)
	tempDir := t.TempDir()
	policy := absTempPolicy(tempDir)
	shellTool := NewShellTool(tempDir, policy)

	// Test command with stdout
	args := `{"command": "echo 'stdout message'"}`
	result, err := shellTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "stdout message")
	assert.Contains(t, result, "STDOUT:")

	// Test command that produces stderr without redirection
	args = `{"command": "ls /nonexistent"}`
	_, err = shellTool.Execute(context.Background(), args)
	assert.Error(t, err)
}

func TestShellTool_ErrorHandling(t *testing.T) {
	skipOnDarwin(t)
	tempDir := t.TempDir()
	policy := absTempPolicy(tempDir)
	shellTool := NewShellTool(tempDir, policy)

	// Test command that fails
	args := `{"command": "ls /nonexistent/file"}`
	_, err := shellTool.Execute(context.Background(), args)
	assert.Error(t, err) // Tool should return error for failed commands
	assert.Contains(t, err.Error(), "command failed")
}

func TestShellTool_Configuration(t *testing.T) {
	skipOnDarwin(t)
	tempDir := t.TempDir()
	policy := absTempPolicy(tempDir)
	shellTool := NewShellTool(tempDir, policy)

	// Test setting allowed commands
	newAllowed := []string{"custom1", "custom2"}
	shellTool.SetAllowedCommands(newAllowed)
	// Note: This is just testing the setter, actual validation would need to be implemented

	// Test setting blocked commands
	newBlocked := []string{"dangerous1", "dangerous2"}
	shellTool.SetBlockedCommands(newBlocked)

	// Test setting timeout
	newTimeout := 60 * time.Second
	shellTool.SetTimeout(newTimeout)
}

func TestShellTool_PathSanitization(t *testing.T) {
	skipOnDarwin(t)
	tempDir := t.TempDir()
	policy := absTempPolicy(tempDir)
	shellTool := NewShellTool(tempDir, policy)

	// Test path traversal attempt
	args := `{"command": "ls", "working_dir": "../../../etc"}`
	_, err := shellTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path outside project bounds")

	// Test absolute path
	args = `{"command": "ls", "working_dir": "/etc"}`
	_, err = shellTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "absolute paths are not allowed")
}

func TestShellTool_ContextCancellation(t *testing.T) {
	skipOnDarwin(t)
	tempDir := t.TempDir()
	policy := absTempPolicy(tempDir)
	shellTool := NewShellTool(tempDir, policy)

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	args := `{"command": "echo test"}`
	_, err := shellTool.Execute(ctx, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
