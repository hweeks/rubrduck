package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hammie/rubrduck/internal/ai"
)

func TestNewOpenAIProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"api_key": "test-key",
			},
			wantErr: false,
		},
		{
			name: "missing api key",
			config: map[string]interface{}{
				"base_url": "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name: "custom base url",
			config: map[string]interface{}{
				"api_key":  "test-key",
				"base_url": "https://custom.openai.com/v1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOpenAIProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenAIProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewOpenAIProvider() returned nil provider")
			}
		})
	}
}

func TestOpenAIProvider_Chat(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}

		// Check headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("Expected Authorization Bearer header, got %s", r.Header.Get("Authorization"))
		}

		// Mock response
		response := map[string]interface{}{
			"id": "chatcmpl-123",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello! How can I help you today?",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 8,
				"total_tokens":      18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with test server URL
	provider, err := NewOpenAIProvider(map[string]interface{}{
		"api_key":  "test-key",
		"base_url": server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test chat request
	req := &ai.ChatRequest{
		Model: "gpt-4",
		Messages: []ai.Message{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.ID != "chatcmpl-123" {
		t.Errorf("Expected ID 'chatcmpl-123', got '%s'", resp.ID)
	}

	if len(resp.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != "Hello! How can I help you today?" {
		t.Errorf("Expected content 'Hello! How can I help you today?', got '%s'", resp.Choices[0].Message.Content)
	}

	if resp.Usage.TotalTokens != 18 {
		t.Errorf("Expected total tokens 18, got %d", resp.Usage.TotalTokens)
	}
}

func TestOpenAIProvider_Chat_Error(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid API key",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider(map[string]interface{}{
		"api_key":  "invalid-key",
		"base_url": server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &ai.ChatRequest{
		Model: "gpt-4",
		Messages: []ai.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = provider.Chat(ctx, req)
	if err == nil {
		t.Error("Expected error for invalid API key")
	}

	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Expected error to contain status 400, got: %v", err)
	}
}

func TestOpenAIProvider_StreamChat(t *testing.T) {
	// Create a test server for streaming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("Expected Accept text/event-stream, got %s", r.Header.Get("Accept"))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Send SSE data
		fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" world\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider(map[string]interface{}{
		"api_key":  "test-key",
		"base_url": server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &ai.ChatRequest{
		Model: "gpt-4",
		Messages: []ai.Message{
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := provider.StreamChat(ctx, req)
	if err != nil {
		t.Fatalf("StreamChat() error = %v", err)
	}
	defer stream.Close()

	// Read stream chunks
	chunks := []*ai.ChatStreamChunk{}
	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk from stream")
	}
}

func TestOpenAIProvider_ConvertRequest(t *testing.T) {
	provider := &OpenAIProvider{}

	req := &ai.ChatRequest{
		Model: "gpt-4",
		Messages: []ai.Message{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
		Stream:      true,
		Tools: []ai.Tool{
			{
				Type: "function",
				Function: ai.ToolFunction{
					Name:        "test_function",
					Description: "A test function",
					Parameters:  map[string]interface{}{},
				},
			},
		},
	}

	openAIReq := provider.convertRequest(req)

	if openAIReq["model"] != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%v'", openAIReq["model"])
	}

	if openAIReq["temperature"] != float32(0.7) {
		t.Errorf("Expected temperature 0.7, got '%v'", openAIReq["temperature"])
	}

	if openAIReq["max_tokens"] != 100 {
		t.Errorf("Expected max_tokens 100, got '%v'", openAIReq["max_tokens"])
	}

	if openAIReq["stream"] != true {
		t.Errorf("Expected stream true, got '%v'", openAIReq["stream"])
	}

	if len(openAIReq["tools"].([]ai.Tool)) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(openAIReq["tools"].([]ai.Tool)))
	}
}

func TestOpenAIProvider_ConvertResponse(t *testing.T) {
	provider := &OpenAIProvider{}

	openAIResp := &openAIChatResponse{
		ID: "chatcmpl-123",
		Choices: []ai.Choice{
			{
				Index: 0,
				Message: ai.Message{
					Role:    "assistant",
					Content: "Hello!",
				},
				FinishReason: "stop",
			},
		},
		Usage: ai.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	resp := provider.convertResponse(openAIResp)

	if resp.ID != "chatcmpl-123" {
		t.Errorf("Expected ID 'chatcmpl-123', got '%s'", resp.ID)
	}

	if len(resp.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != "Hello!" {
		t.Errorf("Expected content 'Hello!', got '%s'", resp.Choices[0].Message.Content)
	}

	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Expected total tokens 15, got %d", resp.Usage.TotalTokens)
	}
}

func TestOpenAIProvider_GetName(t *testing.T) {
	provider := &OpenAIProvider{}
	if provider.GetName() != "OpenAI" {
		t.Errorf("Expected name 'OpenAI', got '%s'", provider.GetName())
	}
}
