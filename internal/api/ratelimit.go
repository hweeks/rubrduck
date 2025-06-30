package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	// Set default burst size if not specified
	if config.BurstSize == 0 {
		config.BurstSize = config.RequestsPerMinute / 12 // 5 seconds worth of requests
		if config.BurstSize < 1 {
			config.BurstSize = 1
		}
	}

	rl := &RateLimiter{
		config:  config,
		buckets: make(map[string]*bucket),
		mu:      sync.Mutex{},
		metrics: &RateLimiterMetrics{},
	}

	// Start cleanup goroutine if configured
	if config.CleanupInterval > 0 {
		go rl.cleanupLoop()
	}

	return rl
}

// RateLimiter implements rate limiting
type RateLimiter struct {
	config  RateLimiterConfig
	buckets map[string]*bucket
	mu      sync.Mutex
	metrics *RateLimiterMetrics
}

type bucket struct {
	tokens   float64
	lastFill time.Time
	lastUsed time.Time
}

// Allow checks if a request is allowed
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics.TotalRequests++

	// Get rate limit for this key
	limit := r.config.RequestsPerMinute
	burst := r.config.BurstSize

	// Check for custom limits
	if r.config.CustomLimits != nil {
		if custom, exists := r.config.CustomLimits[key]; exists {
			limit = custom.RequestsPerMinute
			burst = custom.BurstSize
		}
	}

	// Get or create bucket
	b, exists := r.buckets[key]
	if !exists {
		b = &bucket{
			tokens:   float64(burst),
			lastFill: time.Now(),
			lastUsed: time.Now(),
		}
		r.buckets[key] = b
	}

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(b.lastFill).Seconds()
	tokensToAdd := elapsed * (float64(limit) / 60.0)
	b.tokens = min(float64(burst), b.tokens+tokensToAdd)
	b.lastFill = now
	b.lastUsed = now

	// Check if we have tokens
	if b.tokens >= 1 {
		b.tokens--
		r.metrics.AllowedRequests++
		return true
	}

	r.metrics.DeniedRequests++
	return false
}

// Reset resets the rate limiter
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.buckets = make(map[string]*bucket)
	r.metrics = &RateLimiterMetrics{}
}

// ActiveKeys returns the number of active keys
func (r *RateLimiter) ActiveKeys() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.buckets)
}

// GetMetrics returns rate limiter metrics
func (r *RateLimiter) GetMetrics() *RateLimiterMetrics {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics.UniqueKeys = len(r.buckets)
	return r.metrics
}

// cleanupLoop periodically cleans up inactive buckets
func (r *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(r.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for key, b := range r.buckets {
			if r.config.MaxInactiveTime > 0 && now.Sub(b.lastUsed) > r.config.MaxInactiveTime {
				delete(r.buckets, key)
			}
		}
		r.mu.Unlock()
	}
}

// RateLimitMiddleware creates rate limiting middleware
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get identifier
			identifier := r.RemoteAddr
			if limiter.config.IdentifierFunc != nil {
				identifier = limiter.config.IdentifierFunc(r)
			}

			// Check rate limit
			if !limiter.Allow(identifier) {
				// Set rate limit headers
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.config.RequestsPerMinute))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
				w.Header().Set("Retry-After", "60")

				// Custom response if configured
				if limiter.config.CustomResponse != nil {
					info := RateLimitInfo{RetryAfter: 60}
					limiter.config.CustomResponse(w, r, info)
					return
				}

				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// NewSlidingWindowRateLimiter creates a sliding window rate limiter
func NewSlidingWindowRateLimiter(config SlidingWindowConfig) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		config:   config,
		windows:  make(map[string]*slidingWindow),
		mu:       sync.Mutex{},
		timeFunc: time.Now,
	}
}

// SlidingWindowRateLimiter implements sliding window rate limiting
type SlidingWindowRateLimiter struct {
	config   SlidingWindowConfig
	windows  map[string]*slidingWindow
	mu       sync.Mutex
	timeFunc func() time.Time
}

type slidingWindow struct {
	requests []time.Time
}

// Allow checks if a request is allowed
func (r *SlidingWindowRateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.timeFunc()
	windowStart := now.Add(-r.config.WindowSize)

	// Get or create window
	window, exists := r.windows[key]
	if !exists {
		window = &slidingWindow{
			requests: make([]time.Time, 0),
		}
		r.windows[key] = window
	}

	// Remove old requests
	validRequests := make([]time.Time, 0)
	for _, reqTime := range window.requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}
	window.requests = validRequests

	// Check if we can add another request
	if len(window.requests) < r.config.MaxRequests {
		window.requests = append(window.requests, now)
		return true
	}

	return false
}

// SetTimeFunc sets the time function for testing
func (r *SlidingWindowRateLimiter) SetTimeFunc(f func() time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.timeFunc = f
}

// NewDistributedRateLimiter creates a distributed rate limiter
func NewDistributedRateLimiter(config DistributedRateLimiterConfig) (*DistributedRateLimiter, error) {
	// For testing, we'll just return a simple mock
	// In production, this would connect to Redis
	return &DistributedRateLimiter{
		config: config,
		counts: make(map[string]int),
		mu:     sync.Mutex{},
	}, nil
}

// DistributedRateLimiter implements distributed rate limiting
type DistributedRateLimiter struct {
	config DistributedRateLimiterConfig
	counts map[string]int
	mu     sync.Mutex
}

// Allow checks if a request is allowed
func (r *DistributedRateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	fullKey := r.config.KeyPrefix + key
	count := r.counts[fullKey]

	if count < r.config.RequestsPerMinute {
		r.counts[fullKey]++
		return true
	}

	return false
}

// Helper function
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
