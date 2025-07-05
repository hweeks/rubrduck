package agent

import (
	"context"
	"io"
	"testing"

	"github.com/hammie/rubrduck/internal/ai"
	"github.com/hammie/rubrduck/internal/config"
	"github.com/stretchr/testify/require"
)

type mockStream struct {
	chunks []*ai.ChatStreamChunk
	idx    int
}

func (m *mockStream) Recv() (*ai.ChatStreamChunk, error) {
	if m.idx >= len(m.chunks) {
		return nil, io.EOF
	}
	ch := m.chunks[m.idx]
	m.idx++
	return ch, nil
}

func (m *mockStream) Close() error { return nil }

type mockProvider struct{}

func (m *mockProvider) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	return &ai.ChatResponse{
		Choices: []ai.Choice{{Message: ai.Message{Role: "assistant", Content: "final"}}},
		Usage:   ai.Usage{TotalTokens: 1},
	}, nil
}

func (m *mockProvider) StreamChat(ctx context.Context, req *ai.ChatRequest) (ai.ChatStream, error) {
	return &mockStream{chunks: []*ai.ChatStreamChunk{
		{Choices: []ai.ChatStreamChoice{{Delta: ai.ChatStreamDelta{Content: "Hello "}}}},
		{Choices: []ai.ChatStreamChoice{{Delta: ai.ChatStreamDelta{Content: "World"}}}},
	}}, nil
}

func (m *mockProvider) GetName() string { return "mock" }

func TestAgentStreamEvents(t *testing.T) {
	ai.RegisterProvider("mock", func(cfg map[string]interface{}) (ai.Provider, error) { return &mockProvider{}, nil })

	cfg := &config.Config{
		Provider: "mock",
		Model:    "gpt-4",
		Providers: map[string]config.Provider{
			"mock": {Name: "mock"},
		},
		Agent:   config.AgentConfig{ApprovalMode: "suggest"},
		Sandbox: config.SandboxPolicy{},
	}

	ag, err := New(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	ch, err := ag.StreamEvents(ctx, "hi")
	require.NoError(t, err)

	var out string
	for ev := range ch {
		if ev.Type == EventTokenChunk {
			out += ev.Token
		}
	}

	require.Equal(t, "Hello World", out)
	require.Greater(t, len(ag.GetHistory()), 1)
}

// Test streaming with tool calls
func TestAgentStreamEventsWithToolCalls(t *testing.T) {
	// Mock provider that returns tool calls
	mockProviderWithTools := &mockProviderWithToolCalls{}

	ai.RegisterProvider("mock-tools", func(cfg map[string]interface{}) (ai.Provider, error) {
		return mockProviderWithTools, nil
	})

	cfg := &config.Config{
		Provider: "mock-tools",
		Model:    "gpt-4",
		Providers: map[string]config.Provider{
			"mock-tools": {Name: "mock-tools"},
		},
		Agent:   config.AgentConfig{ApprovalMode: "auto-edit"},
		Sandbox: config.SandboxPolicy{},
	}

	ag, err := New(cfg)
	require.NoError(t, err)

	// Set up auto-approval callback
	ag.SetApprovalCallback(func(req ApprovalRequest) (ApprovalResult, error) {
		return ApprovalResult{Approved: true, Reason: "auto-approved"}, nil
	})

	ctx := context.Background()
	ch, err := ag.StreamEvents(ctx, "list files")
	require.NoError(t, err)

	var events []StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}

	// Should have token chunks, tool results, and done event
	hasTokenChunk := false
	hasToolResult := false
	hasDone := false

	for _, ev := range events {
		switch ev.Type {
		case EventTokenChunk:
			hasTokenChunk = true
		case EventToolResult:
			hasToolResult = true
		case EventDone:
			hasDone = true
		}
	}

	require.True(t, hasTokenChunk, "Should have token chunks")
	require.True(t, hasToolResult, "Should have tool results")
	require.True(t, hasDone, "Should have done event")
}

// Mock provider that simulates streaming tool calls
type mockProviderWithToolCalls struct{}

func (m *mockProviderWithToolCalls) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	return &ai.ChatResponse{
		Choices: []ai.Choice{{Message: ai.Message{Role: "assistant", Content: "Task completed"}}},
		Usage:   ai.Usage{TotalTokens: 10},
	}, nil
}

func (m *mockProviderWithToolCalls) StreamChat(ctx context.Context, req *ai.ChatRequest) (ai.ChatStream, error) {
	return &mockStreamWithTools{chunks: []*ai.ChatStreamChunk{
		{Choices: []ai.ChatStreamChoice{{Delta: ai.ChatStreamDelta{Content: "I'll help you "}}}},
		{Choices: []ai.ChatStreamChoice{{Delta: ai.ChatStreamDelta{Content: "list the files."}}}},
		// Tool call delta chunks - simulating how OpenAI sends them
		{Choices: []ai.ChatStreamChoice{{Delta: ai.ChatStreamDelta{
			ToolCalls: []ai.ToolCall{{
				ID:   "call_123",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      "file_operations",
					Arguments: `{"type":"list","path":"."}`,
				},
			}},
		}}}},
	}}, nil
}

func (m *mockProviderWithToolCalls) GetName() string { return "mock-tools" }

type mockStreamWithTools struct {
	chunks []*ai.ChatStreamChunk
	idx    int
}

func (m *mockStreamWithTools) Recv() (*ai.ChatStreamChunk, error) {
	if m.idx >= len(m.chunks) {
		return nil, io.EOF
	}
	ch := m.chunks[m.idx]
	m.idx++
	return ch, nil
}

func (m *mockStreamWithTools) Close() error { return nil }

// Test tool call delta merging
func TestMergeToolCallDeltas(t *testing.T) {
	cfg := &config.Config{
		Provider:  "mock",
		Model:     "gpt-4",
		Providers: map[string]config.Provider{"mock": {Name: "mock"}},
		Agent:     config.AgentConfig{ApprovalMode: "suggest"},
		Sandbox:   config.SandboxPolicy{},
	}

	ag, err := New(cfg)
	require.NoError(t, err)

	// Test merging tool call deltas
	existing := []ai.ToolCall{}

	// First delta - starts a tool call
	delta1 := []ai.ToolCall{{
		ID:   "call_123",
		Type: "function",
		Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{
			Name:      "file_operations",
			Arguments: `{"type":"`,
		},
	}}

	result1 := ag.mergeToolCallDeltas(existing, delta1)
	require.Len(t, result1, 1)
	require.Equal(t, "call_123", result1[0].ID)
	require.Equal(t, "file_operations", result1[0].Function.Name)
	require.Equal(t, `{"type":"`, result1[0].Function.Arguments)

	// Second delta - continues the arguments
	delta2 := []ai.ToolCall{{
		ID: "call_123",
		Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{
			Arguments: `list","path":"."}`,
		},
	}}

	result2 := ag.mergeToolCallDeltas(result1, delta2)
	require.Len(t, result2, 1)
	require.Equal(t, `{"type":"list","path":"."}`, result2[0].Function.Arguments)
}
