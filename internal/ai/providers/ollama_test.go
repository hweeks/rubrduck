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

func TestNewOllamaProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config with custom url",
			config: map[string]interface{}{
				"base_url": "http://localhost:11434",
			},
			wantErr: false,
		},
		{
			name:    "empty config uses default",
			config:  map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "custom base url",
			config: map[string]interface{}{
				"base_url": "http://custom.ollama.com:11434",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOllamaProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOllamaProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewOllamaProvider() returned nil provider")
			}
		})
	}
}

func TestOllamaProvider_Chat(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/chat" {
			t.Errorf("Expected path /api/chat, got %s", r.URL.Path)
		}

		// Check headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Mock response
		response := map[string]interface{}{
			"model":      "llama3.2:3b",
			"created_at": "2024-01-01T00:00:00Z",
			"message": map[string]interface{}{
				"role":    "assistant",
				"content": "Hello! How can I help you today?",
			},
			"done":              true,
			"total_duration":    1000000000,
			"load_duration":     50000000,
			"prompt_eval_count": 10,
			"eval_count":        8,
			"eval_duration":     950000000,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with test server URL
	provider, err := NewOllamaProvider(map[string]interface{}{
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

	if resp.ID != "llama3.2:3b" {
		t.Errorf("Expected ID 'llama3.2:3b', got '%s'", resp.ID)
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

func TestOllamaProvider_ModelConversion(t *testing.T) {
	provider := &OllamaProvider{}

	tests := []struct {
		input    string
		expected string
	}{
		{"gpt-4", "llama3.2:3b"},
		{"gpt-3.5-turbo", "llama3.2:3b"},
		{"claude-3-opus-20240229", "llama3.2:3b"},
		{"claude-3-sonnet-20240229", "llama3.2:3b"},
		{"gemini-1.5-pro", "llama3.2:3b"},
		{"llama3.2:3b", "llama3.2:3b"},
		{"mistral:7b", "mistral:7b"},
		{"unknown", "llama3.2:3b"},
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

func TestOllamaProvider_ConvertRequest(t *testing.T) {
	provider := &OllamaProvider{}

	req := &ai.ChatRequest{
		Model: "gpt-4",
		Messages: []ai.Message{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	ollamaReq := provider.convertRequest(req)

	if ollamaReq["model"] != "llama3.2:3b" {
		t.Errorf("Expected model 'llama3.2:3b', got '%v'", ollamaReq["model"])
	}

	if len(ollamaReq["messages"].([]map[string]interface{})) != 1 {
		t.Errorf("Expected 1 message, got %d", len(ollamaReq["messages"].([]map[string]interface{})))
	}

	options := ollamaReq["options"].(map[string]interface{})
	if options["temperature"] != float32(0.7) {
		t.Errorf("Expected temperature 0.7, got '%v'", options["temperature"])
	}

	if options["num_predict"] != 100 {
		t.Errorf("Expected num_predict 100, got '%v'", options["num_predict"])
	}
}

func TestOllamaProvider_GetName(t *testing.T) {
	provider := &OllamaProvider{}
	if provider.GetName() != "Ollama" {
		t.Errorf("Expected name 'Ollama', got '%s'", provider.GetName())
	}
}
