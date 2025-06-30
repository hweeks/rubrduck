package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	tests := []struct {
		name            string
		config          RateLimiterConfig
		requests        int
		requestDelay    time.Duration
		expectedAllowed int
		expectedDenied  int
	}{
		{
			name: "basic rate limiting",
			config: RateLimiterConfig{
				RequestsPerMinute: 60,
				BurstSize:         10,
			},
			requests:        15,
			requestDelay:    0,
			expectedAllowed: 10, // Burst size
			expectedDenied:  5,
		},
		{
			name: "rate limiting with delay",
			config: RateLimiterConfig{
				RequestsPerMinute: 60, // 1 per second
				BurstSize:         5,
			},
			requests:        10,
			requestDelay:    100 * time.Millisecond, // Should allow ~1 request per 100ms
			expectedAllowed: 10,                     // All should pass with delay
			expectedDenied:  0,
		},
		{
			name: "strict rate limiting",
			config: RateLimiterConfig{
				RequestsPerMinute: 10,
				BurstSize:         1,
			},
			requests:        5,
			requestDelay:    0,
			expectedAllowed: 1,
			expectedDenied:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewRateLimiter(tt.config)

			allowed := 0
			denied := 0

			for i := 0; i < tt.requests; i++ {
				if tt.requestDelay > 0 {
					time.Sleep(tt.requestDelay)
				}

				if limiter.Allow("test-key") {
					allowed++
				} else {
					denied++
				}
			}

			assert.Equal(t, tt.expectedAllowed, allowed, "Allowed requests mismatch")
			assert.Equal(t, tt.expectedDenied, denied, "Denied requests mismatch")
		})
	}
}

func TestRateLimiterMiddleware(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerMinute: 10,
		BurstSize:         5,
		IdentifierFunc: func(r *http.Request) string {
			// Use client IP as identifier
			return r.RemoteAddr
		},
	}

	limiter := NewRateLimiter(config)
	middleware := RateLimitMiddleware(limiter)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	t.Run("allows requests within limit", func(t *testing.T) {
		// Reset limiter
		limiter.Reset()

		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "OK", rec.Body.String())
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		// Continue from previous test - should be at limit
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Contains(t, rec.Body.String(), "rate limit exceeded")

		// Check rate limit headers
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
		assert.NotEmpty(t, rec.Header().Get("Retry-After"))
	})

	t.Run("different identifiers have separate limits", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.2:1234" // Different IP
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestRateLimiterWithAPIKey(t *testing.T) {
	// Test rate limiting based on API key instead of IP
	config := RateLimiterConfig{
		RequestsPerMinute: 10,
		BurstSize:         5,
		IdentifierFunc: func(r *http.Request) string {
			// Use API key as identifier
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "" {
				return "apikey:" + apiKey
			}
			return r.RemoteAddr
		},
		// Custom limits per API key
		CustomLimits: map[string]RateLimit{
			"apikey:sk-premium": {
				RequestsPerMinute: 100,
				BurstSize:         20,
			},
			"apikey:sk-basic": {
				RequestsPerMinute: 20,
				BurstSize:         5,
			},
		},
	}

	limiter := NewRateLimiter(config)
	middleware := RateLimitMiddleware(limiter)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("premium API key gets higher limit", func(t *testing.T) {
		limiter.Reset()

		// Premium key should allow 20 burst requests
		for i := 0; i < 20; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-API-Key", "sk-premium")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "Request %d should succeed", i+1)
		}

		// 21st request should be rate limited
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "sk-premium")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})

	t.Run("basic API key gets lower limit", func(t *testing.T) {
		limiter.Reset()

		// Basic key should allow only 5 burst requests
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-API-Key", "sk-basic")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		}

		// 6th request should be rate limited
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "sk-basic")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})
}

func TestSlidingWindowRateLimiter(t *testing.T) {
	config := SlidingWindowConfig{
		WindowSize:      1 * time.Minute,
		MaxRequests:     10,
		CleanupInterval: 10 * time.Second,
	}

	limiter := NewSlidingWindowRateLimiter(config)

	t.Run("allows requests within window", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			allowed := limiter.Allow("test-key")
			assert.True(t, allowed, "Request %d should be allowed", i+1)
		}

		// 11th request should be denied
		allowed := limiter.Allow("test-key")
		assert.False(t, allowed, "11th request should be denied")
	})

	t.Run("resets after window passes", func(t *testing.T) {
		// Mock time to simulate window passing
		limiter.SetTimeFunc(func() time.Time {
			return time.Now().Add(2 * time.Minute)
		})

		// Should allow requests again
		allowed := limiter.Allow("test-key")
		assert.True(t, allowed, "Should allow request after window reset")
	})
}

func TestConcurrentRateLimiting(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerMinute: 100,
		BurstSize:         50,
	}

	limiter := NewRateLimiter(config)

	// Test concurrent access
	var wg sync.WaitGroup
	allowed := 0
	denied := 0
	var mu sync.Mutex

	// Launch 100 concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			key := fmt.Sprintf("user-%d", id%10) // 10 different users

			if limiter.Allow(key) {
				mu.Lock()
				allowed++
				mu.Unlock()
			} else {
				mu.Lock()
				denied++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 100, allowed+denied, "Total requests should be 100")
	// Each user should get at most 50 requests (burst size)
	assert.LessOrEqual(t, allowed, 50*10, "Allowed requests should not exceed total burst capacity")
}

func TestRateLimiterCleanup(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerMinute: 10,
		BurstSize:         5,
		CleanupInterval:   100 * time.Millisecond,
		MaxInactiveTime:   200 * time.Millisecond,
	}

	limiter := NewRateLimiter(config)

	// Use the limiter
	limiter.Allow("test-key-1")
	limiter.Allow("test-key-2")

	// Verify keys exist
	assert.Equal(t, 2, limiter.ActiveKeys())

	// Wait for cleanup
	time.Sleep(300 * time.Millisecond)

	// Keys should be cleaned up
	assert.Equal(t, 0, limiter.ActiveKeys())
}

func TestDistributedRateLimiting(t *testing.T) {
	// Test rate limiting with Redis backend for distributed systems
	config := DistributedRateLimiterConfig{
		RedisAddr:         "localhost:6379",
		RequestsPerMinute: 100,
		WindowSize:        1 * time.Minute,
		KeyPrefix:         "ratelimit:",
	}

	// Skip if Redis is not available
	limiter, err := NewDistributedRateLimiter(config)
	if err != nil {
		t.Skip("Redis not available, skipping distributed rate limiting tests")
	}

	t.Run("distributed rate limiting", func(t *testing.T) {
		key := "distributed-test-key"

		// Simulate multiple instances
		allowed := 0
		for i := 0; i < 150; i++ {
			if limiter.Allow(key) {
				allowed++
			}
		}

		// Should respect global limit
		assert.LessOrEqual(t, allowed, 100, "Should not exceed global limit")
	})
}

func TestRateLimiterMetrics(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerMinute: 10,
		BurstSize:         5,
		EnableMetrics:     true,
	}

	limiter := NewRateLimiter(config)

	// Make some requests
	for i := 0; i < 10; i++ {
		limiter.Allow("test-key")
	}

	// Get metrics
	metrics := limiter.GetMetrics()

	assert.Equal(t, 10, metrics.TotalRequests)
	assert.Equal(t, 5, metrics.AllowedRequests)
	assert.Equal(t, 5, metrics.DeniedRequests)
	assert.Equal(t, 1, metrics.UniqueKeys)
}

func TestCustomRateLimitResponse(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerMinute: 1,
		BurstSize:         1,
		CustomResponse: func(w http.ResponseWriter, r *http.Request, info RateLimitInfo) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":"Custom rate limit message","retry_after":%d}`, info.RetryAfter)
		},
	}

	limiter := NewRateLimiter(config)
	middleware := RateLimitMiddleware(limiter)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request should succeed
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Second request should get custom response
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Contains(t, rec.Body.String(), "Custom rate limit message")
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}
