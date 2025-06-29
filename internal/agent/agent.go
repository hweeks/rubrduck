package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hammie/rubrduck/internal/agent/tools"
	"github.com/hammie/rubrduck/internal/ai"
	"github.com/hammie/rubrduck/internal/config"
	"github.com/hammie/rubrduck/internal/sandbox"
	"github.com/rs/zerolog/log"
)

// Agent represents the core AI agent that handles conversations and actions
type Agent struct {
	config   *config.Config
	provider ai.Provider
	tools    map[string]Tool
	history  []ai.Message
}

// Tool represents an action the agent can perform
type Tool interface {
	// GetDefinition returns the tool definition for the AI
	GetDefinition() ai.Tool

	// Execute runs the tool with the given arguments
	Execute(ctx context.Context, args string) (string, error)
}

// New creates a new agent instance
func New(cfg *config.Config) (*Agent, error) {
	// Create AI provider
	providerConfig := make(map[string]interface{})
	if p, ok := cfg.Providers[cfg.Provider]; ok {
		providerConfig["name"] = p.Name
		providerConfig["base_url"] = p.BaseURL
		providerConfig["api_key"] = p.APIKey
	}

	provider, err := ai.GetProvider(cfg.Provider, providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI provider: %w", err)
	}

	agent := &Agent{
		config:   cfg,
		provider: provider,
		tools:    make(map[string]Tool),
		history:  []ai.Message{},
	}

	// Register default tools
	agent.registerDefaultTools()

	return agent, nil
}

// Chat processes a user message and returns the response
func (a *Agent) Chat(ctx context.Context, message string) (string, error) {
	// Add user message to history
	a.history = append(a.history, ai.Message{
		Role:    "user",
		Content: message,
	})

	// Prepare chat request
	req := &ai.ChatRequest{
		Model:    a.config.Model,
		Messages: a.history,
		Tools:    a.getToolDefinitions(),
	}

	// Send request to AI provider
	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to get AI response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	// Get the assistant's message
	assistantMsg := resp.Choices[0].Message
	a.history = append(a.history, assistantMsg)

	// Check if the AI wants to use tools
	if len(assistantMsg.ToolCalls) > 0 {
		// Execute tool calls
		toolResults := a.executeToolCalls(ctx, assistantMsg.ToolCalls)

		// Add tool results to history
		for _, result := range toolResults {
			a.history = append(a.history, result)
		}

		// Get final response after tool execution
		req.Messages = a.history
		resp, err = a.provider.Chat(ctx, req)
		if err != nil {
			return "", fmt.Errorf("failed to get AI response after tool execution: %w", err)
		}

		if len(resp.Choices) > 0 {
			finalMsg := resp.Choices[0].Message
			a.history = append(a.history, finalMsg)
			return finalMsg.Content, nil
		}
	}

	return assistantMsg.Content, nil
}

// StreamChat processes a user message and streams the response
func (a *Agent) StreamChat(ctx context.Context, message string, callback func(chunk string)) error {
	// Add user message to history
	a.history = append(a.history, ai.Message{
		Role:    "user",
		Content: message,
	})

	// Prepare chat request
	req := &ai.ChatRequest{
		Model:    a.config.Model,
		Messages: a.history,
		Tools:    a.getToolDefinitions(),
		Stream:   true,
	}

	// Send request to AI provider
	stream, err := a.provider.StreamChat(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start streaming: %w", err)
	}
	defer stream.Close()

	var fullResponse strings.Builder
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("streaming error: %w", err)
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			fullResponse.WriteString(content)
			callback(content)
		}
	}

	// Add complete response to history
	a.history = append(a.history, ai.Message{
		Role:    "assistant",
		Content: fullResponse.String(),
	})

	return nil
}

// RegisterTool adds a tool to the agent
func (a *Agent) RegisterTool(name string, tool Tool) {
	a.tools[name] = tool
}

// registerDefaultTools registers the built-in tools
func (a *Agent) registerDefaultTools() {
	// Use current working directory as base path
	basePath := "."

	// Register file operations tool
	fileTool := tools.NewFileTool(basePath)
	a.RegisterTool("file_operations", fileTool)

	// Register shell execution tool
	shellPolicy := sandbox.Policy{
		AllowReadPaths:  a.config.Sandbox.AllowReadPaths,
		AllowWritePaths: a.config.Sandbox.AllowWritePaths,
		BlockPaths:      a.config.Sandbox.BlockPaths,
		AllowNetwork:    a.config.Sandbox.AllowNetwork,
		AllowedHosts:    a.config.Sandbox.AllowedHosts,
		MaxProcesses:    a.config.Sandbox.MaxProcesses,
		MaxMemoryMB:     a.config.Sandbox.MaxMemoryMB,
		MaxCPUTime:      time.Duration(a.config.Sandbox.MaxCPUTime) * time.Second,
		AllowedCommands: a.config.Sandbox.AllowedCommands,
		BlockedCommands: a.config.Sandbox.BlockedCommands,
		AllowedEnvVars:  a.config.Sandbox.AllowedEnvVars,
		BlockedEnvVars:  a.config.Sandbox.BlockedEnvVars,
	}
	shellTool := tools.NewShellTool(basePath, shellPolicy)
	a.RegisterTool("shell_execute", shellTool)

	// Register git operations tool
	gitTool := tools.NewGitTool(basePath)
	a.RegisterTool("git_operations", gitTool)

	log.Debug().Msg("Registered default tools: file_operations, shell_execute, git_operations")
}

// getToolDefinitions returns tool definitions for the AI
func (a *Agent) getToolDefinitions() []ai.Tool {
	var tools []ai.Tool
	for _, tool := range a.tools {
		tools = append(tools, tool.GetDefinition())
	}
	return tools
}

// executeToolCalls executes the requested tool calls
func (a *Agent) executeToolCalls(ctx context.Context, toolCalls []ai.ToolCall) []ai.Message {
	var results []ai.Message

	for _, call := range toolCalls {
		tool, ok := a.tools[call.Function.Name]
		if !ok {
			results = append(results, ai.Message{
				Role:    "tool",
				Content: fmt.Sprintf("Error: Unknown tool '%s'", call.Function.Name),
				Name:    call.ID,
			})
			continue
		}

		// Check approval mode
		if a.config.Agent.ApprovalMode == "suggest" {
			// In suggest mode, we would need user approval here
			// For now, we'll just log it
			log.Info().
				Str("tool", call.Function.Name).
				Str("args", call.Function.Arguments).
				Msg("Tool execution requires approval")
		}

		// Execute the tool
		result, err := tool.Execute(ctx, call.Function.Arguments)
		if err != nil {
			results = append(results, ai.Message{
				Role:    "tool",
				Content: fmt.Sprintf("Error executing %s: %v", call.Function.Name, err),
				Name:    call.ID,
			})
		} else {
			results = append(results, ai.Message{
				Role:    "tool",
				Content: result,
				Name:    call.ID,
			})
		}
	}

	return results
}

// ClearHistory clears the conversation history
func (a *Agent) ClearHistory() {
	a.history = []ai.Message{}
}

// GetHistory returns the current conversation history
func (a *Agent) GetHistory() []ai.Message {
	return a.history
}
