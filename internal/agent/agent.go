package agent

import (
	"context"
	"fmt"
	"io"
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
	config         *config.Config
	provider       ai.Provider
	tools          map[string]Tool
	history        []ai.Message
	approvalSystem *ApprovalSystem
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

	// Initialize approval system
	approvalConfig := &Config{
		Mode:               cfg.Agent.ApprovalMode,
		AutoApproveLowRisk: true, // Always auto-approve low-risk operations like git status, file reads
		AutoApproveSafeCommands: append(cfg.Sandbox.AllowedCommands, []string{
			"git status", "git log", "git diff", "git show", "git branch",
			"ls", "pwd", "cat", "head", "tail", "grep", "find", "which",
		}...),
		AutoApproveSafePaths: cfg.Sandbox.AllowWritePaths,
		BlockedCommands:      cfg.Sandbox.BlockedCommands,
		BlockedPaths:         cfg.Sandbox.BlockPaths,
		MaxBatchSize:         10,
		Timeout:              time.Duration(cfg.Agent.Timeout) * time.Second,
		Policies:             make(map[string]Policy),
	}

	// Create approval callback that will be set by the TUI
	agent.approvalSystem = NewApprovalSystem(approvalConfig, nil)

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
		a.history = append(a.history, toolResults...)

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

// StreamEvents processes a user message and emits streaming events.
func (a *Agent) StreamEvents(ctx context.Context, message string) (<-chan StreamEvent, error) {
	events := make(chan StreamEvent)

	a.history = append(a.history, ai.Message{
		Role:    "user",
		Content: message,
	})

	req := &ai.ChatRequest{
		Model:    a.config.Model,
		Messages: a.history,
		Tools:    a.getToolDefinitions(),
		Stream:   true,
	}

	log.Debug().
		Str("provider", a.config.Provider).
		Str("model", a.config.Model).
		Int("message_count", len(a.history)).
		Int("tool_count", len(a.tools)).
		Str("user_message", message).
		Msg("Starting streaming chat")

	stream, err := a.provider.StreamChat(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Str("provider", a.config.Provider).
			Str("model", a.config.Model).
			Msg("Failed to start streaming chat")
		return nil, fmt.Errorf("failed to start streaming: %w", err)
	}

	go func() {
		defer stream.Close()
		defer close(events)

		assistant := ai.Message{Role: "assistant"}
		var pendingToolCalls []ai.ToolCall
		chunkCount := 0

		log.Debug().Msg("Starting to process streaming chunks")

		for {
			chunk, err := stream.Recv()
			chunkCount++

			if err != nil {
				if err == io.EOF {
					log.Debug().
						Int("total_chunks", chunkCount).
						Msg("Stream completed normally")
					break
				}
				log.Error().
					Err(err).
					Int("chunk_number", chunkCount).
					Str("provider", a.config.Provider).
					Msg("Streaming error occurred")
				events <- StreamEvent{Type: EventDone, Err: err}
				return
			}

			log.Trace().
				Int("chunk_number", chunkCount).
				Int("choices_count", len(chunk.Choices)).
				Msg("Processing chunk")

			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta

				// Handle content streaming
				if delta.Content != "" {
					log.Trace().
						Str("content", delta.Content).
						Int("content_length", len(delta.Content)).
						Msg("Received content chunk")
					events <- StreamEvent{Type: EventTokenChunk, Token: delta.Content}
					assistant.Content += delta.Content
				}

				// Handle tool call deltas - accumulate properly
				if len(delta.ToolCalls) > 0 {
					log.Debug().
						Int("tool_calls_in_delta", len(delta.ToolCalls)).
						Int("existing_tool_calls", len(pendingToolCalls)).
						Msg("Processing tool call delta")

					// Log the raw tool call data for debugging only if we have substantial content
					for i, tc := range delta.ToolCalls {
						if tc.ID != "" || tc.Function.Name != "" || len(tc.Function.Arguments) > 5 {
							log.Trace().
								Int("tool_call_index", i).
								Str("tool_call_id", tc.ID).
								Str("tool_call_type", tc.Type).
								Str("function_name", tc.Function.Name).
								Str("function_args", tc.Function.Arguments).
								Msg("Tool call delta details")
						}
					}

					pendingToolCalls = a.mergeToolCallDeltas(pendingToolCalls, delta.ToolCalls)

					// Only log after significant changes, not every character fragment
					if chunkCount%20 == 0 || len(pendingToolCalls) != len(delta.ToolCalls) {
						log.Debug().
							Int("pending_tool_calls_after_merge", len(pendingToolCalls)).
							Msg("Tool calls merged")
					}
				}
			}
		}

		// Add the assistant message to history
		assistant.ToolCalls = pendingToolCalls
		a.history = append(a.history, assistant)

		log.Info().
			Int("final_content_length", len(assistant.Content)).
			Int("final_tool_calls", len(pendingToolCalls)).
			Msg("Streaming completed, processing tool calls")

		// Process any complete tool calls
		if len(pendingToolCalls) > 0 {
			// Emit tool request events for approval
			for i, toolCall := range pendingToolCalls {
				log.Debug().
					Int("tool_call_index", i).
					Str("tool_call_id", toolCall.ID).
					Str("tool_name", toolCall.Function.Name).
					Str("tool_args", toolCall.Function.Arguments).
					Msg("Processing tool call")

				// Validate tool call before requesting approval
				if toolCall.Function.Name == "" || toolCall.Function.Arguments == "" {
					log.Warn().
						Str("tool_call_id", toolCall.ID).
						Str("function_name", toolCall.Function.Name).
						Str("arguments", toolCall.Function.Arguments).
						Msg("Skipping incomplete tool call")

					events <- StreamEvent{
						Type:   EventToolResult,
						ToolID: toolCall.ID,
						Result: fmt.Sprintf("Error: Incomplete tool call - name='%s', args='%s'", toolCall.Function.Name, toolCall.Function.Arguments),
					}
					continue
				}

				log.Debug().
					Str("tool_call_id", toolCall.ID).
					Str("tool_name", toolCall.Function.Name).
					Msg("Requesting approval for tool call")

				// Request approval for the tool execution
				approvalResult, err := a.approvalSystem.RequestApproval(ctx, toolCall.Function.Name, toolCall.Function.Arguments, toolCall)
				if err != nil {
					log.Error().
						Err(err).
						Str("tool_call_id", toolCall.ID).
						Str("tool_name", toolCall.Function.Name).
						Msg("Approval request failed")

					events <- StreamEvent{
						Type:     EventToolResult,
						ToolID:   toolCall.ID,
						ToolName: toolCall.Function.Name,
						Result:   fmt.Sprintf("Error: Approval failed for %s: %v", toolCall.Function.Name, err),
					}
					continue
				}

				if !approvalResult.Approved {
					log.Info().
						Str("tool_call_id", toolCall.ID).
						Str("tool_name", toolCall.Function.Name).
						Str("denial_reason", approvalResult.Reason).
						Msg("Tool call denied")

					events <- StreamEvent{
						Type:     EventToolResult,
						ToolID:   toolCall.ID,
						ToolName: toolCall.Function.Name,
						Result:   fmt.Sprintf("Operation denied: %s", approvalResult.Reason),
					}
					continue
				}

				log.Info().
					Str("tool_call_id", toolCall.ID).
					Str("tool_name", toolCall.Function.Name).
					Str("approval_reason", approvalResult.Reason).
					Msg("Tool call approved, executing")

				// Execute the approved tool
				tool, ok := a.tools[toolCall.Function.Name]
				if !ok {
					log.Error().
						Str("tool_call_id", toolCall.ID).
						Str("tool_name", toolCall.Function.Name).
						Strs("available_tools", func() []string {
							var names []string
							for name := range a.tools {
								names = append(names, name)
							}
							return names
						}()).
						Msg("Unknown tool requested")

					events <- StreamEvent{
						Type:     EventToolResult,
						ToolID:   toolCall.ID,
						ToolName: toolCall.Function.Name,
						Result:   fmt.Sprintf("Error: Unknown tool '%s'", toolCall.Function.Name),
					}
					continue
				}

				startTime := time.Now()
				result, err := tool.Execute(ctx, toolCall.Function.Arguments)
				duration := time.Since(startTime)

				if err != nil {
					log.Error().
						Err(err).
						Str("tool_call_id", toolCall.ID).
						Str("tool_name", toolCall.Function.Name).
						Dur("execution_duration", duration).
						Msg("Tool execution failed")

					events <- StreamEvent{
						Type:     EventToolResult,
						ToolID:   toolCall.ID,
						ToolName: toolCall.Function.Name,
						Result:   fmt.Sprintf("Error executing %s: %v", toolCall.Function.Name, err),
					}
				} else {
					log.Info().
						Str("tool_call_id", toolCall.ID).
						Str("tool_name", toolCall.Function.Name).
						Dur("execution_duration", duration).
						Int("result_length", len(result)).
						Msg("Tool execution completed successfully")

					events <- StreamEvent{
						Type:     EventToolResult,
						ToolID:   toolCall.ID,
						ToolName: toolCall.Function.Name,
						Result:   result,
					}
				}

				// Add tool result to history
				a.history = append(a.history, ai.Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: toolCall.ID,
				})
			}

			log.Debug().Msg("Getting final response after tool execution")

			// Get final response after tool execution
			// Don't include tools since we've already executed them
			req := &ai.ChatRequest{
				Model:    a.config.Model,
				Messages: a.history,
				// Tools:    a.getToolDefinitions(), // Remove tools to avoid confusion
			}
			resp, err := a.provider.Chat(ctx, req)
			if err != nil {
				log.Error().
					Err(err).
					Str("provider", a.config.Provider).
					Msg("Failed to get final response after tool execution")
				events <- StreamEvent{Type: EventDone, Err: err}
				return
			}
			if len(resp.Choices) > 0 {
				final := resp.Choices[0].Message
				a.history = append(a.history, final)

				log.Info().
					Int("final_response_length", len(final.Content)).
					Msg("Final response received")

				events <- StreamEvent{Type: EventTokenChunk, Token: final.Content}
				events <- StreamEvent{Type: EventDone, Usage: resp.Usage}
				return
			}
		}

		log.Info().Msg("Stream processing completed")
		events <- StreamEvent{Type: EventDone}
	}()

	return events, nil
}

// mergeToolCallDeltas properly accumulates tool call deltas from streaming
func (a *Agent) mergeToolCallDeltas(existing []ai.ToolCall, newDeltas []ai.ToolCall) []ai.ToolCall {
	// OpenAI sends tool call deltas with an index to indicate which tool call they belong to
	// We need to properly merge based on the position/index and streaming pattern

	for _, delta := range newDeltas {
		// Only log substantial deltas to avoid spam
		if delta.ID != "" || delta.Function.Name != "" || len(delta.Function.Arguments) > 10 {
			log.Trace().
				Str("delta_id", delta.ID).
				Str("delta_type", delta.Type).
				Str("delta_function_name", delta.Function.Name).
				Str("delta_function_args", delta.Function.Arguments).
				Int("existing_count", len(existing)).
				Msg("Processing tool call delta")
		}

		// For OpenAI streaming, we need to handle several patterns:
		// 1. First delta has ID and function name
		// 2. Subsequent deltas have empty ID/name but contain argument fragments
		// 3. We need to merge fragments into the most recent tool call

		var targetCall *ai.ToolCall
		targetIndex := -1

		// Strategy 1: Try to find exact ID match first
		if delta.ID != "" {
			for i := range existing {
				if existing[i].ID == delta.ID {
					targetCall = &existing[i]
					targetIndex = i
					break
				}
			}
		}

		// Strategy 2: If no ID match and this delta has function name, try to find by name
		if targetCall == nil && delta.Function.Name != "" {
			for i := range existing {
				if existing[i].Function.Name == delta.Function.Name && existing[i].ID != "" {
					targetCall = &existing[i]
					targetIndex = i
					break
				}
			}
		}

		// Strategy 3: If this delta has no ID/name but has arguments, it's likely a fragment
		// that should be merged into the most recent tool call
		if targetCall == nil && delta.ID == "" && delta.Function.Name == "" && delta.Function.Arguments != "" {
			if len(existing) > 0 {
				// Use the last (most recent) tool call
				targetCall = &existing[len(existing)-1]
				targetIndex = len(existing) - 1
				if len(delta.Function.Arguments) > 20 { // Only log substantial merges
					log.Trace().
						Int("target_index", targetIndex).
						Str("target_id", targetCall.ID).
						Str("fragment", delta.Function.Arguments).
						Msg("Merging argument fragment into most recent tool call")
				}
			}
		}

		if targetCall == nil {
			// This is a new tool call, append it
			log.Trace().
				Str("delta_id", delta.ID).
				Str("delta_function_name", delta.Function.Name).
				Msg("Creating new tool call from delta")
			existing = append(existing, delta)
		} else {
			// Merge the delta into the existing tool call
			if delta.ID != "" || delta.Function.Name != "" || len(delta.Function.Arguments) > 20 {
				log.Trace().
					Int("target_index", targetIndex).
					Str("existing_id", targetCall.ID).
					Str("delta_id", delta.ID).
					Str("delta_args", delta.Function.Arguments).
					Msg("Merging delta into existing tool call")
			}

			// Only update non-empty fields
			if delta.Type != "" {
				targetCall.Type = delta.Type
			}
			if delta.ID != "" {
				targetCall.ID = delta.ID
			}
			if delta.Function.Name != "" {
				targetCall.Function.Name = delta.Function.Name
			}
			if delta.Function.Arguments != "" {
				// For streaming, arguments come in fragments, so append them
				targetCall.Function.Arguments += delta.Function.Arguments

				// Warn if arguments are getting very large
				if len(targetCall.Function.Arguments) > 100000 { // 100KB
					log.Warn().
						Str("tool_id", targetCall.ID).
						Str("tool_name", targetCall.Function.Name).
						Int("args_size", len(targetCall.Function.Arguments)).
						Msg("Tool call arguments are very large - this may cause timeout issues")
				}
			}
		}
	}

	// Log final state for debugging - only log every 100 calls or when complete
	if len(existing) == 1 || len(newDeltas) == 0 {
		for i, tc := range existing {
			// Log warning for very large tool calls
			if len(tc.Function.Arguments) > 50000 { // 50KB
				log.Warn().
					Int("index", i).
					Str("id", tc.ID).
					Str("name", tc.Function.Name).
					Int("args_length", len(tc.Function.Arguments)).
					Msg("Large tool call detected - consider breaking into smaller operations")
			} else {
				log.Trace().
					Int("index", i).
					Str("id", tc.ID).
					Str("type", tc.Type).
					Str("name", tc.Function.Name).
					Int("args_length", len(tc.Function.Arguments)).
					Str("args_preview", func() string {
						if len(tc.Function.Arguments) > 50 {
							return tc.Function.Arguments[:50] + "..."
						}
						return tc.Function.Arguments
					}()).
					Msg("Final tool call state")
			}
		}
	}

	return existing
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

// SetApprovalCallback sets the approval callback for the agent
func (a *Agent) SetApprovalCallback(callback ApprovalCallback) {
	if a.approvalSystem != nil {
		a.approvalSystem.callback = callback
	}
}

// executeToolCalls executes the requested tool calls
func (a *Agent) executeToolCalls(ctx context.Context, toolCalls []ai.ToolCall) []ai.Message {
	var results []ai.Message

	// First pass: collect all approval requests
	for _, call := range toolCalls {
		tool, ok := a.tools[call.Function.Name]
		if !ok {
			results = append(results, ai.Message{
				Role:       "tool",
				Content:    fmt.Sprintf("Error: Unknown tool '%s'", call.Function.Name),
				ToolCallID: call.ID,
			})
			continue
		}

		// Request approval for the tool execution
		approvalResult, err := a.approvalSystem.RequestApproval(ctx, call.Function.Name, call.Function.Arguments, call)
		if err != nil {
			results = append(results, ai.Message{
				Role:       "tool",
				Content:    fmt.Sprintf("Error: Approval failed for %s: %v", call.Function.Name, err),
				ToolCallID: call.ID,
			})
			continue
		}

		if !approvalResult.Approved {
			results = append(results, ai.Message{
				Role:       "tool",
				Content:    fmt.Sprintf("Operation denied: %s", approvalResult.Reason),
				ToolCallID: call.ID,
			})
			continue
		}

		// Execute the approved tool
		result, err := tool.Execute(ctx, call.Function.Arguments)
		if err != nil {
			results = append(results, ai.Message{
				Role:       "tool",
				Content:    fmt.Sprintf("Error executing %s: %v", call.Function.Name, err),
				ToolCallID: call.ID,
			})
		} else {
			results = append(results, ai.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: call.ID,
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

// GetTool returns a tool by name
func (a *Agent) GetTool(name string) Tool {
	return a.tools[name]
}
