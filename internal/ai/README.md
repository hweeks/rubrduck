# AI Provider System

The AI Provider System in RubrDuck provides a unified interface for interacting with multiple AI services. It supports OpenAI, Anthropic Claude, Google Gemini, and local Ollama models.

## Features

- **Multi-Provider Support**: OpenAI, Anthropic, Gemini, and Ollama
- **Streaming Support**: Real-time streaming responses for all providers
- **Function Calling**: Tool/function calling support
- **Model Mapping**: Automatic model name conversion between providers
- **Error Handling**: Comprehensive error handling and retry logic
- **Extensible**: Easy to add new providers

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/hammie/rubrduck/internal/ai"
)

func main() {
    // Create an OpenAI provider
    provider, err := ai.GetProvider("openai", map[string]interface{}{
        "api_key": "your-openai-api-key",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Make a chat request
    ctx := context.Background()
    req := &ai.ChatRequest{
        Model: "gpt-4",
        Messages: []ai.Message{
            {Role: "user", Content: "Hello, world!"},
        },
    }

    resp, err := provider.Chat(ctx, req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

## Supported Providers

### OpenAI

```go
provider, err := ai.GetProvider("openai", map[string]interface{}{
    "api_key": "your-openai-api-key",
    "base_url": "https://api.openai.com/v1", // optional, defaults to OpenAI
})
```

**Supported Models**: `gpt-4`, `gpt-3.5-turbo`, `gpt-4-turbo`, etc.

### Anthropic Claude

```go
provider, err := ai.GetProvider("anthropic", map[string]interface{}{
    "api_key": "your-anthropic-api-key",
    "base_url": "https://api.anthropic.com/v1", // optional
})
```

**Supported Models**: `claude-3-opus-20240229`, `claude-3-sonnet-20240229`, `claude-3-haiku-20240307`

### Google Gemini

```go
provider, err := ai.GetProvider("gemini", map[string]interface{}{
    "api_key": "your-gemini-api-key",
    "base_url": "https://generativelanguage.googleapis.com/v1beta", // optional
})
```

**Supported Models**: `gemini-1.5-pro`, `gemini-1.5-flash`, `gemini-pro`

### Local Ollama

```go
provider, err := ai.GetProvider("ollama", map[string]interface{}{
    "base_url": "http://localhost:11434", // optional, defaults to localhost
})
```

**Supported Models**: Any model available in your Ollama installation (e.g., `llama3.2:3b`, `mistral:7b`, `codellama:7b`)

## Model Mapping

The system automatically maps common model names to provider-specific models:

| Generic Name    | OpenAI          | Anthropic                  | Gemini             | Ollama        |
| --------------- | --------------- | -------------------------- | ------------------ | ------------- |
| `gpt-4`         | `gpt-4`         | `claude-3-opus-20240229`   | `gemini-1.5-pro`   | `llama3.2:3b` |
| `gpt-3.5-turbo` | `gpt-3.5-turbo` | `claude-3-sonnet-20240229` | `gemini-1.5-flash` | `llama3.2:3b` |

## Streaming Responses

All providers support streaming responses:

```go
req := &ai.ChatRequest{
    Model: "gpt-4",
    Messages: []ai.Message{
        {Role: "user", Content: "Write a story"},
    },
    Stream: true,
}

stream, err := provider.StreamChat(ctx, req)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

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
```

## Function Calling

All providers support function calling (tools):

```go
tools := []ai.Tool{
    {
        Type: "function",
        Function: ai.ToolFunction{
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

req := &ai.ChatRequest{
    Model: "gpt-4",
    Messages: []ai.Message{
        {Role: "user", Content: "Read the file 'main.go'"},
    },
    Tools: tools,
}

resp, err := provider.Chat(ctx, req)
if err != nil {
    log.Fatal(err)
}

// Check for tool calls
if len(resp.Choices[0].Message.ToolCalls) > 0 {
    for _, toolCall := range resp.Choices[0].Message.ToolCalls {
        fmt.Printf("Tool call: %s with args: %s\n",
            toolCall.Function.Name, toolCall.Function.Arguments)
    }
}
```

## Error Handling

The system provides comprehensive error handling:

```go
provider, err := ai.GetProvider("openai", map[string]interface{}{
    "api_key": "invalid-key",
})
if err != nil {
    // Handle provider creation error
    log.Printf("Provider creation failed: %v", err)
    return
}

resp, err := provider.Chat(ctx, req)
if err != nil {
    // Handle API error
    if strings.Contains(err.Error(), "401") {
        log.Printf("Authentication failed: %v", err)
    } else if strings.Contains(err.Error(), "429") {
        log.Printf("Rate limit exceeded: %v", err)
    } else {
        log.Printf("API error: %v", err)
    }
    return
}
```

## Configuration

### Environment Variables

You can configure providers using environment variables:

```bash
export OPENAI_API_KEY="your-openai-key"
export ANTHROPIC_API_KEY="your-anthropic-key"
export GEMINI_API_KEY="your-gemini-key"
```

### Configuration File

```yaml
ai:
  providers:
    openai:
      api_key: "${OPENAI_API_KEY}"
      base_url: "https://api.openai.com/v1"
    anthropic:
      api_key: "${ANTHROPIC_API_KEY}"
      base_url: "https://api.anthropic.com/v1"
    gemini:
      api_key: "${GEMINI_API_KEY}"
      base_url: "https://generativelanguage.googleapis.com/v1beta"
    ollama:
      base_url: "http://localhost:11434"
```

## Adding Custom Providers

You can easily add custom providers by implementing the `Provider` interface:

```go
type CustomProvider struct {
    // Your provider fields
}

func (p *CustomProvider) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
    // Implement chat functionality
}

func (p *CustomProvider) StreamChat(ctx context.Context, req *ai.ChatRequest) (ai.ChatStream, error) {
    // Implement streaming functionality
}

func (p *CustomProvider) GetName() string {
    return "Custom Provider"
}

// Register the provider
func init() {
    ai.RegisterProvider("custom", func(config map[string]interface{}) (ai.Provider, error) {
        return &CustomProvider{}, nil
    })
}
```

## Testing

Run the test suite:

```bash
go test ./internal/ai/... -v
```

The test suite includes:

- Unit tests for all providers
- Integration tests with mock servers
- Error handling tests
- Streaming tests

## Performance Considerations

- **Connection Pooling**: Each provider uses a single HTTP client with connection pooling
- **Timeouts**: Configurable timeouts (60s for cloud providers, 120s for local Ollama)
- **Streaming**: Efficient streaming with minimal memory usage
- **Error Retries**: Built-in retry logic for transient failures

## Security

- **API Key Management**: Secure API key handling
- **Request Validation**: Input validation for all requests
- **Error Sanitization**: Sensitive information is not leaked in error messages
- **HTTPS**: All cloud providers use HTTPS by default

## Troubleshooting

### Common Issues

1. **Authentication Errors**: Check your API keys
2. **Rate Limiting**: Implement exponential backoff
3. **Network Issues**: Check your internet connection
4. **Ollama Not Running**: Ensure Ollama is running on the expected port

### Debug Mode

Enable debug logging:

```go
// Set log level to debug
log.SetLevel(log.DebugLevel)
```

## Contributing

To add a new provider:

1. Create a new file in `internal/ai/providers/`
2. Implement the `Provider` interface
3. Add tests in a corresponding `_test.go` file
4. Register the provider in the `init()` function
5. Update this documentation

## License

This code is part of the RubrDuck project and follows the same license terms.
