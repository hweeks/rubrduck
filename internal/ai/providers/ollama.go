package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hammie/rubrduck/internal/ai"
)

func init() {
	// Register the Ollama provider
	ai.RegisterProvider("ollama", NewOllamaProvider)
}

// OllamaProvider implements the ai.Provider interface for local Ollama models
type OllamaProvider struct {
	baseURL    string
	httpClient *http.Client
}

// NewOllamaProvider creates a new Ollama provider instance
func NewOllamaProvider(config map[string]interface{}) (ai.Provider, error) {
	baseURL, _ := config["base_url"].(string)
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Create HTTP client with reasonable timeouts
	httpClient := &http.Client{
		Timeout: 120 * time.Second, // Longer timeout for local models
	}

	return &OllamaProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}, nil
}

// GetName returns the provider name
func (p *OllamaProvider) GetName() string {
	return "Ollama"
}

// Chat sends a chat completion request
func (p *OllamaProvider) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	// Convert our request to Ollama format
	ollamaReq := p.convertRequest(req)

	// Marshal request
	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our format
	return p.convertResponse(&ollamaResp), nil
}

// StreamChat sends a streaming chat completion request
func (p *OllamaProvider) StreamChat(ctx context.Context, req *ai.ChatRequest) (ai.ChatStream, error) {
	// Convert our request to Ollama format with streaming
	ollamaReq := p.convertRequest(req)
	ollamaReq["stream"] = true

	// Marshal request
	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Return stream
	return &ollamaStream{
		reader:  resp.Body,
		scanner: bufio.NewScanner(resp.Body),
	}, nil
}

// convertRequest converts our request format to Ollama format
func (p *OllamaProvider) convertRequest(req *ai.ChatRequest) map[string]interface{} {
	// Convert messages to Ollama format
	messages := make([]map[string]interface{}, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	ollamaReq := map[string]interface{}{
		"model":    p.convertModel(req.Model),
		"messages": messages,
	}

	if req.Temperature > 0 {
		ollamaReq["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		ollamaReq["num_predict"] = req.MaxTokens
	}

	// Add options for better control
	options := map[string]interface{}{}
	if req.Temperature > 0 {
		options["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		options["num_predict"] = req.MaxTokens
	}
	if len(options) > 0 {
		ollamaReq["options"] = options
	}

	return ollamaReq
}

// convertModel converts our model names to Ollama model names
func (p *OllamaProvider) convertModel(model string) string {
	switch model {
	case "gpt-4":
		return "llama3.2:3b" // Default to a reasonable local model
	case "gpt-3.5-turbo":
		return "llama3.2:3b"
	case "claude-3-opus-20240229":
		return "llama3.2:3b"
	case "claude-3-sonnet-20240229":
		return "llama3.2:3b"
	case "gemini-1.5-pro":
		return "llama3.2:3b"
	default:
		// If it's already an Ollama model name, return as is
		if strings.Contains(model, ":") || strings.HasPrefix(model, "llama") || strings.HasPrefix(model, "mistral") {
			return model
		}
		// Default to Llama 3.2 3B
		return "llama3.2:3b"
	}
}

// convertResponse converts Ollama response to our format
func (p *OllamaProvider) convertResponse(resp *ollamaChatResponse) *ai.ChatResponse {
	choices := []ai.Choice{
		{
			Index: 0,
			Message: ai.Message{
				Role:    "assistant",
				Content: resp.Message.Content,
			},
			FinishReason: "stop",
		},
	}

	return &ai.ChatResponse{
		ID:      resp.Model,
		Choices: choices,
		Usage: ai.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
	}
}

// Ollama response types
type ollamaChatResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done            bool  `json:"done"`
	TotalDuration   int64 `json:"total_duration"`
	LoadDuration    int64 `json:"load_duration"`
	PromptEvalCount int   `json:"prompt_eval_count"`
	EvalCount       int   `json:"eval_count"`
	EvalDuration    int64 `json:"eval_duration"`
}

// ollamaStream implements ai.ChatStream for Ollama streaming responses
type ollamaStream struct {
	reader  io.ReadCloser
	scanner *bufio.Scanner
}

func (s *ollamaStream) Recv() (*ai.ChatStreamChunk, error) {
	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse JSON chunk
		var chunk ai.ChatStreamChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			// Skip malformed JSON
			continue
		}

		return &chunk, nil
	}

	// Check for scanner errors
	if err := s.scanner.Err(); err != nil {
		return nil, fmt.Errorf("stream scanner error: %w", err)
	}

	return nil, io.EOF
}

func (s *ollamaStream) Close() error {
	return s.reader.Close()
}
