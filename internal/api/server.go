package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Server implementation

// NewServer creates a new API server
func NewServer(config ServerConfig) (*Server, error) {
	// Validate configuration
	if config.Port < 0 || config.Port > 65535 {
		return nil, fmt.Errorf("invalid port number: %d", config.Port)
	}

	// Set default timeouts if not provided
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 30 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 120 * time.Second
	}
	if config.ShutdownTimeout == 0 {
		config.ShutdownTimeout = 30 * time.Second
	}

	s := &Server{
		config:   config,
		router:   http.NewServeMux(),
		handlers: NewHandler(nil),
	}

	// Setup HTTP server
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      s.setupRoutes(),
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	return s, nil
}

// Start starts the API server
func (s *Server) Start(ctx context.Context) error {
	// If port is 0, let the system assign one
	if s.config.Port == 0 {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return err
		}
		s.server.Addr = listener.Addr().String()

		// Start server in goroutine
		go func() {
			_ = s.server.Serve(listener)
		}()
	} else {
		// Start server in goroutine
		go func() {
			_ = s.server.ListenAndServe()
		}()
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}

// GetAddress returns the server address
func (s *Server) GetAddress() string {
	if s.server == nil {
		return ""
	}

	// If we have a specific address, return it
	if s.server.Addr != "" && s.server.Addr[0] != ':' {
		return s.server.Addr
	}

	// Otherwise, get the actual listening address
	// For testing, we'll return localhost with the port
	if s.config.Port == 0 && s.server.Addr != "" {
		return s.server.Addr
	}

	return fmt.Sprintf("localhost:%d", s.config.Port)
}

// setupRoutes sets up all the routes
func (s *Server) setupRoutes() http.Handler {
	// Create a new router to avoid conflicts
	router := http.NewServeMux()

	// Setup routes
	router.HandleFunc("/health", s.handleHealth)
	router.HandleFunc("/chat", s.handlers.HandleChat)
	router.HandleFunc("/stream", s.handlers.HandleStream)
	router.HandleFunc("/tools", s.handlers.HandleTools)
	router.HandleFunc("/history", s.handlers.HandleHistory)
	router.HandleFunc("/ws", s.handleWebSocket)

	// For testing panic recovery
	router.HandleFunc("/panic-test", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// For testing slow requests
	router.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	// Create middleware chain
	handler := http.Handler(router)

	// Add panic recovery middleware
	handler = recoveryMiddleware(handler)

	// Add CORS middleware if enabled
	if s.config.EnableCORS {
		handler = corsMiddleware(s.config.CORSAllowedOrigins)(handler)
	}

	// Add rate limiting middleware if enabled
	if s.config.EnableRateLimiting {
		handler = rateLimitingMiddleware(handler)
	}

	// Add auth middleware if enabled
	if s.config.EnableAuth {
		handler = authMiddleware(handler)
	}

	return handler
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// handleWebSocket handles WebSocket upgrade requests
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection.
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// Allow all origins for now; production code should validate the Origin header.
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// If the upgrade failed we cannot write an error using the
		// WebSocket connection, fall back to HTTP error.
		http.Error(w, "websocket upgrade failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer conn.Close()

	// Simple echo server for now. This provides basic real-time behaviour
	// and can be extended later to integrate with the AI chat engine.
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			// Client closed connection or an error occurred; terminate the loop.
			return
		}
		// Echo the message back to the client.
		if err := conn.WriteMessage(messageType, message); err != nil {
			return
		}
	}
}

// Middleware functions

// recoveryMiddleware recovers from panics
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware handles CORS
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
					break
				}
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// authMiddleware wraps the provided handler with basic bearer-token authentication.
// In production this should be replaced with project-specific configuration, but
// this implementation is sufficient for development and test purposes.
func authMiddleware(next http.Handler) http.Handler {
	mw := NewAuthMiddleware(AuthConfig{
		Type:    "bearer",
		Enabled: true,
		Methods: []string{"bearer"},
		// Rate-limiting inside the auth middleware is not enabled here because
		// we attach a global rate-limiting middleware below.
	})
	return mw.Wrap(next)
}

// rateLimitingMiddleware applies a simple token-bucket rate limiter to all
// incoming requests. For now it limits each unique remote address to 60
// requests per minute with a modest burst allowance.
func rateLimitingMiddleware(next http.Handler) http.Handler {
	limiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 60,
		EnableMetrics:     false,
		IdentifierFunc: func(r *http.Request) string {
			// Use remote address as the rate-limit key.
			return r.RemoteAddr
		},
	})
	return RateLimitMiddleware(limiter)(next)
}
