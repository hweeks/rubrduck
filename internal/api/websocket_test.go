package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type  string      `json:"type"`
	ID    string      `json:"id,omitempty"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// WebSocketConfig holds WebSocket handler configuration
type WebSocketConfig struct {
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	MaxMessageSize     int64
	HeartbeatInterval  time.Duration
	RequireAuth        bool
	AuthFunc           func(*http.Request) (string, error)
	MaxConnections     int
	EnableReconnection bool
	SessionTimeout     time.Duration
	RateLimitPerSecond int
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	config      WebSocketConfig
	connections sync.Map
	sessions    sync.Map
}

func NewWebSocketHandler(config WebSocketConfig) *WebSocketHandler {
	// Set defaults
	if config.MaxMessageSize == 0 {
		config.MaxMessageSize = 1024 * 1024 // 1MB default
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 60 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 10 * time.Second
	}

	return &WebSocketHandler{
		config: config,
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check if it's a WebSocket upgrade request
	if r.Header.Get("Upgrade") != "websocket" {
		http.Error(w, "Not a WebSocket request", http.StatusBadRequest)
		return
	}

	// Check authentication if required
	if h.config.RequireAuth && h.config.AuthFunc != nil {
		_, err := h.config.AuthFunc(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Check max connections
	var connCount int
	h.connections.Range(func(_, _ interface{}) bool {
		connCount++
		return true
	})

	if h.config.MaxConnections > 0 && connCount >= h.config.MaxConnections {
		http.Error(w, "Too many connections", http.StatusServiceUnavailable)
		return
	}

	// In a real implementation, we would upgrade the connection here
	// For testing, we just return switching protocols status
	w.Header().Set("Upgrade", "websocket")
	w.Header().Set("Connection", "Upgrade")
	w.WriteHeader(http.StatusSwitchingProtocols)
}

// Tests start here

func TestWebSocketHandlerConfiguration(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		handler := NewWebSocketHandler(WebSocketConfig{})

		assert.Equal(t, int64(1024*1024), handler.config.MaxMessageSize)
		assert.Equal(t, 60*time.Second, handler.config.ReadTimeout)
		assert.Equal(t, 10*time.Second, handler.config.WriteTimeout)
	})

	t.Run("custom configuration", func(t *testing.T) {
		config := WebSocketConfig{
			ReadTimeout:        30 * time.Second,
			WriteTimeout:       5 * time.Second,
			MaxMessageSize:     2 * 1024 * 1024,
			HeartbeatInterval:  15 * time.Second,
			MaxConnections:     100,
			RateLimitPerSecond: 10,
		}

		handler := NewWebSocketHandler(config)

		assert.Equal(t, 30*time.Second, handler.config.ReadTimeout)
		assert.Equal(t, 5*time.Second, handler.config.WriteTimeout)
		assert.Equal(t, int64(2*1024*1024), handler.config.MaxMessageSize)
		assert.Equal(t, 15*time.Second, handler.config.HeartbeatInterval)
		assert.Equal(t, 100, handler.config.MaxConnections)
		assert.Equal(t, 10, handler.config.RateLimitPerSecond)
	})
}

func TestWebSocketUpgradeValidation(t *testing.T) {
	handler := NewWebSocketHandler(WebSocketConfig{})

	t.Run("valid upgrade request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Connection", "upgrade")

		rec := httptest.NewRecorder()
		handler.HandleWebSocket(rec, req)

		assert.Equal(t, http.StatusSwitchingProtocols, rec.Code)
		assert.Equal(t, "websocket", rec.Header().Get("Upgrade"))
		assert.Equal(t, "Upgrade", rec.Header().Get("Connection"))
	})

	t.Run("non-websocket request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)

		rec := httptest.NewRecorder()
		handler.HandleWebSocket(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Not a WebSocket request")
	})
}

func TestWebSocketAuthentication(t *testing.T) {
	t.Run("authentication required", func(t *testing.T) {
		handler := NewWebSocketHandler(WebSocketConfig{
			RequireAuth: true,
			AuthFunc: func(r *http.Request) (string, error) {
				token := r.Header.Get("Authorization")
				if token == "Bearer valid-token" {
					return "user-123", nil
				}
				return "", fmt.Errorf("invalid token")
			},
		})

		// Valid auth
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Authorization", "Bearer valid-token")

		rec := httptest.NewRecorder()
		handler.HandleWebSocket(rec, req)

		assert.Equal(t, http.StatusSwitchingProtocols, rec.Code)

		// Invalid auth
		req = httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Authorization", "Bearer invalid-token")

		rec = httptest.NewRecorder()
		handler.HandleWebSocket(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("authentication not required", func(t *testing.T) {
		handler := NewWebSocketHandler(WebSocketConfig{
			RequireAuth: false,
		})

		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.Header.Set("Upgrade", "websocket")

		rec := httptest.NewRecorder()
		handler.HandleWebSocket(rec, req)

		assert.Equal(t, http.StatusSwitchingProtocols, rec.Code)
	})
}

func TestWebSocketConnectionLimit(t *testing.T) {
	handler := NewWebSocketHandler(WebSocketConfig{
		MaxConnections: 2,
	})

	// Simulate adding connections
	handler.connections.Store("conn1", true)
	handler.connections.Store("conn2", true)

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Upgrade", "websocket")

	rec := httptest.NewRecorder()
	handler.HandleWebSocket(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "Too many connections")

	// Remove a connection
	handler.connections.Delete("conn1")

	rec = httptest.NewRecorder()
	handler.HandleWebSocket(rec, req)

	assert.Equal(t, http.StatusSwitchingProtocols, rec.Code)
}

func TestWebSocketMessageStructures(t *testing.T) {
	t.Run("message serialization", func(t *testing.T) {
		msg := WebSocketMessage{
			Type: "chat",
			ID:   "msg-123",
			Data: map[string]interface{}{
				"content": "Hello, world!",
			},
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		var decoded WebSocketMessage
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, msg.Type, decoded.Type)
		assert.Equal(t, msg.ID, decoded.ID)
	})

	t.Run("error message", func(t *testing.T) {
		msg := WebSocketMessage{
			Type:  "error",
			Error: "Something went wrong",
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		var decoded WebSocketMessage
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "error", decoded.Type)
		assert.Equal(t, "Something went wrong", decoded.Error)
	})
}

func TestWebSocketSessionManagement(t *testing.T) {
	handler := NewWebSocketHandler(WebSocketConfig{
		EnableReconnection: true,
		SessionTimeout:     5 * time.Second,
	})

	t.Run("session creation", func(t *testing.T) {
		sessionID := "session-123"
		sessionData := map[string]interface{}{
			"user_id": "user-456",
			"created": time.Now(),
		}

		handler.sessions.Store(sessionID, sessionData)

		// Verify session exists
		data, exists := handler.sessions.Load(sessionID)
		assert.True(t, exists)
		assert.NotNil(t, data)
	})

	t.Run("session retrieval", func(t *testing.T) {
		sessionID := "session-456"
		handler.sessions.Store(sessionID, map[string]interface{}{
			"messages": []string{"msg1", "msg2"},
		})

		data, exists := handler.sessions.Load(sessionID)
		assert.True(t, exists)

		sessionData := data.(map[string]interface{})
		messages := sessionData["messages"].([]string)
		assert.Len(t, messages, 2)
	})
}

func TestWebSocketConcurrency(t *testing.T) {
	handler := NewWebSocketHandler(WebSocketConfig{
		MaxConnections: 100,
	})

	t.Run("concurrent connection handling", func(t *testing.T) {
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		// Simulate 50 concurrent connection attempts
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Add connection
				connID := fmt.Sprintf("conn-%d", id)
				handler.connections.Store(connID, true)

				// Simulate some work
				time.Sleep(10 * time.Millisecond)

				// Remove connection
				handler.connections.Delete(connID)

				mu.Lock()
				successCount++
				mu.Unlock()
			}(i)
		}

		wg.Wait()
		assert.Equal(t, 50, successCount)
	})

	t.Run("concurrent session access", func(t *testing.T) {
		var wg sync.WaitGroup
		sessionID := "shared-session"

		// Initialize session
		handler.sessions.Store(sessionID, &sync.Map{})

		// Multiple goroutines accessing the same session
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				sessionData, _ := handler.sessions.Load(sessionID)
				session := sessionData.(*sync.Map)

				// Add data to session
				key := fmt.Sprintf("data-%d", id)
				session.Store(key, fmt.Sprintf("value-%d", id))

				// Read data
				val, exists := session.Load(key)
				assert.True(t, exists)
				assert.Equal(t, fmt.Sprintf("value-%d", id), val)
			}(i)
		}

		wg.Wait()

		// Verify all data was stored
		sessionData, _ := handler.sessions.Load(sessionID)
		session := sessionData.(*sync.Map)

		count := 0
		session.Range(func(_, _ interface{}) bool {
			count++
			return true
		})
		assert.Equal(t, 10, count)
	})
}

func TestWebSocketRateLimiting(t *testing.T) {
	handler := NewWebSocketHandler(WebSocketConfig{
		RateLimitPerSecond: 5,
	})

	t.Run("rate limit configuration", func(t *testing.T) {
		assert.Equal(t, 5, handler.config.RateLimitPerSecond)
	})

	t.Run("rate limit tracking", func(t *testing.T) {
		// In a real implementation, we would track message rates
		// For testing, we verify the configuration is set
		assert.Greater(t, handler.config.RateLimitPerSecond, 0)
	})
}
