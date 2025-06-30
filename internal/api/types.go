package api

import (
	"net/http"
	"time"
)

// ServerConfig holds the configuration for the API server
type ServerConfig struct {
	Port               int
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	ShutdownTimeout    time.Duration
	EnableAuth         bool
	EnableRateLimiting bool
	EnableCORS         bool
	CORSAllowedOrigins []string
}

// Server represents the main API server
type Server struct {
	config   ServerConfig
	router   *http.ServeMux
	server   *http.Server
	handlers *Handler
}

// Handler holds the handlers for API endpoints
type Handler struct {
	// Add fields as needed for implementation
}

// User represents an authenticated user
type User struct {
	ID          string
	Email       string
	Permissions []string
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Type                string
	Enabled             bool
	EnableRateLimiting  bool
	AllowQueryParamAuth bool
	Methods             []string
}

// RateLimiterConfig holds rate limiter configuration
type RateLimiterConfig struct {
	RequestsPerMinute int
	BurstSize         int
	IdentifierFunc    func(*http.Request) string
	CustomLimits      map[string]RateLimit
	CleanupInterval   time.Duration
	MaxInactiveTime   time.Duration
	EnableMetrics     bool
	CustomResponse    func(http.ResponseWriter, *http.Request, RateLimitInfo)
}

// RateLimit represents rate limit settings
type RateLimit struct {
	RequestsPerMinute int
	BurstSize         int
}

// RateLimitInfo contains rate limit information
type RateLimitInfo struct {
	RetryAfter int
}

// SlidingWindowConfig holds sliding window rate limiter configuration
type SlidingWindowConfig struct {
	WindowSize      time.Duration
	MaxRequests     int
	CleanupInterval time.Duration
}

// DistributedRateLimiterConfig holds distributed rate limiter configuration
type DistributedRateLimiterConfig struct {
	RedisAddr         string
	RequestsPerMinute int
	WindowSize        time.Duration
	KeyPrefix         string
}

// TokenValidatorConfig holds token validator configuration
type TokenValidatorConfig struct {
	Secret            string
	TokenExpiration   time.Duration
	RefreshExpiration time.Duration
}

// APIKeyValidatorConfig holds API key validator configuration
type APIKeyValidatorConfig struct {
	ValidKeys map[string]APIKeyInfo
}

// APIKeyInfo holds information about an API key
type APIKeyInfo struct {
	UserID      string
	Permissions []string
	RateLimit   int
	ExpiresAt   time.Time
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID string
	Extra  map[string]interface{}
}

// RateLimiterMetrics holds rate limiter metrics
type RateLimiterMetrics struct {
	TotalRequests   int
	AllowedRequests int
	DeniedRequests  int
	UniqueKeys      int
}

// Request and Response types for handlers

// ChatRequest represents a chat API request
type ChatRequest struct {
	Messages []Message `json:"messages"`
	Model    string    `json:"model,omitempty"`
	Stream   bool      `json:"stream,omitempty"`
	ID       string    `json:"id,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents a chat API response
type ChatResponse struct {
	ID      string    `json:"id"`
	Message Message   `json:"message"`
	Created time.Time `json:"created"`
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

// ToolRequest represents a tool execution request
type ToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResponse represents a tool execution response
type ToolResponse struct {
	ID     string `json:"id"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

// HistoryResponse represents a conversation history response
type HistoryResponse struct {
	Conversations []Conversation `json:"conversations"`
	Total         int            `json:"total"`
	Page          int            `json:"page"`
	PerPage       int            `json:"per_page"`
}

// Conversation represents a conversation in history
type Conversation struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
	Messages []Message `json:"messages,omitempty"`
}

// NewHandler creates a new handler
func NewHandler(config interface{}) *Handler {
	// Implementation will be added
	return &Handler{}
}

// NewAuthMiddleware creates authentication middleware
func NewAuthMiddleware(config AuthConfig) *AuthMiddleware {
	// Implementation will be added
	return &AuthMiddleware{
		config: config,
	}
}

// AuthMiddleware implements authentication middleware
type AuthMiddleware struct {
	config      AuthConfig
	rateLimiter *RateLimiter
}

func (a *AuthMiddleware) SetRateLimiter(limiter *RateLimiter) {
	a.rateLimiter = limiter
}

// ServeHTTP makes AuthMiddleware implement http.Handler interface
func (a *AuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Implementation will be added
	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Auth logic will be implemented here
		w.WriteHeader(http.StatusOK)
	}).ServeHTTP(w, r)
}
