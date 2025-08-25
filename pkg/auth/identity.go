package auth

// Identity represents an authenticated user identity
type Identity interface {
	// GetID returns the unique identifier
	GetID() string

	// GetUsername returns the username
	GetUsername() string

	// GetRoles returns the user's roles
	GetRoles() []string

	// HasRole checks if the identity has a specific role
	HasRole(role string) bool

	// HasAnyRole checks if the identity has any of the specified roles
	HasAnyRole(roles []string) bool

	// GetClaims returns additional identity information
	GetClaims() map[string]interface{}
}

// DefaultIdentity is a basic implementation of the Identity interface
type DefaultIdentity struct {
	id       string
	username string
	roles    []string
	claims   map[string]interface{}
}

// NewIdentity creates a new DefaultIdentity
func NewIdentity(id, username string, roles []string, claims map[string]interface{}) *DefaultIdentity {
	if claims == nil {
		claims = make(map[string]interface{})
	}

	return &DefaultIdentity{
		id:       id,
		username: username,
		roles:    roles,
		claims:   claims,
	}
}

// GetID implements the Identity interface
func (i *DefaultIdentity) GetID() string {
	return i.id
}

// GetUsername implements the Identity interface
func (i *DefaultIdentity) GetUsername() string {
	return i.username
}

// GetRoles implements the Identity interface
func (i *DefaultIdentity) GetRoles() []string {
	return i.roles
}

// HasRole implements the Identity interface
func (i *DefaultIdentity) HasRole(role string) bool {
	for _, r := range i.roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole implements the Identity interface
func (i *DefaultIdentity) HasAnyRole(roles []string) bool {
	for _, role := range roles {
		if i.HasRole(role) {
			return true
		}
	}
	return false
}

// GetClaims implements the Identity interface
func (i *DefaultIdentity) GetClaims() map[string]interface{} {
	return i.claims
}

// WithRole adds a role to the identity and returns the updated identity
func (i *DefaultIdentity) WithRole(role string) *DefaultIdentity {
	// Check if role already exists
	for _, r := range i.roles {
		if r == role {
			return i
		}
	}

	// Add the role
	i.roles = append(i.roles, role)
	return i
}

// WithClaim adds a claim to the identity and returns the updated identity
func (i *DefaultIdentity) WithClaim(key string, value interface{}) *DefaultIdentity {
	i.claims[key] = value
	return i
}

// JWTIdentity is an identity implementation based on JWT claims
type JWTIdentity struct {
	*DefaultIdentity
	token string
}

// NewJWTIdentity creates a new JWTIdentity from JWT claims
func NewJWTIdentity(claims map[string]interface{}, token string) *JWTIdentity {
	// Extract standard claims
	id, _ := claims["sub"].(string)
	username, _ := claims["name"].(string)

	// Extract roles
	roles := []string{}
	if rolesValue, ok := claims["roles"]; ok {
		if rolesSlice, ok := rolesValue.([]interface{}); ok {
			for _, role := range rolesSlice {
				if roleStr, ok := role.(string); ok {
					roles = append(roles, roleStr)
				}
			}
		}
	}

	return &JWTIdentity{
		DefaultIdentity: NewIdentity(id, username, roles, claims),
		token:           token,
	}
}

// GetToken returns the JWT token string
func (i *JWTIdentity) GetToken() string {
	return i.token
}

// AnonymousIdentity represents an unauthenticated user
type AnonymousIdentity struct{}

// GetID implements Identity interface for anonymous users
func (i *AnonymousIdentity) GetID() string {
	return "anonymous"
}

// GetUsername implements Identity interface for anonymous users
func (i *AnonymousIdentity) GetUsername() string {
	return "anonymous"
}

// GetRoles implements Identity interface for anonymous users
func (i *AnonymousIdentity) GetRoles() []string {
	return []string{"ANONYMOUS"}
}

// HasRole implements Identity interface for anonymous users
func (i *AnonymousIdentity) HasRole(role string) bool {
	return role == "ANONYMOUS"
}

// HasAnyRole implements Identity interface for anonymous users
func (i *AnonymousIdentity) HasAnyRole(roles []string) bool {
	for _, role := range roles {
		if role == "ANONYMOUS" {
			return true
		}
	}
	return false
}

// GetClaims implements Identity interface for anonymous users
func (i *AnonymousIdentity) GetClaims() map[string]interface{} {
	return map[string]interface{}{
		"anonymous": true,
	}
}

// NewAnonymousIdentity creates a new anonymous identity
func NewAnonymousIdentity() *AnonymousIdentity {
	return &AnonymousIdentity{}
}
