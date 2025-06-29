package ai

import (
	"context"
	"io"
	"testing"
)

func TestProviderInterface(t *testing.T) {
	// This test ensures the interface is properly defined
	var _ Provider = (*MockProvider)(nil)
}

func TestChatRequest(t *testing.T) {
	req := &ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "user", Content: "Hello, world!"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	if req.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", req.Model)
	}

	if len(req.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(req.Messages))
	}

	if req.Messages[0].Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", req.Messages[0].Role)
	}

	if req.Messages[0].Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", req.Messages[0].Content)
	}
}

func TestMessageWithToolCalls(t *testing.T) {
	msg := Message{
		Role:    "assistant",
		Content: "I'll help you with that.",
		ToolCalls: []ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      "read_file",
					Arguments: `{"path": "test.txt"}`,
				},
			},
		},
	}

	if len(msg.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(msg.ToolCalls))
	}

	if msg.ToolCalls[0].ID != "call_123" {
		t.Errorf("Expected tool call ID 'call_123', got '%s'", msg.ToolCalls[0].ID)
	}

	if msg.ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("Expected function name 'read_file', got '%s'", msg.ToolCalls[0].Function.Name)
	}
}

func TestToolDefinition(t *testing.T) {
	tool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "read_file",
			Description: "Read the contents of a file",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to read",
					},
				},
				"required": []string{"path"},
			},
		},
	}

	if tool.Type != "function" {
		t.Errorf("Expected tool type 'function', got '%s'", tool.Type)
	}

	if tool.Function.Name != "read_file" {
		t.Errorf("Expected function name 'read_file', got '%s'", tool.Function.Name)
	}

	if tool.Function.Description != "Read the contents of a file" {
		t.Errorf("Expected description 'Read the contents of a file', got '%s'", tool.Function.Description)
	}
}

func TestProviderRegistry(t *testing.T) {
	// Clear existing providers for test
	providers = make(map[string]ProviderFactory)

	// Test registering a provider
	RegisterProvider("test", func(config map[string]interface{}) (Provider, error) {
		return &MockProvider{name: "test"}, nil
	})

	// Test getting a registered provider
	provider, err := GetProvider("test", nil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if provider.GetName() != "test" {
		t.Errorf("Expected provider name 'test', got '%s'", provider.GetName())
	}

	// Test getting a non-existent provider
	_, err = GetProvider("nonexistent", nil)
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}
}

// MockProvider is a test implementation of the Provider interface
type MockProvider struct {
	name string
}

func (m *MockProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{
		ID: "mock_response_123",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "This is a mock response",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

func (m *MockProvider) StreamChat(ctx context.Context, req *ChatRequest) (ChatStream, error) {
	return &MockStream{}, nil
}

func (m *MockProvider) GetName() string {
	return m.name
}

// MockStream is a test implementation of ChatStream
type MockStream struct {
	closed bool
}

func (m *MockStream) Recv() (*ChatStreamChunk, error) {
	if m.closed {
		return nil, io.EOF
	}
	return &ChatStreamChunk{
		ID: "mock_stream_123",
		Choices: []ChatStreamChoice{
			{
				Index: 0,
				Delta: ChatStreamDelta{
					Content: "Mock stream content",
				},
			},
		},
	}, nil
}

func (m *MockStream) Close() error {
	m.closed = true
	return nil
}
