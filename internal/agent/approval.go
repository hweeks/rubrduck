package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hammie/rubrduck/internal/ai"
	"github.com/rs/zerolog/log"
)

// ApprovalRequest represents a request for user approval
type ApprovalRequest struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "file_write", "shell_execute", "git_operation"
	Tool        string                 `json:"tool"`
	Arguments   string                 `json:"arguments"`
	Description string                 `json:"description"`
	Risk        RiskLevel              `json:"risk"`
	Preview     string                 `json:"preview"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

// RiskLevel represents the risk level of an operation
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// ApprovalResult represents the result of an approval request
type ApprovalResult struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

// ApprovalCallback is a function that handles approval requests
type ApprovalCallback func(req ApprovalRequest) (ApprovalResult, error)

// ApprovalSystem handles approval requests and policy enforcement
type ApprovalSystem struct {
	config      *Config
	callback    ApprovalCallback
	pending     map[string]ApprovalRequest
	autoApprove map[string]bool
}

// Config represents approval system configuration
type Config struct {
	Mode                    string            `json:"mode"` // "suggest", "auto-edit", "full-auto"
	AutoApproveLowRisk      bool              `json:"auto_approve_low_risk"`
	AutoApproveSafeCommands []string          `json:"auto_approve_safe_commands"`
	AutoApproveSafePaths    []string          `json:"auto_approve_safe_paths"`
	BlockedCommands         []string          `json:"blocked_commands"`
	BlockedPaths            []string          `json:"blocked_paths"`
	MaxBatchSize            int               `json:"max_batch_size"`
	Timeout                 time.Duration     `json:"timeout"`
	Policies                map[string]Policy `json:"policies"`
}

// Policy represents a specific approval policy
type Policy struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	AutoApprove bool      `json:"auto_approve"`
	AllowedOps  []string  `json:"allowed_ops"`
	BlockedOps  []string  `json:"blocked_ops"`
	RiskLevel   RiskLevel `json:"risk_level"`
}

// NewApprovalSystem creates a new approval system
func NewApprovalSystem(config *Config, callback ApprovalCallback) *ApprovalSystem {
	return &ApprovalSystem{
		config:      config,
		callback:    callback,
		pending:     make(map[string]ApprovalRequest),
		autoApprove: make(map[string]bool),
	}
}

// RequestApproval requests approval for a tool execution
func (a *ApprovalSystem) RequestApproval(ctx context.Context, tool string, args string, toolCall ai.ToolCall) (ApprovalResult, error) {
	// Parse arguments to determine operation type and risk
	opType, risk, preview, err := a.analyzeOperation(tool, args)
	if err != nil {
		return ApprovalResult{Approved: false, Reason: fmt.Sprintf("Failed to analyze operation: %v", err)}, nil
	}

	// Check if operation is blocked by policy
	if a.isBlocked(tool, args, opType) {
		return ApprovalResult{Approved: false, Reason: "Operation blocked by policy"}, nil
	}

	// Check if operation can be auto-approved
	if a.canAutoApprove(tool, args, opType, risk) {
		log.Info().
			Str("tool", tool).
			Str("operation", opType).
			Str("risk", string(risk)).
			Msg("Auto-approving operation")
		return ApprovalResult{Approved: true, Reason: "Auto-approved"}, nil
	}

	// Create approval request
	req := ApprovalRequest{
		ID:          toolCall.ID,
		Type:        opType,
		Tool:        tool,
		Arguments:   args,
		Description: a.generateDescription(tool, args, opType),
		Risk:        risk,
		Preview:     preview,
		Metadata:    a.extractMetadata(tool, args),
		CreatedAt:   time.Now(),
	}

	// Store pending request
	a.pending[req.ID] = req

	// Request user approval
	if a.callback != nil {
		result, err := a.callback(req)
		if err != nil {
			delete(a.pending, req.ID)
			return ApprovalResult{Approved: false, Reason: fmt.Sprintf("Approval failed: %v", err)}, err
		}

		// Clean up pending request
		delete(a.pending, req.ID)

		return result, nil
	}

	// No callback available, default to requiring approval
	return ApprovalResult{Approved: false, Reason: "No approval handler available"}, nil
}

// RequestBatchApproval requests approval for multiple operations
func (a *ApprovalSystem) RequestBatchApproval(ctx context.Context, requests []ApprovalRequest) ([]ApprovalResult, error) {
	if len(requests) == 0 {
		return []ApprovalResult{}, nil
	}

	// Check batch size limit
	if len(requests) > a.config.MaxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds maximum %d", len(requests), a.config.MaxBatchSize)
	}

	// Analyze batch risk
	batchRisk := a.analyzeBatchRisk(requests)

	// Check if entire batch can be auto-approved
	if a.canAutoApproveBatch(requests, batchRisk) {
		results := make([]ApprovalResult, len(requests))
		for i := range results {
			results[i] = ApprovalResult{Approved: true, Reason: "Batch auto-approved"}
		}
		return results, nil
	}

	// Request batch approval
	if a.callback != nil {
		// Create a batch request
		batchReq := ApprovalRequest{
			ID:          fmt.Sprintf("batch_%d", time.Now().Unix()),
			Type:        "batch",
			Tool:        "batch_operations",
			Arguments:   fmt.Sprintf("%d operations", len(requests)),
			Description: a.generateBatchDescription(requests),
			Risk:        batchRisk,
			Preview:     a.generateBatchPreview(requests),
			Metadata: map[string]interface{}{
				"operations": requests,
				"count":      len(requests),
			},
			CreatedAt: time.Now(),
		}

		result, err := a.callback(batchReq)
		if err != nil {
			return nil, err
		}

		// Apply batch result to all operations
		results := make([]ApprovalResult, len(requests))
		for i := range results {
			results[i] = result
		}

		return results, nil
	}

	// Default to requiring approval for all
	results := make([]ApprovalResult, len(requests))
	for i := range results {
		results[i] = ApprovalResult{Approved: false, Reason: "No batch approval handler available"}
	}

	return results, nil
}

// analyzeOperation analyzes a tool operation to determine type, risk, and generate preview
func (a *ApprovalSystem) analyzeOperation(tool, args string) (opType string, risk RiskLevel, preview string, err error) {
	switch tool {
	case "file_operations":
		return a.analyzeFileOperation(args)
	case "shell_execute":
		return a.analyzeShellOperation(args)
	case "git_operations":
		return a.analyzeGitOperation(args)
	default:
		return "unknown", RiskHigh, "Unknown operation type", nil
	}
}

// analyzeFileOperation analyzes file operations
func (a *ApprovalSystem) analyzeFileOperation(args string) (opType string, risk RiskLevel, preview string, err error) {
	var params struct {
		Type    string `json:"type"`
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", RiskHigh, "", err
	}

	switch params.Type {
	case "read":
		return "file_read", RiskLow, fmt.Sprintf("Reading file: %s", params.Path), nil
	case "write":
		risk = a.assessFileWriteRisk(params.Path, params.Content)
		preview = a.generateFileWritePreview(params.Path, params.Content)
		return "file_write", risk, preview, nil
	case "list":
		return "file_list", RiskLow, fmt.Sprintf("Listing directory: %s", params.Path), nil
	case "search":
		return "file_search", RiskLow, fmt.Sprintf("Searching in: %s", params.Path), nil
	default:
		return "file_unknown", RiskMedium, "Unknown file operation", nil
	}
}

// analyzeShellOperation analyzes shell operations
func (a *ApprovalSystem) analyzeShellOperation(args string) (opType string, risk RiskLevel, preview string, err error) {
	var params struct {
		Command string `json:"command"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", RiskHigh, "", err
	}

	risk = a.assessShellCommandRisk(params.Command)
	preview = a.generateShellCommandPreview(params.Command)
	return "shell_execute", risk, preview, nil
}

// analyzeGitOperation analyzes git operations
func (a *ApprovalSystem) analyzeGitOperation(args string) (opType string, risk RiskLevel, preview string, err error) {
	var params struct {
		Operation string `json:"operation"`
		Args      string `json:"args"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", RiskHigh, "", err
	}

	risk = a.assessGitOperationRisk(params.Operation)
	preview = a.generateGitOperationPreview(params.Operation, params.Args)
	return "git_operation", risk, preview, nil
}

// assessFileWriteRisk assesses the risk of a file write operation
func (a *ApprovalSystem) assessFileWriteRisk(path, content string) RiskLevel {
	// Check for dangerous file extensions
	dangerousExts := []string{".exe", ".sh", ".bat", ".cmd", ".ps1", ".py", ".js", ".php"}
	for _, ext := range dangerousExts {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return RiskHigh
		}
	}

	// Check for system files
	systemPaths := []string{"/etc/", "/var/", "/usr/", "/bin/", "/sbin/", "/System/"}
	for _, sysPath := range systemPaths {
		if strings.HasPrefix(path, sysPath) {
			return RiskCritical
		}
	}

	// Check for large files
	if len(content) > 1024*1024 { // 1MB
		return RiskMedium
	}

	// Check for sensitive patterns in content
	sensitivePatterns := []string{
		"password", "secret", "key", "token", "credential",
		"api_key", "private_key", "ssh_key",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(strings.ToLower(content), pattern) {
			return RiskHigh
		}
	}

	return RiskLow
}

// assessShellCommandRisk assesses the risk of a shell command
func (a *ApprovalSystem) assessShellCommandRisk(command string) RiskLevel {
	// Check for critical patterns first (highest priority)
	criticalPatterns := []string{
		"eval", "exec",
	}

	for _, pattern := range criticalPatterns {
		if strings.Contains(command, pattern) {
			return RiskCritical
		}
	}

	// Check for dangerous patterns
	dangerousPatterns := []string{
		"&&", "||", ";", "|", ">", "<", ">>", "<<", "2>", "&>",
		"$((", "`", "source", ".",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return RiskCritical
		}
	}

	// Check for redirection attempts
	if strings.Contains(command, ">") || strings.Contains(command, "<") {
		return RiskCritical
	}

	// Check for background execution
	if strings.Contains(command, "&") {
		return RiskCritical
	}

	// Check for dangerous commands
	dangerousCommands := []string{
		"rm", "rmdir", "del", "format", "mkfs", "dd", "shred",
		"sudo", "su", "chmod", "chown", "passwd", "useradd",
		"wget", "curl", "nc", "netcat", "ssh", "scp", "rsync",
	}

	for _, cmd := range dangerousCommands {
		if strings.Contains(command, cmd) {
			return RiskHigh
		}
	}

	return RiskLow
}

// assessGitOperationRisk assesses the risk of a git operation
func (a *ApprovalSystem) assessGitOperationRisk(operation string) RiskLevel {
	switch operation {
	case "commit", "add", "status", "log", "diff", "show":
		return RiskLow
	case "push", "pull", "fetch":
		return RiskMedium
	case "reset", "revert", "checkout", "branch", "merge":
		return RiskHigh
	case "force", "delete", "prune":
		return RiskCritical
	default:
		return RiskMedium
	}
}

// generateFileWritePreview generates a preview for file write operations
func (a *ApprovalSystem) generateFileWritePreview(path, content string) string {
	var preview strings.Builder
	preview.WriteString(fmt.Sprintf("File: %s\n", path))
	preview.WriteString(fmt.Sprintf("Size: %d bytes\n", len(content)))

	// Show first few lines
	lines := strings.Split(content, "\n")
	if len(lines) > 10 {
		preview.WriteString("Preview (first 10 lines):\n")
		for i := 0; i < 10; i++ {
			preview.WriteString(fmt.Sprintf("  %s\n", lines[i]))
		}
		preview.WriteString(fmt.Sprintf("  ... and %d more lines\n", len(lines)-10))
	} else {
		preview.WriteString("Content:\n")
		for _, line := range lines {
			preview.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}

	return preview.String()
}

// generateShellCommandPreview generates a preview for shell commands
func (a *ApprovalSystem) generateShellCommandPreview(command string) string {
	return fmt.Sprintf("Command: %s\nWorking Directory: Current project directory", command)
}

// generateGitOperationPreview generates a preview for git operations
func (a *ApprovalSystem) generateGitOperationPreview(operation, args string) string {
	return fmt.Sprintf("Git %s: %s", operation, args)
}

// generateDescription generates a human-readable description of the operation
func (a *ApprovalSystem) generateDescription(tool, args, opType string) string {
	switch opType {
	case "file_write":
		var params struct {
			Path string `json:"path"`
		}
		if json.Unmarshal([]byte(args), &params) == nil {
			return fmt.Sprintf("Write file: %s", params.Path)
		}
	case "shell_execute":
		var params struct {
			Command string `json:"command"`
		}
		if json.Unmarshal([]byte(args), &params) == nil {
			return fmt.Sprintf("Execute command: %s", params.Command)
		}
	case "git_operation":
		var params struct {
			Operation string `json:"operation"`
		}
		if json.Unmarshal([]byte(args), &params) == nil {
			return fmt.Sprintf("Git %s", params.Operation)
		}
	}
	return fmt.Sprintf("%s operation", opType)
}

// generateBatchDescription generates a description for batch operations
func (a *ApprovalSystem) generateBatchDescription(requests []ApprovalRequest) string {
	if len(requests) == 0 {
		return "No operations"
	}

	var description strings.Builder
	description.WriteString(fmt.Sprintf("Batch of %d operations:\n", len(requests)))

	for i, req := range requests {
		description.WriteString(fmt.Sprintf("  %d. %s\n", i+1, req.Description))
	}

	return description.String()
}

// generateBatchPreview generates a preview for batch operations
func (a *ApprovalSystem) generateBatchPreview(requests []ApprovalRequest) string {
	var preview strings.Builder
	preview.WriteString("Operations to be executed:\n\n")

	for i, req := range requests {
		preview.WriteString(fmt.Sprintf("%d. %s (%s risk)\n", i+1, req.Description, req.Risk))
		preview.WriteString(fmt.Sprintf("   %s\n\n", req.Preview))
	}

	return preview.String()
}

// extractMetadata extracts metadata from tool arguments
func (a *ApprovalSystem) extractMetadata(tool, args string) map[string]interface{} {
	metadata := make(map[string]interface{})

	switch tool {
	case "file_operations":
		var params struct {
			Type string `json:"type"`
			Path string `json:"path"`
		}
		if json.Unmarshal([]byte(args), &params) == nil {
			metadata["file_type"] = params.Type
			metadata["file_path"] = params.Path
		}
	case "shell_execute":
		var params struct {
			Command string `json:"command"`
		}
		if json.Unmarshal([]byte(args), &params) == nil {
			metadata["command"] = params.Command
		}
	case "git_operations":
		var params struct {
			Operation string `json:"operation"`
		}
		if json.Unmarshal([]byte(args), &params) == nil {
			metadata["git_operation"] = params.Operation
		}
	}

	return metadata
}

// isBlocked checks if an operation is blocked by policy
func (a *ApprovalSystem) isBlocked(tool, args, opType string) bool {
	// Check blocked commands
	for _, blocked := range a.config.BlockedCommands {
		if strings.Contains(args, blocked) {
			return true
		}
	}

	// Check blocked paths
	for _, blocked := range a.config.BlockedPaths {
		if strings.Contains(args, blocked) {
			return true
		}
	}

	return false
}

// canAutoApprove checks if an operation can be auto-approved
func (a *ApprovalSystem) canAutoApprove(tool, args, opType string, risk RiskLevel) bool {
	// Check approval mode
	if a.config.Mode == "full-auto" {
		return true
	}

	// Auto-approve low risk operations if configured
	if a.config.AutoApproveLowRisk && risk == RiskLow {
		return true
	}

	// Check safe commands
	for _, safe := range a.config.AutoApproveSafeCommands {
		if strings.Contains(args, safe) {
			return true
		}
	}

	// Check safe paths
	for _, safe := range a.config.AutoApproveSafePaths {
		if strings.Contains(args, safe) {
			return true
		}
	}

	return false
}

// canAutoApproveBatch checks if a batch can be auto-approved
func (a *ApprovalSystem) canAutoApproveBatch(requests []ApprovalRequest, batchRisk RiskLevel) bool {
	// Check approval mode
	if a.config.Mode == "full-auto" {
		return true
	}

	// Auto-approve low risk batches
	if a.config.AutoApproveLowRisk && batchRisk == RiskLow {
		return true
	}

	// Check if all operations in batch can be auto-approved
	for _, req := range requests {
		if !a.canAutoApprove(req.Tool, req.Arguments, req.Type, req.Risk) {
			return false
		}
	}

	return true
}

// analyzeBatchRisk analyzes the overall risk of a batch operation
func (a *ApprovalSystem) analyzeBatchRisk(requests []ApprovalRequest) RiskLevel {
	if len(requests) == 0 {
		return RiskLow
	}

	// Find the highest risk level in the batch
	highestRisk := RiskLow
	for _, req := range requests {
		switch req.Risk {
		case RiskCritical:
			return RiskCritical
		case RiskHigh:
			highestRisk = RiskHigh
		case RiskMedium:
			if highestRisk == RiskLow {
				highestRisk = RiskMedium
			}
		}
	}

	return highestRisk
}

// GetPendingRequests returns all pending approval requests
func (a *ApprovalSystem) GetPendingRequests() []ApprovalRequest {
	requests := make([]ApprovalRequest, 0, len(a.pending))
	for _, req := range a.pending {
		requests = append(requests, req)
	}
	return requests
}

// ClearPendingRequests clears all pending requests
func (a *ApprovalSystem) ClearPendingRequests() {
	a.pending = make(map[string]ApprovalRequest)
}
