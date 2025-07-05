package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hammie/rubrduck/internal/ai"
)

func TestNewAnthropicProvider(t *testing.T) {
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
				"base_url": "https://api.anthropic.com/v1",
			},
			wantErr: true,
		},
		{
			name: "custom base url",
			config: map[string]interface{}{
				"api_key":  "test-key",
				"base_url": "https://custom.anthropic.com/v1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAnthropicProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAnthropicProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewAnthropicProvider() returned nil provider")
			}
		})
	}
}

func TestAnthropicProvider_Chat(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Errorf("Expected path /messages, got %s", r.URL.Path)
		}

		// Check headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("Expected x-api-key header, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("Expected anthropic-version header, got %s", r.Header.Get("anthropic-version"))
		}

		// Mock response
		response := map[string]interface{}{
			"id":    "msg_123",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-3-sonnet-20240229",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "Hello! How can I help you today?",
				},
			},
			"stop_reason": "end_turn",
			"usage": map[string]interface{}{
				"input_tokens":  10,
				"output_tokens": 8,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with test server URL
	provider, err := NewAnthropicProvider(map[string]interface{}{
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

	if resp.ID != "msg_123" {
		t.Errorf("Expected ID 'msg_123', got '%s'", resp.ID)
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

func TestAnthropicProvider_ModelConversion(t *testing.T) {
	provider := &AnthropicProvider{}

	tests := []struct {
		input    string
		expected string
	}{
		{"gpt-4", "claude-3-opus-20240229"},
		{"gpt-3.5-turbo", "claude-3-sonnet-20240229"},
		{"claude-3-opus-20240229", "claude-3-opus-20240229"},
		{"unknown", "claude-3-sonnet-20240229"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := provider.convertModel(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAnthropicProvider_ConvertRequest(t *testing.T) {
	provider := &AnthropicProvider{}

	req := &ai.ChatRequest{
		Model: "gpt-4",
		Messages: []ai.Message{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
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

	anthropicReq := provider.convertRequest(req)

	if anthropicReq["model"] != "claude-3-opus-20240229" {
		t.Errorf("Expected model 'claude-3-opus-20240229', got '%v'", anthropicReq["model"])
	}

	if anthropicReq["temperature"] != float32(0.7) {
		t.Errorf("Expected temperature 0.7, got '%v'", anthropicReq["temperature"])
	}

	if anthropicReq["max_tokens"] != 100 {
		t.Errorf("Expected max_tokens 100, got '%v'", anthropicReq["max_tokens"])
	}

	if len(anthropicReq["tools"].([]map[string]interface{})) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(anthropicReq["tools"].([]map[string]interface{})))
	}
}

func TestAnthropicProvider_GetName(t *testing.T) {
	provider := &AnthropicProvider{}
	if provider.GetName() != "Anthropic" {
		t.Errorf("Expected name 'Anthropic', got '%s'", provider.GetName())
	}
}
