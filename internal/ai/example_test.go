package ai

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Example demonstrates how to use the AI provider system
func Example() {
	// This example shows how to use different AI providers
	// Note: This is a test example and won't actually run API calls

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Example 1: Using OpenAI
	fmt.Println("=== OpenAI Example ===")
	openAIProvider, err := GetProvider("openai", map[string]interface{}{
		"api_key":  "your-openai-api-key",
		"base_url": "https://api.openai.com/v1",
	})
	if err != nil {
		log.Printf("Failed to create OpenAI provider: %v", err)
	} else {
		fmt.Printf("Provider: %s\n", openAIProvider.GetName())
	}

	// Example 2: Using Anthropic Claude
	fmt.Println("\n=== Anthropic Example ===")
	anthropicProvider, err := GetProvider("anthropic", map[string]interface{}{
		"api_key":  "your-anthropic-api-key",
		"base_url": "https://api.anthropic.com/v1",
	})
	if err != nil {
		log.Printf("Failed to create Anthropic provider: %v", err)
	} else {
		fmt.Printf("Provider: %s\n", anthropicProvider.GetName())
	}

	// Example 3: Using Google Gemini
	fmt.Println("\n=== Gemini Example ===")
	geminiProvider, err := GetProvider("gemini", map[string]interface{}{
		"api_key":  "your-gemini-api-key",
		"base_url": "https://generativelanguage.googleapis.com/v1beta",
	})
	if err != nil {
		log.Printf("Failed to create Gemini provider: %v", err)
	} else {
		fmt.Printf("Provider: %s\n", geminiProvider.GetName())
	}

	// Example 4: Using Local Ollama
	fmt.Println("\n=== Ollama Example ===")
	ollamaProvider, err := GetProvider("ollama", map[string]interface{}{
		"base_url": "http://localhost:11434",
	})
	if err != nil {
		log.Printf("Failed to create Ollama provider: %v", err)
	} else {
		fmt.Printf("Provider: %s\n", ollamaProvider.GetName())
	}

	// Example 5: Making a chat request
	fmt.Println("\n=== Chat Request Example ===")
	if openAIProvider != nil {
		req := &ChatRequest{
			Model: "gpt-4",
			Messages: []Message{
				{Role: "user", Content: "Hello! Can you help me with a coding problem?"},
			},
			Temperature: 0.7,
			MaxTokens:   100,
		}

		resp, err := openAIProvider.Chat(ctx, req)
		if err != nil {
			log.Printf("Chat request failed: %v", err)
		} else {
			fmt.Printf("Response ID: %s\n", resp.ID)
			fmt.Printf("Response: %s\n", resp.Choices[0].Message.Content)
			fmt.Printf("Tokens used: %d\n", resp.Usage.TotalTokens)
		}
	}

	// Example 6: Using tools/functions
	fmt.Println("\n=== Tools Example ===")
	tools := []Tool{
		{
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
		},
	}

	reqWithTools := &ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "user", Content: "Please read the file 'main.go' and explain what it does."},
		},
		Tools: tools,
	}

	if openAIProvider != nil {
		resp, err := openAIProvider.Chat(ctx, reqWithTools)
		if err != nil {
			log.Printf("Chat request with tools failed: %v", err)
		} else {
			fmt.Printf("Response with tools: %s\n", resp.Choices[0].Message.Content)
			if len(resp.Choices[0].Message.ToolCalls) > 0 {
				fmt.Printf("Tool calls: %d\n", len(resp.Choices[0].Message.ToolCalls))
			}
		}
	}

	// Example 7: Streaming response
	fmt.Println("\n=== Streaming Example ===")
	if openAIProvider != nil {
		streamReq := &ChatRequest{
			Model: "gpt-4",
			Messages: []Message{
				{Role: "user", Content: "Write a short story about a robot."},
			},
			Stream: true,
		}

		stream, err := openAIProvider.StreamChat(ctx, streamReq)
		if err != nil {
			log.Printf("Stream request failed: %v", err)
		} else {
			defer stream.Close()

			fmt.Print("Streaming response: ")
			for {
				chunk, err := stream.Recv()
				if err != nil {
					break
				}

				for _, choice := range chunk.Choices {
					if choice.Delta.Content != "" {
						fmt.Print(choice.Delta.Content)
					}
				}
			}
			fmt.Println()
		}
	}
}

// ExampleGetProvider demonstrates the provider registry
func ExampleGetProvider() {
	fmt.Println("=== Provider Registry Example ===")

	// Register a custom provider
	RegisterProvider("custom", func(config map[string]interface{}) (Provider, error) {
		return &MockProvider{name: "Custom Provider"}, nil
	})

	// Get the custom provider
	provider, err := GetProvider("custom", nil)
	if err != nil {
		log.Printf("Failed to get custom provider: %v", err)
	} else {
		fmt.Printf("Custom provider: %s\n", provider.GetName())
	}

	// List available providers (this would require additional registry methods)
	fmt.Println("Available providers: openai, anthropic, gemini, ollama, custom")
}

// ExampleChatRequest demonstrates error handling patterns
func ExampleChatRequest() {
	fmt.Println("=== Error Handling Example ===")

	// Try to get a non-existent provider
	_, err := GetProvider("nonexistent", nil)
	if err != nil {
		fmt.Printf("Expected error for non-existent provider: %v\n", err)
	}

	// Try to create OpenAI provider without API key
	_, err = GetProvider("openai", map[string]interface{}{})
	if err != nil {
		fmt.Printf("Expected error for missing API key: %v\n", err)
	}

	// Try to create Anthropic provider without API key
	_, err = GetProvider("anthropic", map[string]interface{}{})
	if err != nil {
		fmt.Printf("Expected error for missing API key: %v\n", err)
	}
}

// ExampleMessage demonstrates model name mapping
func ExampleMessage() {
	fmt.Println("=== Model Mapping Example ===")

	// Different providers map model names differently
	providers := []string{"openai", "anthropic", "gemini", "ollama"}

	for _, providerName := range providers {
		provider, err := GetProvider(providerName, map[string]interface{}{
			"api_key": "dummy-key", // Will fail but we can still test model mapping
		})
		if err != nil {
			fmt.Printf("%s: %v\n", providerName, err)
			continue
		}

		fmt.Printf("%s provider: %s\n", providerName, provider.GetName())
	}

	fmt.Println("Model mappings:")
	fmt.Println("- gpt-4 -> OpenAI: gpt-4, Anthropic: claude-3-opus-20240229, Gemini: gemini-1.5-pro, Ollama: llama3.2:3b")
	fmt.Println("- gpt-3.5-turbo -> OpenAI: gpt-3.5-turbo, Anthropic: claude-3-sonnet-20240229, Gemini: gemini-1.5-flash, Ollama: llama3.2:3b")
}
