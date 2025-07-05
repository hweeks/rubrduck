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
	// Register the OpenAI provider
	ai.RegisterProvider("openai", NewOpenAIProvider)
}

// OpenAIProvider implements the ai.Provider interface for OpenAI
type OpenAIProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider instance
func NewOpenAIProvider(config map[string]interface{}) (ai.Provider, error) {
	apiKey, _ := config["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	baseURL, _ := config["base_url"].(string)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	// Create HTTP client with reasonable timeouts
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	return &OpenAIProvider{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}, nil
}

// GetName returns the provider name
func (p *OpenAIProvider) GetName() string {
	return "OpenAI"
}

// Chat sends a chat completion request
func (p *OpenAIProvider) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	// Convert our request to OpenAI format
	openAIReq := p.convertRequest(req)

	// Marshal request
	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openAIResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our format
	return p.convertResponse(&openAIResp), nil
}

// StreamChat sends a streaming chat completion request
func (p *OpenAIProvider) StreamChat(ctx context.Context, req *ai.ChatRequest) (ai.ChatStream, error) {
	// Set streaming flag
	req.Stream = true

	// Convert our request to OpenAI format
	openAIReq := p.convertRequest(req)

	// Marshal request
	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
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
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Return stream
	return &openAIStream{
		reader:  resp.Body,
		scanner: bufio.NewScanner(resp.Body),
	}, nil
}

// convertRequest converts our request format to OpenAI format
func (p *OpenAIProvider) convertRequest(req *ai.ChatRequest) map[string]interface{} {
	// Convert messages to support text and image parts
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
						"type":      "image_url",
						"image_url": map[string]interface{}{"url": part.ImageURL},
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
		if msg.Name != "" {
			m["name"] = msg.Name
		}
		if msg.ToolCallID != "" {
			m["tool_call_id"] = msg.ToolCallID
		}
		if len(msg.ToolCalls) > 0 {
			m["tool_calls"] = msg.ToolCalls
		}
		messages[i] = m
	}

	openAIReq := map[string]interface{}{
		"model":      req.Model,
		"messages":   messages,
		"stream":     req.Stream,
		"max_tokens": 2048, // Reduced to leave more room for input context
	}

	if req.Temperature > 0 {
		openAIReq["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		openAIReq["max_tokens"] = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		openAIReq["tools"] = req.Tools
	}

	return openAIReq
}

// convertResponse converts OpenAI response to our format
func (p *OpenAIProvider) convertResponse(resp *openAIChatResponse) *ai.ChatResponse {
	choices := make([]ai.Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		choices[i] = ai.Choice{
			Index:        c.Index,
			Message:      c.Message,
			FinishReason: c.FinishReason,
		}
	}

	return &ai.ChatResponse{
		ID:      resp.ID,
		Choices: choices,
		Usage:   resp.Usage,
	}
}

// OpenAI response types
type openAIChatResponse struct {
	ID      string      `json:"id"`
	Choices []ai.Choice `json:"choices"`
	Usage   ai.Usage    `json:"usage"`
}

// openAIStream implements ai.ChatStream for OpenAI SSE responses
type openAIStream struct {
	reader  io.ReadCloser
	scanner *bufio.Scanner
}

func (s *openAIStream) Recv() (*ai.ChatStreamChunk, error) {
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

func (s *openAIStream) Close() error {
	return s.reader.Close()
}
