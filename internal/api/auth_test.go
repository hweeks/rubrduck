package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func(req *http.Request)
		authConfig     AuthConfig
		expectedStatus int
		expectedUser   string
		checkResponse  func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "valid bearer token",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer valid-token-123")
			},
			authConfig: AuthConfig{
				Type:    "bearer",
				Enabled: true,
			},
			expectedStatus: http.StatusOK,
			expectedUser:   "user-123",
		},
		{
			name: "valid API key in header",
			setupRequest: func(req *http.Request) {
				req.Header.Set("X-API-Key", "sk-valid-api-key")
			},
			authConfig: AuthConfig{
				Type:    "apikey",
				Enabled: true,
			},
			expectedStatus: http.StatusOK,
			expectedUser:   "api-user-456",
		},
		{
			name: "valid API key in query param",
			setupRequest: func(req *http.Request) {
				q := req.URL.Query()
				q.Add("api_key", "sk-valid-api-key")
				req.URL.RawQuery = q.Encode()
			},
			authConfig: AuthConfig{
				Type:                "apikey",
				Enabled:             true,
				AllowQueryParamAuth: true,
			},
			expectedStatus: http.StatusOK,
			expectedUser:   "api-user-456",
		},
		{
			name:         "missing authorization",
			setupRequest: func(req *http.Request) {},
			authConfig: AuthConfig{
				Type:    "bearer",
				Enabled: true,
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Contains(t, rec.Body.String(), "authorization required")
			},
		},
		{
			name: "invalid bearer token format",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "InvalidFormat token")
			},
			authConfig: AuthConfig{
				Type:    "bearer",
				Enabled: true,
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Contains(t, rec.Body.String(), "invalid authorization format")
			},
		},
		{
			name: "expired token",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer expired-token")
			},
			authConfig: AuthConfig{
				Type:    "bearer",
				Enabled: true,
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Contains(t, rec.Body.String(), "token expired")
			},
		},
		{
			name: "revoked token",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer revoked-token")
			},
			authConfig: AuthConfig{
				Type:    "bearer",
				Enabled: true,
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Contains(t, rec.Body.String(), "token revoked")
			},
		},
		{
			name: "auth disabled",
			setupRequest: func(req *http.Request) {
				// No auth header
			},
			authConfig: AuthConfig{
				Type:    "bearer",
				Enabled: false,
			},
			expectedStatus: http.StatusOK,
			expectedUser:   "", // No user when auth is disabled
		},
		{
			name: "API key query param disabled",
			setupRequest: func(req *http.Request) {
				q := req.URL.Query()
				q.Add("api_key", "sk-valid-api-key")
				req.URL.RawQuery = q.Encode()
			},
			authConfig: AuthConfig{
				Type:                "apikey",
				Enabled:             true,
				AllowQueryParamAuth: false, // Query param auth disabled
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that will be protected by auth
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if user context is set
				user := GetUserFromContext(r.Context())
				if tt.expectedUser != "" {
					assert.Equal(t, tt.expectedUser, user.ID)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Create auth middleware
			authMiddleware := NewAuthMiddleware(tt.authConfig)
			protectedHandler := authMiddleware.Wrap(testHandler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}

			// Execute request
			rec := httptest.NewRecorder()
			protectedHandler.ServeHTTP(rec, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

func TestTokenValidation(t *testing.T) {
	validator := NewTokenValidator(TokenValidatorConfig{
		Secret:            "test-secret",
		TokenExpiration:   24 * time.Hour,
		RefreshExpiration: 7 * 24 * time.Hour,
	})

	t.Run("generate and validate token", func(t *testing.T) {
		// Generate token
		token, err := validator.GenerateToken("user-123", map[string]interface{}{
			"role": "admin",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Validate token
		claims, err := validator.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
		assert.Equal(t, "admin", claims.Extra["role"])
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := validator.ValidateToken("invalid-token")
		assert.Error(t, err)
	})

	t.Run("token with invalid signature", func(t *testing.T) {
		// Generate token with different secret
		otherValidator := NewTokenValidator(TokenValidatorConfig{
			Secret: "different-secret",
		})
		token, err := otherValidator.GenerateToken("user-123", nil)
		require.NoError(t, err)

		// Try to validate with original validator
		_, err = validator.ValidateToken(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid signature")
	})

	t.Run("expired token", func(t *testing.T) {
		// Create validator with very short expiration
		shortValidator := NewTokenValidator(TokenValidatorConfig{
			Secret:          "test-secret",
			TokenExpiration: 1 * time.Millisecond,
		})

		token, err := shortValidator.GenerateToken("user-123", nil)
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(2 * time.Millisecond)

		_, err = shortValidator.ValidateToken(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})
}

func TestAPIKeyValidation(t *testing.T) {
	validator := NewAPIKeyValidator(APIKeyValidatorConfig{
		ValidKeys: map[string]APIKeyInfo{
			"sk-test-key-1": {
				UserID:      "user-1",
				Permissions: []string{"read", "write"},
				RateLimit:   100,
			},
			"sk-test-key-2": {
				UserID:      "user-2",
				Permissions: []string{"read"},
				RateLimit:   50,
				ExpiresAt:   time.Now().Add(24 * time.Hour),
			},
			"sk-expired-key": {
				UserID:    "user-3",
				ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
			},
		},
	})

	t.Run("valid API key", func(t *testing.T) {
		info, err := validator.ValidateAPIKey("sk-test-key-1")
		require.NoError(t, err)
		assert.Equal(t, "user-1", info.UserID)
		assert.Equal(t, []string{"read", "write"}, info.Permissions)
		assert.Equal(t, 100, info.RateLimit)
	})

	t.Run("invalid API key", func(t *testing.T) {
		_, err := validator.ValidateAPIKey("sk-invalid-key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API key")
	})

	t.Run("expired API key", func(t *testing.T) {
		_, err := validator.ValidateAPIKey("sk-expired-key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key expired")
	})

	t.Run("check permissions", func(t *testing.T) {
		// User with read/write permissions
		info1, _ := validator.ValidateAPIKey("sk-test-key-1")
		assert.True(t, info1.HasPermission("read"))
		assert.True(t, info1.HasPermission("write"))
		assert.False(t, info1.HasPermission("delete"))

		// User with only read permission
		info2, _ := validator.ValidateAPIKey("sk-test-key-2")
		assert.True(t, info2.HasPermission("read"))
		assert.False(t, info2.HasPermission("write"))
	})
}

func TestAuthRateLimiting(t *testing.T) {
	// Test that auth middleware can integrate with rate limiting
	authConfig := AuthConfig{
		Type:               "apikey",
		Enabled:            true,
		EnableRateLimiting: true,
	}

	rateLimiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 10,
		BurstSize:         5,
	})

	authMiddleware := NewAuthMiddleware(authConfig)
	authMiddleware.SetRateLimiter(rateLimiter)

	handler := authMiddleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make requests with API key that has rate limit
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "sk-rate-limited-key")

	// Should succeed for first few requests
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Should eventually get rate limited
	var rateLimited bool
	for i := 0; i < 10; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			rateLimited = true
			break
		}
	}
	assert.True(t, rateLimited, "Expected to be rate limited")
}

func TestCORSWithAuth(t *testing.T) {
	// Test that CORS preflight requests bypass auth
	authConfig := AuthConfig{
		Type:    "bearer",
		Enabled: true,
	}

	authMiddleware := NewAuthMiddleware(authConfig)
	handler := authMiddleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Preflight request should bypass auth
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should return OK without auth
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMultipleAuthMethods(t *testing.T) {
	// Test supporting multiple auth methods simultaneously
	authConfig := AuthConfig{
		Type:    "multiple",
		Enabled: true,
		Methods: []string{"bearer", "apikey"},
	}

	authMiddleware := NewAuthMiddleware(authConfig)
	handler := authMiddleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(user.ID))
	}))

	t.Run("auth with bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-token")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "bearer-user", rec.Body.String())
	})

	t.Run("auth with API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "sk-valid-key")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "apikey-user", rec.Body.String())
	})

	t.Run("no auth fails", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
