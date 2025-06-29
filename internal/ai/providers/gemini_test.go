package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hammie/rubrduck/internal/ai"
)

func TestNewGeminiProvider(t *testing.T) {
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
				"base_url": "https://generativelanguage.googleapis.com/v1beta",
			},
			wantErr: true,
		},
		{
			name: "custom base url",
			config: map[string]interface{}{
				"api_key":  "test-key",
				"base_url": "https://custom.gemini.com/v1beta",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGeminiProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGeminiProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewGeminiProvider() returned nil provider")
			}
		})
	}
}

func TestGeminiProvider_Chat(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Check URL contains the expected path
		if !strings.Contains(r.URL.String(), "/models/gemini-1.5-pro:generateContent") {
			t.Errorf("Expected URL to contain /models/gemini-1.5-pro:generateContent, got %s", r.URL.String())
		}

		// Check headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Mock response
		response := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"index": 0,
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{
								"text": "Hello! How can I help you today?",
							},
						},
						"role": "assistant",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]interface{}{
				"promptTokenCount":     10,
				"candidatesTokenCount": 8,
				"totalTokenCount":      18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with test server URL
	provider, err := NewGeminiProvider(map[string]interface{}{
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

	if resp.ID != "0" {
		t.Errorf("Expected ID '0', got '%s'", resp.ID)
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

func TestGeminiProvider_ModelConversion(t *testing.T) {
	provider := &GeminiProvider{}

	tests := []struct {
		input    string
		expected string
	}{
		{"gpt-4", "gemini-1.5-pro"},
		{"gpt-3.5-turbo", "gemini-1.5-flash"},
		{"gemini-1.5-pro", "gemini-1.5-pro"},
		{"unknown", "gemini-1.5-pro"},
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

func TestGeminiProvider_ConvertRequest(t *testing.T) {
	provider := &GeminiProvider{}

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

	geminiReq := provider.convertRequest(req)

	if len(geminiReq["contents"].([]map[string]interface{})) != 1 {
		t.Errorf("Expected 1 content, got %d", len(geminiReq["contents"].([]map[string]interface{})))
	}

	generationConfig := geminiReq["generationConfig"].(map[string]interface{})
	if generationConfig["temperature"] != float32(0.7) {
		t.Errorf("Expected temperature 0.7, got '%v'", generationConfig["temperature"])
	}

	if generationConfig["maxOutputTokens"] != 100 {
		t.Errorf("Expected maxOutputTokens 100, got '%v'", generationConfig["maxOutputTokens"])
	}

	if len(geminiReq["tools"].([]map[string]interface{})) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(geminiReq["tools"].([]map[string]interface{})))
	}
}

func TestGeminiProvider_GetName(t *testing.T) {
	provider := &GeminiProvider{}
	if provider.GetName() != "Gemini" {
		t.Errorf("Expected name 'Gemini', got '%s'", provider.GetName())
	}
}
