package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name    string
		config  ServerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ServerConfig{
				Port:         8080,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  60 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			config: ServerConfig{
				Port: -1,
			},
			wantErr: true,
		},
		{
			name: "default timeouts",
			config: ServerConfig{
				Port: 8080,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, server)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, server)
			}
		})
	}
}

func TestServer_Start_Stop(t *testing.T) {
	config := ServerConfig{
		Port:         0, // Use random available port
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	server, err := NewServer(config)
	require.NoError(t, err)

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	addr := server.GetAddress()
	assert.NotEmpty(t, addr)

	resp, err := http.Get("http://" + addr + "/health")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Stop server
	cancel()

	// Wait for server to stop
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Server did not stop in time")
	}
}

func TestServer_Middleware(t *testing.T) {
	config := ServerConfig{
		Port:               0,
		EnableAuth:         true,
		EnableRateLimiting: true,
		EnableCORS:         true,
		CORSAllowedOrigins: []string{"http://localhost:3000"},
	}

	server, err := NewServer(config)
	require.NoError(t, err)

	// Test that middleware is properly configured
	handler := server.setupRoutes()
	assert.NotNil(t, handler)

	// Test CORS headers
	req, err := http.NewRequest(http.MethodOptions, "/chat", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "http://localhost:3000", recorder.Header().Get("Access-Control-Allow-Origin"))
}

func TestServer_ErrorHandling(t *testing.T) {
	config := ServerConfig{
		Port: 8080,
	}

	server, err := NewServer(config)
	require.NoError(t, err)

	// Test panic recovery middleware
	handler := server.setupRoutes()

	// Add a test route that panics
	server.router.HandleFunc("/panic-test", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req, err := http.NewRequest(http.MethodGet, "/panic-test", nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestServer_GracefulShutdown(t *testing.T) {
	config := ServerConfig{
		Port:            0,
		ShutdownTimeout: 2 * time.Second,
	}

	server, err := NewServer(config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	// Start server
	started := make(chan bool)
	errCh := make(chan error, 1)
	go func() {
		started <- true
		errCh <- server.Start(ctx)
	}()

	<-started
	time.Sleep(100 * time.Millisecond)

	// Create a long-running request
	addr := server.GetAddress()
	client := &http.Client{Timeout: 5 * time.Second}

	// Start a request that will take time
	reqCtx, reqCancel := context.WithCancel(context.Background())
	defer reqCancel()

	go func() {
		req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, "http://"+addr+"/slow", nil)
		client.Do(req)
	}()

	// Give request time to start
	time.Sleep(50 * time.Millisecond)

	// Trigger shutdown
	cancel()

	// Server should wait for requests to complete
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Server did not shutdown gracefully")
	}
}
