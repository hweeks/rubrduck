package ai

import (
	"context"
	"io"
)

// Provider defines the interface for AI providers
type Provider interface {
	// Chat sends a chat completion request and returns the response
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// StreamChat sends a chat completion request and streams the response
	StreamChat(ctx context.Context, req *ChatRequest) (ChatStream, error)

	// GetName returns the provider name
	GetName() string
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Name      string     `json:"name,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Tool represents a function that can be called by the AI
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a function that can be called
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a function call made by the AI
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a response choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatStream represents a stream of chat responses
type ChatStream interface {
	// Recv receives the next chunk from the stream
	Recv() (*ChatStreamChunk, error)

	// Close closes the stream
	Close() error
}

// ChatStreamChunk represents a chunk of streamed response
type ChatStreamChunk struct {
	ID      string             `json:"id"`
	Choices []ChatStreamChoice `json:"choices"`
}

// ChatStreamChoice represents a streamed choice
type ChatStreamChoice struct {
	Index        int             `json:"index"`
	Delta        ChatStreamDelta `json:"delta"`
	FinishReason *string         `json:"finish_reason"`
}

// ChatStreamDelta represents the delta content in a stream
type ChatStreamDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ProviderFactory creates a provider instance
type ProviderFactory func(config map[string]interface{}) (Provider, error)

// Registry of available providers
var providers = make(map[string]ProviderFactory)

// RegisterProvider registers a new AI provider
func RegisterProvider(name string, factory ProviderFactory) {
	providers[name] = factory
}

// GetProvider returns a provider by name
func GetProvider(name string, config map[string]interface{}) (Provider, error) {
	factory, ok := providers[name]
	if !ok {
		return nil, io.EOF
	}
	return factory(config)
}
