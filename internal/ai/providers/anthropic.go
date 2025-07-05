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
	// Register the Anthropic provider
	ai.RegisterProvider("anthropic", NewAnthropicProvider)
}

// AnthropicProvider implements the ai.Provider interface for Anthropic Claude
type AnthropicProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewAnthropicProvider creates a new Anthropic provider instance
func NewAnthropicProvider(config map[string]interface{}) (ai.Provider, error) {
	apiKey, _ := config["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	baseURL, _ := config["base_url"].(string)
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	// Create HTTP client with reasonable timeouts
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	return &AnthropicProvider{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}, nil
}

// GetName returns the provider name
func (p *AnthropicProvider) GetName() string {
	return "Anthropic"
}

// Chat sends a chat completion request
func (p *AnthropicProvider) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	// Convert our request to Anthropic format
	anthropicReq := p.convertRequest(req)

	// Marshal request
	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var anthropicResp anthropicMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our format
	return p.convertResponse(&anthropicResp), nil
}

// StreamChat sends a streaming chat completion request
func (p *AnthropicProvider) StreamChat(ctx context.Context, req *ai.ChatRequest) (ai.ChatStream, error) {
	// Convert our request to Anthropic format with streaming
	anthropicReq := p.convertRequest(req)
	anthropicReq["stream"] = true

	// Marshal request
	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Return stream
	return &anthropicStream{
		reader:  resp.Body,
		scanner: bufio.NewScanner(resp.Body),
	}, nil
}

// convertRequest converts our request format to Anthropic format
func (p *AnthropicProvider) convertRequest(req *ai.ChatRequest) map[string]interface{} {
	// Convert messages to Anthropic format
	messages := make([]map[string]interface{}, len(req.Messages))
	for i, msg := range req.Messages {
		m := map[string]interface{}{
			"role": msg.Role,
		}
		if len(msg.Parts) > 0 {
			parts := make([]map[string]interface{}, len(msg.Parts))
			for j, part := range msg.Parts {
				if part.Type == "image_url" {
					parts[j] = map[string]interface{}{
						"type":   "image",
						"source": map[string]interface{}{"data": part.ImageURL, "media_type": "image/png"},
					}
				} else {
					parts[j] = map[string]interface{}{
						"type": "text",
						"text": part.Text,
					}
				}
			}
			m["content"] = parts
		} else {
			m["content"] = msg.Content
		}
		messages[i] = m
	}

	anthropicReq := map[string]interface{}{
		"model":      p.convertModel(req.Model),
		"messages":   messages,
		"max_tokens": 2048, // Reduced to leave more room for input context
	}

	if req.Temperature > 0 {
		anthropicReq["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		anthropicReq["max_tokens"] = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		anthropicReq["tools"] = p.convertTools(req.Tools)
	}

	return anthropicReq
}

// convertModel converts our model names to Anthropic model names
func (p *AnthropicProvider) convertModel(model string) string {
	switch model {
	case "gpt-4":
		return "claude-3-opus-20240229"
	case "gpt-3.5-turbo":
		return "claude-3-sonnet-20240229"
	default:
		// If it's already an Anthropic model name, return as is
		if strings.HasPrefix(model, "claude-") {
			return model
		}
		// Default to Claude 3 Sonnet
		return "claude-3-sonnet-20240229"
	}
}

// convertTools converts our tool format to Anthropic format
func (p *AnthropicProvider) convertTools(tools []ai.Tool) []map[string]interface{} {
	anthropicTools := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		anthropicTools[i] = map[string]interface{}{
			"name":        tool.Function.Name,
			"description": tool.Function.Description,
			"input_schema": map[string]interface{}{
				"type":       "object",
				"properties": tool.Function.Parameters,
			},
		}
	}
	return anthropicTools
}

// convertResponse converts Anthropic response to our format
func (p *AnthropicProvider) convertResponse(resp *anthropicMessageResponse) *ai.ChatResponse {
	choices := make([]ai.Choice, len(resp.Content))
	for i, content := range resp.Content {
		choices[i] = ai.Choice{
			Index: i,
			Message: ai.Message{
				Role:    "assistant",
				Content: content.Text,
			},
			FinishReason: resp.StopReason,
		}
	}

	return &ai.ChatResponse{
		ID:      resp.ID,
		Choices: choices,
		Usage: ai.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

// Anthropic response types
type anthropicMessageResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []anthropicContent `json:"content"`
	Model      string             `json:"model"`
	StopReason string             `json:"stop_reason"`
	StopSeq    *string            `json:"stop_sequence"`
	Usage      anthropicUsage     `json:"usage"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// anthropicStream implements ai.ChatStream for Anthropic SSE responses
type anthropicStream struct {
	reader  io.ReadCloser
	scanner *bufio.Scanner
}

func (s *anthropicStream) Recv() (*ai.ChatStreamChunk, error) {
	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for [DONE] marker
		if line == "data: [DONE]" {
			return nil, io.EOF
		}

		// Parse SSE data line
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Skip empty data
			if data == "" {
				continue
			}

			// Parse JSON chunk
			var chunk ai.ChatStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// Skip malformed JSON
				continue
			}

			return &chunk, nil
		}
	}

	// Check for scanner errors
	if err := s.scanner.Err(); err != nil {
		return nil, fmt.Errorf("stream scanner error: %w", err)
	}

	return nil, io.EOF
}

func (s *anthropicStream) Close() error {
	return s.reader.Close()
}
