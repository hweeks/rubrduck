package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test structures are now defined in types.go

func TestChatHandler(t *testing.T) {
	tests := []struct {
		name           string
		request        ChatRequest
		expectedStatus int
		checkResponse  func(t *testing.T, resp ChatResponse)
		setupMock      func(h *Handler)
	}{
		{
			name: "successful chat request",
			request: ChatRequest{
				Messages: []Message{
					{Role: "user", Content: "Hello, how are you?"},
				},
				Model: "gpt-4",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp ChatResponse) {
				assert.NotEmpty(t, resp.ID)
				assert.Equal(t, "assistant", resp.Message.Role)
				assert.NotEmpty(t, resp.Message.Content)
				assert.False(t, resp.Created.IsZero())
			},
		},
		{
			name: "empty messages",
			request: ChatRequest{
				Messages: []Message{},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid message role",
			request: ChatRequest{
				Messages: []Message{
					{Role: "invalid", Content: "test"},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "model not specified uses default",
			request: ChatRequest{
				Messages: []Message{
					{Role: "user", Content: "test"},
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp ChatResponse) {
				assert.NotEmpty(t, resp.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(nil) // Will be mocked
			if tt.setupMock != nil {
				tt.setupMock(handler)
			}

			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.HandleChat(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var resp ChatResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestStreamHandler(t *testing.T) {
	tests := []struct {
		name           string
		request        ChatRequest
		expectedStatus int
		checkStream    func(t *testing.T, chunks []StreamChunk)
		setupMock      func(h *Handler)
	}{
		{
			name: "successful stream request",
			request: ChatRequest{
				Messages: []Message{
					{Role: "user", Content: "Stream this response"},
				},
				Stream: true,
			},
			expectedStatus: http.StatusOK,
			checkStream: func(t *testing.T, chunks []StreamChunk) {
				assert.Greater(t, len(chunks), 0)
				// Check that last chunk has done=true
				lastChunk := chunks[len(chunks)-1]
				assert.True(t, lastChunk.Done)
			},
		},
		{
			name: "stream with invalid request",
			request: ChatRequest{
				Messages: []Message{},
				Stream:   true,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(nil)
			if tt.setupMock != nil {
				tt.setupMock(handler)
			}

			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/stream", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.HandleStream(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedStatus == http.StatusOK && tt.checkStream != nil {
				// Parse SSE stream
				chunks := parseSSEStream(t, rec.Body.String())
				tt.checkStream(t, chunks)
			}
		})
	}
}

func TestToolsHandler(t *testing.T) {
	tests := []struct {
		name           string
		request        ToolRequest
		expectedStatus int
		checkResponse  func(t *testing.T, resp ToolResponse)
		setupMock      func(h *Handler)
	}{
		{
			name: "successful file read tool",
			request: ToolRequest{
				Name: "file_read",
				Arguments: map[string]interface{}{
					"path": "test.txt",
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp ToolResponse) {
				assert.NotEmpty(t, resp.ID)
				assert.NotEmpty(t, resp.Result)
				assert.Empty(t, resp.Error)
			},
		},
		{
			name: "unknown tool",
			request: ToolRequest{
				Name:      "unknown_tool",
				Arguments: map[string]interface{}{},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "tool execution error",
			request: ToolRequest{
				Name: "file_read",
				Arguments: map[string]interface{}{
					"path": "/etc/passwd", // Should fail due to security
				},
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, resp ToolResponse) {
				assert.NotEmpty(t, resp.Error)
			},
		},
		{
			name: "missing required arguments",
			request: ToolRequest{
				Name:      "file_read",
				Arguments: map[string]interface{}{},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(nil)
			if tt.setupMock != nil {
				tt.setupMock(handler)
			}

			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/tools", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.HandleTools(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if rec.Code == http.StatusOK || rec.Code == http.StatusForbidden {
				var resp ToolResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			}
		})
	}
}

func TestHistoryHandler(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		expectedStatus int
		checkResponse  func(t *testing.T, resp HistoryResponse)
		setupMock      func(h *Handler)
	}{
		{
			name:           "get all conversations",
			queryParams:    map[string]string{},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp HistoryResponse) {
				assert.GreaterOrEqual(t, len(resp.Conversations), 0)
				assert.Equal(t, 1, resp.Page)
				assert.Equal(t, 20, resp.PerPage) // Default
			},
		},
		{
			name: "get specific page",
			queryParams: map[string]string{
				"page":     "2",
				"per_page": "10",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp HistoryResponse) {
				assert.Equal(t, 2, resp.Page)
				assert.Equal(t, 10, resp.PerPage)
			},
		},
		{
			name: "invalid page number",
			queryParams: map[string]string{
				"page": "invalid",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "get conversation by ID",
			queryParams: map[string]string{
				"id": "conv-123",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp HistoryResponse) {
				assert.Len(t, resp.Conversations, 1)
				assert.Equal(t, "conv-123", resp.Conversations[0].ID)
				assert.NotEmpty(t, resp.Conversations[0].Messages)
			},
		},
		{
			name: "conversation not found",
			queryParams: map[string]string{
				"id": "non-existent",
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(nil)
			if tt.setupMock != nil {
				tt.setupMock(handler)
			}

			req := httptest.NewRequest(http.MethodGet, "/history", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			rec := httptest.NewRecorder()

			handler.HandleHistory(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var resp HistoryResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestHandlerMiddleware(t *testing.T) {
	t.Run("content type validation", func(t *testing.T) {
		handler := NewHandler(nil)

		// Test missing content-type
		req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader("{}"))
		rec := httptest.NewRecorder()

		handler.HandleChat(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Content-Type")
	})

	t.Run("method not allowed", func(t *testing.T) {
		handler := NewHandler(nil)

		// Test GET on POST-only endpoint
		req := httptest.NewRequest(http.MethodGet, "/chat", nil)
		rec := httptest.NewRecorder()

		handler.HandleChat(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})

	t.Run("request size limit", func(t *testing.T) {
		handler := NewHandler(nil)

		// Create a very large request body (> 10MB)
		largeBody := make([]byte, 11*1024*1024)
		req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(largeBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.HandleChat(rec, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})
}

// Helper function to parse SSE stream
func parseSSEStream(t *testing.T, stream string) []StreamChunk {
	var chunks []StreamChunk
	lines := strings.Split(stream, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				continue
			}

			var chunk StreamChunk
			err := json.Unmarshal([]byte(data), &chunk)
			require.NoError(t, err)
			chunks = append(chunks, chunk)
		}
	}

	return chunks
}

// Test concurrent requests
func TestConcurrentRequests(t *testing.T) {
	handler := NewHandler(nil)

	// Make 100 concurrent requests
	concurrency := 100
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(i int) {
			defer func() { done <- true }()

			req := ChatRequest{
				Messages: []Message{
					{Role: "user", Content: fmt.Sprintf("Request %d", i)},
				},
			}

			body, _ := json.Marshal(req)
			httpReq := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(body))
			httpReq.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.HandleChat(rec, httpReq)

			assert.Equal(t, http.StatusOK, rec.Code)
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
}
