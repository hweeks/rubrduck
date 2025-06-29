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
	// Register the Gemini provider
	ai.RegisterProvider("gemini", NewGeminiProvider)
}

// GeminiProvider implements the ai.Provider interface for Google Gemini
type GeminiProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewGeminiProvider creates a new Gemini provider instance
func NewGeminiProvider(config map[string]interface{}) (ai.Provider, error) {
	apiKey, _ := config["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	baseURL, _ := config["base_url"].(string)
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	// Create HTTP client with reasonable timeouts
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	return &GeminiProvider{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}, nil
}

// GetName returns the provider name
func (p *GeminiProvider) GetName() string {
	return "Gemini"
}

// Chat sends a chat completion request
func (p *GeminiProvider) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	// Convert our request to Gemini format
	geminiReq := p.convertRequest(req)

	// Marshal request
	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, p.convertModel(req.Model), p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
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
		return nil, fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var geminiResp geminiGenerateContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our format
	return p.convertResponse(&geminiResp), nil
}

// StreamChat sends a streaming chat completion request
func (p *GeminiProvider) StreamChat(ctx context.Context, req *ai.ChatRequest) (ai.ChatStream, error) {
	// Convert our request to Gemini format with streaming
	geminiReq := p.convertRequest(req)

	// Marshal request
	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s", p.baseURL, p.convertModel(req.Model), p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
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
		return nil, fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Return stream
	return &geminiStream{
		reader:  resp.Body,
		scanner: bufio.NewScanner(resp.Body),
	}, nil
}

// convertRequest converts our request format to Gemini format
func (p *GeminiProvider) convertRequest(req *ai.ChatRequest) map[string]interface{} {
	// Convert messages to Gemini format
	contents := make([]map[string]interface{}, len(req.Messages))
	for i, msg := range req.Messages {
		contents[i] = map[string]interface{}{
			"role": msg.Role,
			"parts": []map[string]interface{}{
				{
					"text": msg.Content,
				},
			},
		}
	}

	geminiReq := map[string]interface{}{
		"contents": contents,
	}

	// Add generation config
	generationConfig := map[string]interface{}{}
	if req.Temperature > 0 {
		generationConfig["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		generationConfig["maxOutputTokens"] = req.MaxTokens
	}
	if len(generationConfig) > 0 {
		geminiReq["generationConfig"] = generationConfig
	}

	// Add tools if present
	if len(req.Tools) > 0 {
		geminiReq["tools"] = p.convertTools(req.Tools)
	}

	return geminiReq
}

// convertModel converts our model names to Gemini model names
func (p *GeminiProvider) convertModel(model string) string {
	switch model {
	case "gpt-4":
		return "gemini-1.5-pro"
	case "gpt-3.5-turbo":
		return "gemini-1.5-flash"
	default:
		// If it's already a Gemini model name, return as is
		if strings.HasPrefix(model, "gemini-") {
			return model
		}
		// Default to Gemini 1.5 Pro
		return "gemini-1.5-pro"
	}
}

// convertTools converts our tool format to Gemini format
func (p *GeminiProvider) convertTools(tools []ai.Tool) []map[string]interface{} {
	geminiTools := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		geminiTools[i] = map[string]interface{}{
			"functionDeclarations": []map[string]interface{}{
				{
					"name":        tool.Function.Name,
					"description": tool.Function.Description,
					"parameters":  tool.Function.Parameters,
				},
			},
		}
	}
	return geminiTools
}

// convertResponse converts Gemini response to our format
func (p *GeminiProvider) convertResponse(resp *geminiGenerateContentResponse) *ai.ChatResponse {
	choices := make([]ai.Choice, len(resp.Candidates))
	for i, candidate := range resp.Candidates {
		content := ""
		if len(candidate.Content.Parts) > 0 {
			content = candidate.Content.Parts[0].Text
		}

		choices[i] = ai.Choice{
			Index: i,
			Message: ai.Message{
				Role:    "assistant",
				Content: content,
			},
			FinishReason: candidate.FinishReason,
		}
	}

	return &ai.ChatResponse{
		ID:      fmt.Sprintf("%d", resp.Candidates[0].Index), // Convert index to string
		Choices: choices,
		Usage: ai.Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		},
	}
}

// Gemini response types
type geminiGenerateContentResponse struct {
	Candidates     []geminiCandidate     `json:"candidates"`
	PromptFeedback *geminiPromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  geminiUsageMetadata   `json:"usageMetadata"`
}

type geminiCandidate struct {
	Index         int                  `json:"index"`
	Content       geminiContent        `json:"content"`
	FinishReason  string               `json:"finishReason"`
	SafetyRatings []geminiSafetyRating `json:"safetyRatings,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiPromptFeedback struct {
	SafetyRatings []geminiSafetyRating `json:"safetyRatings"`
}

type geminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// geminiStream implements ai.ChatStream for Gemini streaming responses
type geminiStream struct {
	reader  io.ReadCloser
	scanner *bufio.Scanner
}

func (s *geminiStream) Recv() (*ai.ChatStreamChunk, error) {
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

func (s *geminiStream) Close() error {
	return s.reader.Close()
}
