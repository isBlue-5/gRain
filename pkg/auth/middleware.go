package auth

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// contextKey type for context keys
type contextKey string

const (
	// IdentityKey is the key for storing identity in context
	IdentityKey contextKey = "identity"
)

// WithIdentity stores identity in context
func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, IdentityKey, identity)
}

// GetIdentity retrieves identity from context
func GetIdentity(ctx context.Context) (Identity, bool) {
	value := ctx.Value(IdentityKey)
	if value == nil {
		return nil, false
	}

	identity, ok := value.(Identity)
	return identity, ok
}

// RequireIdentity retrieves identity from context or returns error
func RequireIdentity(ctx context.Context) (Identity, error) {
	identity, ok := GetIdentity(ctx)
	if !ok {
		return nil, ErrNoToken
	}
	return identity, nil
}

// IsAuthenticated checks if the context has an identity
func IsAuthenticated(ctx context.Context) bool {
	_, ok := GetIdentity(ctx)
	return ok
}

// WithIdentityGin stores identity in Gin context
func WithIdentityGin(c *gin.Context, identity Identity) {
	c.Set(string(IdentityKey), identity)

	// Also store in request context for compatibility
	reqCtx := WithIdentity(c.Request.Context(), identity)
	c.Request = c.Request.WithContext(reqCtx)
}

// GetIdentityGin retrieves identity from Gin context
func GetIdentityGin(c *gin.Context) (Identity, bool) {
	value, exists := c.Get(string(IdentityKey))
	if !exists {
		return nil, false
	}

	identity, ok := value.(Identity)
	return identity, ok
}

// IsAuthenticatedGin checks if the Gin context has an identity
func IsAuthenticatedGin(c *gin.Context) bool {
	_, ok := GetIdentityGin(c)
	return ok
}

// authConfig is the configuration for AuthMiddleware
type authConfig struct {
	failureHandler func(*gin.Context, error)
	successHandler func(*gin.Context, Identity)
	skipPaths      []string
	anonymous      bool // if true, anonymous users are allowed
}

// AuthOption is a function that configures AuthMiddleware
type AuthOption func(*authConfig)

// WithFailureHandler sets a custom handler for authentication failures
func WithFailureHandler(handler func(*gin.Context, error)) AuthOption {
	return func(config *authConfig) {
		config.failureHandler = handler
	}
}

// WithSuccessHandler sets a custom handler for authentication success
func WithSuccessHandler(handler func(*gin.Context, Identity)) AuthOption {
	return func(config *authConfig) {
		config.successHandler = handler
	}
}

// WithSkipPaths sets paths to skip authentication
func WithSkipPaths(paths ...string) AuthOption {
	return func(config *authConfig) {
		config.skipPaths = append(config.skipPaths, paths...)
	}
}

// WithAnonymousAccess allows anonymous users
func WithAnonymousAccess() AuthOption {
	return func(config *authConfig) {
		config.anonymous = true
	}
}

// DefaultFailureHandler is the default handler for authentication failures
func DefaultFailureHandler(c *gin.Context, err error) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"error":  "Unauthorized",
		"detail": err.Error(),
	})
	c.Abort()
}

// AuthMiddleware creates a Gin middleware for authentication
func AuthMiddleware(manager AuthManager, options ...AuthOption) gin.HandlerFunc {
	config := &authConfig{
		failureHandler: DefaultFailureHandler,
		successHandler: nil,
		skipPaths:      []string{},
		anonymous:      false,
	}

	for _, option := range options {
		option(config)
	}

	return func(c *gin.Context) {
		// Check for skip paths
		path := c.Request.URL.Path
		for _, skipPath := range config.skipPaths {
			if path == skipPath {
				c.Next()
				return
			}
		}

		// Try to authenticate
		identity, err := manager.Authenticate(c)
		if err != nil {
			if config.anonymous {
				// Allow anonymous access
				anonymousIdentity := NewAnonymousIdentity()
				WithIdentityGin(c, anonymousIdentity)

				if config.successHandler != nil {
					config.successHandler(c, anonymousIdentity)
				}

				c.Next()
				return
			}

			// Handle authentication failure
			if config.failureHandler != nil {
				config.failureHandler(c, err)
				return
			}

			DefaultFailureHandler(c, err)
			return
		}

		// Store identity in context
		WithIdentityGin(c, identity)

		// Call success handler if provided
		if config.successHandler != nil {
			config.successHandler(c, identity)
		}

		c.Next()
	}
}

// RequireAuthMiddleware creates a Gin middleware that requires authentication
func RequireAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsAuthenticatedGin(c) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		identity, _ := GetIdentityGin(c)
		if identity.GetID() == "anonymous" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRolesMiddleware creates a Gin middleware that requires specific roles
func RequireRolesMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		identity, exists := GetIdentityGin(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		if !identity.HasAnyRole(roles) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "Forbidden: insufficient role",
				"required_roles": roles,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
