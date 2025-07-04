package agent

import (
	"context"
	"io"
	"testing"

	"github.com/hammie/rubrduck/internal/ai"
	"github.com/hammie/rubrduck/internal/config"
	"github.com/stretchr/testify/require"
)

type mockStream struct {
	chunks []*ai.ChatStreamChunk
	idx    int
}

func (m *mockStream) Recv() (*ai.ChatStreamChunk, error) {
	if m.idx >= len(m.chunks) {
		return nil, io.EOF
	}
	ch := m.chunks[m.idx]
	m.idx++
	return ch, nil
}

func (m *mockStream) Close() error { return nil }

type mockProvider struct{}

func (m *mockProvider) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	return &ai.ChatResponse{
		Choices: []ai.Choice{{Message: ai.Message{Role: "assistant", Content: "final"}}},
		Usage:   ai.Usage{TotalTokens: 1},
	}, nil
}

func (m *mockProvider) StreamChat(ctx context.Context, req *ai.ChatRequest) (ai.ChatStream, error) {
	return &mockStream{chunks: []*ai.ChatStreamChunk{
		{Choices: []ai.ChatStreamChoice{{Delta: ai.ChatStreamDelta{Content: "Hello "}}}},
		{Choices: []ai.ChatStreamChoice{{Delta: ai.ChatStreamDelta{Content: "World"}}}},
	}}, nil
}

func (m *mockProvider) GetName() string { return "mock" }

func TestAgentStreamEvents(t *testing.T) {
	ai.RegisterProvider("mock", func(cfg map[string]interface{}) (ai.Provider, error) { return &mockProvider{}, nil })

	cfg := &config.Config{
		Provider: "mock",
		Model:    "gpt-4",
		Providers: map[string]config.Provider{
			"mock": {Name: "mock"},
		},
		Agent:   config.AgentConfig{ApprovalMode: "suggest"},
		Sandbox: config.SandboxPolicy{},
	}

	ag, err := New(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	ch, err := ag.StreamEvents(ctx, "hi")
	require.NoError(t, err)

	var out string
	for ev := range ch {
		if ev.Type == EventTokenChunk {
			out += ev.Token
		}
	}

	require.Equal(t, "Hello World", out)
	require.Greater(t, len(ag.GetHistory()), 1)
}
