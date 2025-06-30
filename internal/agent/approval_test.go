package agent

import (
	"context"
	"testing"
	"time"

	"github.com/hammie/rubrduck/internal/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApprovalSystem(t *testing.T) {
	config := &Config{
		Mode:                    "suggest",
		AutoApproveLowRisk:      false,
		AutoApproveSafeCommands: []string{"ls", "cat"},
		AutoApproveSafePaths:    []string{"./safe"},
		BlockedCommands:         []string{"rm", "sudo"},
		BlockedPaths:            []string{"/etc"},
		MaxBatchSize:            5,
		Timeout:                 30 * time.Second,
		Policies:                make(map[string]Policy),
	}

	callback := func(req ApprovalRequest) (ApprovalResult, error) {
		return ApprovalResult{Approved: true, Reason: "test"}, nil
	}

	system := NewApprovalSystem(config, callback)

	assert.NotNil(t, system)
	assert.Equal(t, config, system.config)
	assert.NotNil(t, system.callback)
	assert.NotNil(t, system.pending)
	assert.NotNil(t, system.autoApprove)
}

func TestRequestApproval_AutoApprove(t *testing.T) {
	config := &Config{
		Mode:               "full-auto",
		AutoApproveLowRisk: true,
	}

	callback := func(req ApprovalRequest) (ApprovalResult, error) {
		return ApprovalResult{Approved: false, Reason: "should not be called"}, nil
	}

	system := NewApprovalSystem(config, callback)

	toolCall := ai.ToolCall{
		ID: "test-123",
		Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{
			Name:      "file_operations",
			Arguments: `{"type": "read", "path": "test.txt"}`,
		},
	}

	result, err := system.RequestApproval(context.Background(), "file_operations", toolCall.Function.Arguments, toolCall)
	require.NoError(t, err)
	assert.True(t, result.Approved)
	assert.Equal(t, "Auto-approved", result.Reason)
}

func TestRequestApproval_Blocked(t *testing.T) {
	config := &Config{
		Mode:            "suggest",
		BlockedCommands: []string{"rm"},
	}

	callback := func(req ApprovalRequest) (ApprovalResult, error) {
		return ApprovalResult{Approved: true, Reason: "should not be called"}, nil
	}

	system := NewApprovalSystem(config, callback)

	toolCall := ai.ToolCall{
		ID: "test-123",
		Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{
			Name:      "shell_execute",
			Arguments: `{"command": "rm -rf /"}`,
		},
	}

	result, err := system.RequestApproval(context.Background(), "shell_execute", toolCall.Function.Arguments, toolCall)
	require.NoError(t, err)
	assert.False(t, result.Approved)
	assert.Equal(t, "Operation blocked by policy", result.Reason)
}

func TestAnalyzeFileOperation(t *testing.T) {
	config := &Config{}
	system := NewApprovalSystem(config, nil)

	tests := []struct {
		name     string
		args     string
		expected struct {
			opType  string
			risk    RiskLevel
			preview string
		}
	}{
		{
			name: "file read",
			args: `{"type": "read", "path": "test.txt"}`,
			expected: struct {
				opType  string
				risk    RiskLevel
				preview string
			}{
				opType:  "file_read",
				risk:    RiskLow,
				preview: "Reading file: test.txt",
			},
		},
		{
			name: "file write",
			args: `{"type": "write", "path": "test.txt", "content": "hello world"}`,
			expected: struct {
				opType  string
				risk    RiskLevel
				preview string
			}{
				opType:  "file_write",
				risk:    RiskLow,
				preview: "File: test.txt\nSize: 11 bytes\nContent:\n  hello world\n",
			},
		},
		{
			name: "file list",
			args: `{"type": "list", "path": "."}`,
			expected: struct {
				opType  string
				risk    RiskLevel
				preview string
			}{
				opType:  "file_list",
				risk:    RiskLow,
				preview: "Listing directory: .",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opType, risk, preview, err := system.analyzeFileOperation(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.opType, opType)
			assert.Equal(t, tt.expected.risk, risk)
			assert.Contains(t, preview, tt.expected.preview)
		})
	}
}

func TestAnalyzeShellOperation(t *testing.T) {
	config := &Config{}
	system := NewApprovalSystem(config, nil)

	tests := []struct {
		name     string
		args     string
		expected struct {
			opType  string
			risk    RiskLevel
			preview string
		}
	}{
		{
			name: "safe command",
			args: `{"command": "ls -la"}`,
			expected: struct {
				opType  string
				risk    RiskLevel
				preview string
			}{
				opType:  "shell_execute",
				risk:    RiskLow,
				preview: "Command: ls -la",
			},
		},
		{
			name: "dangerous command",
			args: `{"command": "rm -rf /"}`,
			expected: struct {
				opType  string
				risk    RiskLevel
				preview string
			}{
				opType:  "shell_execute",
				risk:    RiskHigh,
				preview: "Command: rm -rf /",
			},
		},
		{
			name: "critical command",
			args: `{"command": "sudo rm -rf /"}`,
			expected: struct {
				opType  string
				risk    RiskLevel
				preview string
			}{
				opType:  "shell_execute",
				risk:    RiskHigh,
				preview: "Command: sudo rm -rf /",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opType, risk, preview, err := system.analyzeShellOperation(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.opType, opType)
			assert.Equal(t, tt.expected.risk, risk)
			assert.Contains(t, preview, tt.expected.preview)
		})
	}
}

func TestAnalyzeGitOperation(t *testing.T) {
	config := &Config{}
	system := NewApprovalSystem(config, nil)

	tests := []struct {
		name     string
		args     string
		expected struct {
			opType  string
			risk    RiskLevel
			preview string
		}
	}{
		{
			name: "safe git operation",
			args: `{"operation": "status"}`,
			expected: struct {
				opType  string
				risk    RiskLevel
				preview string
			}{
				opType:  "git_operation",
				risk:    RiskLow,
				preview: "Git status: ",
			},
		},
		{
			name: "medium risk git operation",
			args: `{"operation": "push"}`,
			expected: struct {
				opType  string
				risk    RiskLevel
				preview string
			}{
				opType:  "git_operation",
				risk:    RiskMedium,
				preview: "Git push: ",
			},
		},
		{
			name: "high risk git operation",
			args: `{"operation": "reset"}`,
			expected: struct {
				opType  string
				risk    RiskLevel
				preview string
			}{
				opType:  "git_operation",
				risk:    RiskHigh,
				preview: "Git reset: ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opType, risk, preview, err := system.analyzeGitOperation(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.opType, opType)
			assert.Equal(t, tt.expected.risk, risk)
			assert.Contains(t, preview, tt.expected.preview)
		})
	}
}

func TestAssessFileWriteRisk(t *testing.T) {
	config := &Config{}
	system := NewApprovalSystem(config, nil)

	tests := []struct {
		name     string
		path     string
		content  string
		expected RiskLevel
	}{
		{
			name:     "safe file",
			path:     "test.txt",
			content:  "hello world",
			expected: RiskLow,
		},
		{
			name:     "executable file",
			path:     "script.sh",
			content:  "#!/bin/bash\necho hello",
			expected: RiskHigh,
		},
		{
			name:     "system file",
			path:     "/etc/passwd",
			content:  "test",
			expected: RiskCritical,
		},
		{
			name:     "large file",
			path:     "large.txt",
			content:  string(make([]byte, 2*1024*1024)), // 2MB
			expected: RiskMedium,
		},
		{
			name:     "sensitive content",
			path:     "config.txt",
			content:  "api_key=secret123",
			expected: RiskHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := system.assessFileWriteRisk(tt.path, tt.content)
			assert.Equal(t, tt.expected, risk)
		})
	}
}

func TestAssessShellCommandRisk(t *testing.T) {
	config := &Config{}
	system := NewApprovalSystem(config, nil)

	tests := []struct {
		name     string
		command  string
		expected RiskLevel
	}{
		{
			name:     "safe command",
			command:  "ls -la",
			expected: RiskLow,
		},
		{
			name:     "dangerous command",
			command:  "rm -rf /",
			expected: RiskHigh,
		},
		{
			name:     "critical pattern",
			command:  "eval 'rm -rf /'",
			expected: RiskCritical,
		},
		{
			name:     "network command",
			command:  "curl http://example.com",
			expected: RiskHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := system.assessShellCommandRisk(tt.command)
			assert.Equal(t, tt.expected, risk)
		})
	}
}

func TestAssessGitOperationRisk(t *testing.T) {
	config := &Config{}
	system := NewApprovalSystem(config, nil)

	tests := []struct {
		name      string
		operation string
		expected  RiskLevel
	}{
		{"safe operation", "status", RiskLow},
		{"safe operation", "log", RiskLow},
		{"medium risk", "push", RiskMedium},
		{"medium risk", "pull", RiskMedium},
		{"high risk", "reset", RiskHigh},
		{"high risk", "checkout", RiskHigh},
		{"critical risk", "force", RiskCritical},
		{"critical risk", "delete", RiskCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := system.assessGitOperationRisk(tt.operation)
			assert.Equal(t, tt.expected, risk)
		})
	}
}

func TestGenerateFileWritePreview(t *testing.T) {
	config := &Config{}
	system := NewApprovalSystem(config, nil)

	preview := system.generateFileWritePreview("test.txt", "hello\nworld\n")
	assert.Contains(t, preview, "File: test.txt")
	assert.Contains(t, preview, "Size: 12 bytes")
	assert.Contains(t, preview, "hello")
	assert.Contains(t, preview, "world")
}

func TestGenerateShellCommandPreview(t *testing.T) {
	config := &Config{}
	system := NewApprovalSystem(config, nil)

	preview := system.generateShellCommandPreview("ls -la")
	assert.Contains(t, preview, "Command: ls -la")
	assert.Contains(t, preview, "Working Directory")
}

func TestGenerateGitOperationPreview(t *testing.T) {
	config := &Config{}
	system := NewApprovalSystem(config, nil)

	preview := system.generateGitOperationPreview("commit", "Add new feature")
	assert.Contains(t, preview, "Git commit: Add new feature")
}

func TestAnalyzeBatchRisk(t *testing.T) {
	config := &Config{}
	system := NewApprovalSystem(config, nil)

	tests := []struct {
		name     string
		requests []ApprovalRequest
		expected RiskLevel
	}{
		{
			name:     "empty batch",
			requests: []ApprovalRequest{},
			expected: RiskLow,
		},
		{
			name: "low risk batch",
			requests: []ApprovalRequest{
				{Risk: RiskLow},
				{Risk: RiskLow},
			},
			expected: RiskLow,
		},
		{
			name: "mixed risk batch",
			requests: []ApprovalRequest{
				{Risk: RiskLow},
				{Risk: RiskMedium},
			},
			expected: RiskMedium,
		},
		{
			name: "high risk batch",
			requests: []ApprovalRequest{
				{Risk: RiskLow},
				{Risk: RiskHigh},
			},
			expected: RiskHigh,
		},
		{
			name: "critical risk batch",
			requests: []ApprovalRequest{
				{Risk: RiskLow},
				{Risk: RiskCritical},
			},
			expected: RiskCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := system.analyzeBatchRisk(tt.requests)
			assert.Equal(t, tt.expected, risk)
		})
	}
}

func TestCanAutoApprove(t *testing.T) {
	config := &Config{
		Mode:                    "suggest",
		AutoApproveLowRisk:      true,
		AutoApproveSafeCommands: []string{"ls"},
		AutoApproveSafePaths:    []string{"./safe"},
	}

	system := NewApprovalSystem(config, nil)

	tests := []struct {
		name     string
		tool     string
		args     string
		opType   string
		risk     RiskLevel
		expected bool
	}{
		{
			name:     "full auto mode",
			tool:     "test",
			args:     "test",
			opType:   "test",
			risk:     RiskHigh,
			expected: true,
		},
		{
			name:     "low risk auto approve",
			tool:     "test",
			args:     "test",
			opType:   "test",
			risk:     RiskLow,
			expected: true,
		},
		{
			name:     "safe command",
			tool:     "shell_execute",
			args:     `{"command": "ls"}`,
			opType:   "shell_execute",
			risk:     RiskMedium,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Temporarily set mode for full-auto test
			if tt.name == "full auto mode" {
				system.config.Mode = "full-auto"
				defer func() { system.config.Mode = "suggest" }()
			}

			result := system.canAutoApprove(tt.tool, tt.args, tt.opType, tt.risk)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBlocked(t *testing.T) {
	config := &Config{
		BlockedCommands: []string{"rm"},
		BlockedPaths:    []string{"/etc"},
	}

	system := NewApprovalSystem(config, nil)

	tests := []struct {
		name     string
		tool     string
		args     string
		opType   string
		expected bool
	}{
		{
			name:     "blocked command",
			tool:     "shell_execute",
			args:     `{"command": "rm -rf /"}`,
			opType:   "shell_execute",
			expected: true,
		},
		{
			name:     "blocked path",
			tool:     "file_operations",
			args:     `{"type": "write", "path": "/etc/test"}`,
			opType:   "file_write",
			expected: true,
		},
		{
			name:     "safe operation",
			tool:     "file_operations",
			args:     `{"type": "read", "path": "test.txt"}`,
			opType:   "file_read",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := system.isBlocked(tt.tool, tt.args, tt.opType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
