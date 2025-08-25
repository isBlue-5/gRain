package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/sessions"
)

var (
	// ErrNoToken indicates that no authentication token was found
	ErrNoToken = errors.New("no authentication token found")

	// ErrInvalidToken indicates that the authentication token is invalid
	ErrInvalidToken = errors.New("invalid authentication token")

	// ErrTokenExpired indicates that the authentication token has expired
	ErrTokenExpired = errors.New("authentication token expired")

	// ErrNoSession indicates that no session was found
	ErrNoSession = errors.New("no session found")

	// ErrInvalidSession indicates that the session is invalid
	ErrInvalidSession = errors.New("invalid session")
)

// AuthProvider is the interface for authentication providers
type AuthProvider interface {
	// Authenticate extracts and validates identity from the request
	Authenticate(ctx *gin.Context) (Identity, error)

	// Name returns the name of this authentication provider
	Name() string
}

// AuthManager manages multiple authentication providers
type AuthManager interface {
	// Register adds an authentication provider
	Register(provider AuthProvider)

	// Authenticate tries all registered providers to authenticate
	Authenticate(ctx *gin.Context) (Identity, error)

	// GetProviders returns all registered authentication providers
	GetProviders() []AuthProvider
}

// DefaultAuthManager implements AuthManager
type DefaultAuthManager struct {
	providers []AuthProvider
}

// NewAuthManager creates a new authentication manager
func NewAuthManager() *DefaultAuthManager {
	return &DefaultAuthManager{
		providers: make([]AuthProvider, 0),
	}
}

// Register adds a new authentication provider
func (m *DefaultAuthManager) Register(provider AuthProvider) {
	m.providers = append(m.providers, provider)
}

// Authenticate tries all registered providers
func (m *DefaultAuthManager) Authenticate(ctx *gin.Context) (Identity, error) {
	var lastError error

	for _, provider := range m.providers {
		identity, err := provider.Authenticate(ctx)
		if err == nil {
			return identity, nil
		}
		lastError = err
	}

	// If all providers fail, return the last error
	if lastError != nil {
		return nil, lastError
	}

	// No providers configured
	return nil, errors.New("no authentication providers available")
}

// GetProviders returns all registered authentication providers
func (m *DefaultAuthManager) GetProviders() []AuthProvider {
	return m.providers
}

// JWTProvider is an implementation of AuthProvider for JWT authentication
type JWTProvider struct {
	secretKey   []byte
	tokenLookup string
	expiration  time.Duration
	issuer      string
	audience    string
}

// JWTProviderOption represents an option for JWT provider
type JWTProviderOption func(*JWTProvider)

// WithTokenLookup configures where to look for the token
// Format: "header:Authorization,query:token,cookie:jwt"
func WithTokenLookup(tokenLookup string) JWTProviderOption {
	return func(p *JWTProvider) {
		p.tokenLookup = tokenLookup
	}
}

// WithExpiration sets the token expiration time
func WithExpiration(expiration time.Duration) JWTProviderOption {
	return func(p *JWTProvider) {
		p.expiration = expiration
	}
}

// WithIssuer sets the token issuer
func WithIssuer(issuer string) JWTProviderOption {
	return func(p *JWTProvider) {
		p.issuer = issuer
	}
}

// WithAudience sets the token audience
func WithAudience(audience string) JWTProviderOption {
	return func(p *JWTProvider) {
		p.audience = audience
	}
}

// NewJWTProvider creates a new JWT authentication provider
func NewJWTProvider(secretKey string, options ...JWTProviderOption) *JWTProvider {
	provider := &JWTProvider{
		secretKey:   []byte(secretKey),
		tokenLookup: "header:Authorization",
		expiration:  24 * time.Hour,
		issuer:      "gRain",
		audience:    "api",
	}

	for _, option := range options {
		option(provider)
	}

	return provider
}

// Name returns the provider name
func (p *JWTProvider) Name() string {
	return "jwt"
}

// Authenticate implements AuthProvider interface
func (p *JWTProvider) Authenticate(ctx *gin.Context) (Identity, error) {
	// Extract token
	token, err := p.extractToken(ctx)
	if err != nil {
		return nil, err
	}

	// Parse and validate token
	claims, err := p.validateToken(token)
	if err != nil {
		return nil, err
	}

	// Create identity from claims
	identity := NewJWTIdentity(claims, token)

	return identity, nil
}

// extractToken gets the token from the request
func (p *JWTProvider) extractToken(ctx *gin.Context) (string, error) {
	parts := strings.Split(p.tokenLookup, ",")

	for _, part := range parts {
		kv := strings.Split(part, ":")
		if len(kv) != 2 {
			continue
		}

		switch strings.TrimSpace(kv[0]) {
		case "header":
			token := ctx.GetHeader(strings.TrimSpace(kv[1]))
			// Handle Authorization: Bearer token
			if strings.TrimSpace(kv[1]) == "Authorization" {
				token = strings.TrimPrefix(token, "Bearer ")
				token = strings.TrimSpace(token)
			}
			if token != "" {
				return token, nil
			}
		case "query":
			token := ctx.Query(strings.TrimSpace(kv[1]))
			if token != "" {
				return token, nil
			}
		case "cookie":
			token, err := ctx.Cookie(strings.TrimSpace(kv[1]))
			if err == nil && token != "" {
				return token, nil
			}
		}
	}

	return "", ErrNoToken
}

// validateToken validates the JWT token
func (p *JWTProvider) validateToken(tokenString string) (map[string]interface{}, error) {
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Check signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return p.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Validate claims
	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GenerateToken creates a new JWT token for the given claims
func (p *JWTProvider) GenerateToken(claims map[string]interface{}) (string, error) {
	// Create a new token
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims
	mapClaims := token.Claims.(jwt.MapClaims)
	for key, value := range claims {
		mapClaims[key] = value
	}

	// Set standard claims
	now := time.Now()
	mapClaims["iat"] = now.Unix()
	mapClaims["exp"] = now.Add(p.expiration).Unix()

	if p.issuer != "" {
		mapClaims["iss"] = p.issuer
	}

	if p.audience != "" {
		mapClaims["aud"] = p.audience
	}

	// Sign the token
	return token.SignedString(p.secretKey)
}

// SessionProvider is an implementation of AuthProvider for session authentication
type SessionProvider struct {
	sessionName  string
	sessionStore sessions.Store
}

// NewSessionProvider creates a new session authentication provider
func NewSessionProvider(sessionName string, store sessions.Store) *SessionProvider {
	return &SessionProvider{
		sessionName:  sessionName,
		sessionStore: store,
	}
}

// Name returns the provider name
func (p *SessionProvider) Name() string {
	return "session"
}

// Authenticate implements AuthProvider interface
func (p *SessionProvider) Authenticate(ctx *gin.Context) (Identity, error) {
	// Get session
	session, err := p.sessionStore.Get(ctx.Request, p.sessionName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSession, err)
	}

	if session.IsNew {
		return nil, ErrNoSession
	}

	// Extract user info from session
	userID, ok := session.Values["user_id"].(string)
	if !ok || userID == "" {
		return nil, ErrInvalidSession
	}

	username, _ := session.Values["username"].(string)

	// Extract roles
	var roles []string
	if rolesVal, ok := session.Values["roles"]; ok {
		if rolesSlice, ok := rolesVal.([]string); ok {
			roles = rolesSlice
		}
	}

	// Extract other claims
	claims := make(map[string]interface{})
	for key, value := range session.Values {
		if keyStr, ok := key.(string); ok {
			if keyStr != "user_id" && keyStr != "username" && keyStr != "roles" {
				claims[keyStr] = value
			}
		}
	}

	// Create identity
	identity := NewIdentity(userID, username, roles, claims)

	return identity, nil
}

// CreateSession creates a new session for the given identity
func (p *SessionProvider) CreateSession(ctx *gin.Context, identity Identity) error {
	session, _ := p.sessionStore.Get(ctx.Request, p.sessionName)

	// Set user info in session
	session.Values["user_id"] = identity.GetID()
	session.Values["username"] = identity.GetUsername()
	session.Values["roles"] = identity.GetRoles()

	// Add claims to session
	for key, value := range identity.GetClaims() {
		if key != "user_id" && key != "username" && key != "roles" {
			session.Values[key] = value
		}
	}

	// Save session
	return session.Save(ctx.Request, ctx.Writer)
}

// InvalidateSession removes the session
func (p *SessionProvider) InvalidateSession(ctx *gin.Context) error {
	session, err := p.sessionStore.Get(ctx.Request, p.sessionName)
	if err != nil {
		return err
	}

	// Delete session
	session.Options.MaxAge = -1
	return session.Save(ctx.Request, ctx.Writer)
}

// BasicAuthProvider is an implementation of AuthProvider for basic authentication
type BasicAuthProvider struct {
	authenticator func(username, password string) (Identity, error)
	realm         string
}

// NewBasicAuthProvider creates a new basic authentication provider
func NewBasicAuthProvider(authenticator func(username, password string) (Identity, error), realm string) *BasicAuthProvider {
	if realm == "" {
		realm = "Authorization Required"
	}

	return &BasicAuthProvider{
		authenticator: authenticator,
		realm:         realm,
	}
}

// Name returns the provider name
func (p *BasicAuthProvider) Name() string {
	return "basic_auth"
}

// Authenticate implements AuthProvider interface
func (p *BasicAuthProvider) Authenticate(ctx *gin.Context) (Identity, error) {
	// Extract credentials from Authorization header
	username, password, ok := ctx.Request.BasicAuth()
	if !ok {
		ctx.Header("WWW-Authenticate", `Basic realm="`+p.realm+`"`)
		return nil, ErrNoToken
	}

	// Validate credentials
	identity, err := p.authenticator(username, password)
	if err != nil {
		return nil, err
	}

	return identity, nil
}
