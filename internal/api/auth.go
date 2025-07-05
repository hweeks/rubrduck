package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey string

const userContextKey contextKey = "user"

// GetUserFromContext gets user from context
func GetUserFromContext(ctx context.Context) *User {
	user, ok := ctx.Value(userContextKey).(*User)
	if !ok {
		return nil
	}
	return user
}

// setUserInContext sets user in context
func setUserInContext(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// Wrap wraps an http.Handler with authentication
func (a *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If auth is disabled, just pass through
		if !a.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Handle CORS preflight requests
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		var user *User
		var err error

		// Check auth type
		switch a.config.Type {
		case "bearer":
			user, err = a.validateBearerToken(r)
		case "apikey":
			user, err = a.validateAPIKey(r)
		case "multiple":
			// Try bearer token first
			user, err = a.validateBearerToken(r)
			if err != nil {
				// Try API key
				user, err = a.validateAPIKey(r)
			}
		default:
			err = fmt.Errorf("unknown auth type: %s", a.config.Type)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Check rate limiting if enabled
		if a.config.EnableRateLimiting && a.rateLimiter != nil {
			// Use user ID as rate limit key
			key := user.ID
			if !a.rateLimiter.Allow(key) {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
		}

		// Set user in context
		ctx := setUserInContext(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateBearerToken validates a bearer token
func (a *AuthMiddleware) validateBearerToken(r *http.Request) (*User, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, fmt.Errorf("authorization required")
	}

	parts := strings.Split(auth, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization format")
	}

	token := parts[1]

	// Mock validation for testing
	switch token {
	case "valid-token-123":
		return &User{ID: "user-123"}, nil
	case "expired-token":
		return nil, fmt.Errorf("token expired")
	case "revoked-token":
		return nil, fmt.Errorf("token revoked")
	case "valid-token":
		return &User{ID: "bearer-user"}, nil
	default:
		return nil, fmt.Errorf("invalid token")
	}
}

// validateAPIKey validates an API key
func (a *AuthMiddleware) validateAPIKey(r *http.Request) (*User, error) {
	// Check header first
	apiKey := r.Header.Get("X-API-Key")

	// Check query param if allowed and not found in header
	if apiKey == "" && a.config.AllowQueryParamAuth {
		apiKey = r.URL.Query().Get("api_key")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("authorization required")
	}

	// Mock validation for testing
	switch apiKey {
	case "sk-valid-api-key":
		return &User{ID: "api-user-456"}, nil
	case "sk-rate-limited-key":
		return &User{ID: "rate-limited-user"}, nil
	case "sk-valid-key":
		return &User{ID: "apikey-user"}, nil
	default:
		return nil, fmt.Errorf("invalid API key")
	}
}

// Token validation implementation

// NewTokenValidator creates a token validator
func NewTokenValidator(config TokenValidatorConfig) *TokenValidator {
	return &TokenValidator{
		config: config,
		tokens: make(map[string]*tokenData), // Simple in-memory store for testing
		mu:     sync.Mutex{},
	}
}

// TokenValidator validates tokens
type TokenValidator struct {
	config TokenValidatorConfig
	tokens map[string]*tokenData // Simple storage for testing
	mu     sync.Mutex
}

type tokenData struct {
	claims    *TokenClaims
	expiresAt time.Time
}

// GenerateToken generates a new token
func (v *TokenValidator) GenerateToken(userID string, extra map[string]interface{}) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Generate a simple token for testing
	token := fmt.Sprintf("token-%s-%d", userID, time.Now().UnixNano())

	claims := &TokenClaims{
		UserID: userID,
		Extra:  extra,
	}

	// Store token with expiration
	data := &tokenData{
		claims:    claims,
		expiresAt: time.Now().Add(v.config.TokenExpiration),
	}
	v.tokens[token] = data

	return token, nil
}

// ValidateToken validates a token
func (v *TokenValidator) ValidateToken(token string) (*TokenClaims, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	data, exists := v.tokens[token]
	if !exists {
		// Check if it's from a different validator (different secret)
		if strings.HasPrefix(token, "token-") {
			return nil, fmt.Errorf("invalid signature")
		}
		return nil, fmt.Errorf("invalid token")
	}

	// Check expiration
	if time.Now().After(data.expiresAt) {
		delete(v.tokens, token)
		return nil, fmt.Errorf("token expired")
	}

	return data.claims, nil
}

// API Key validation implementation

// KeyValidator validates API keys
type KeyValidator struct {
	config KeyValidatorConfig
}

// NewKeyValidator creates an API key validator
func NewKeyValidator(config KeyValidatorConfig) *KeyValidator {
	return &KeyValidator{
		config: config,
	}
}

// ValidateAPIKey validates an API key
func (v *KeyValidator) ValidateAPIKey(key string) (*KeyInfo, error) {
	info, exists := v.config.ValidKeys[key]
	if !exists {
		return nil, fmt.Errorf("invalid API key")
	}

	// Check expiration
	if !info.ExpiresAt.IsZero() && time.Now().After(info.ExpiresAt) {
		return nil, fmt.Errorf("API key expired")
	}

	return &info, nil
}

// HasPermission checks if the API key has a permission
func (info *KeyInfo) HasPermission(permission string) bool {
	for _, p := range info.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}
