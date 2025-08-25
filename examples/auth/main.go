package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/isBlue-5/grain/pkg/auth"
)

// User represents a simple user model
type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Password string   `json:"password"` // In a real app, this would be hashed
	Roles    []string `json:"roles"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse represents JWT token response
type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expiresIn"`
	Type      string `json:"type"`
}

// UserResponse represents user data response
type UserResponse struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

func main() {
	// Create a router
	r := gin.Default()

	// Setup user database (in-memory for demo)
	users := map[string]User{
		"1": {
			ID:       "1",
			Username: "admin",
			Password: "password123", // In a real app, this would be hashed
			Roles:    []string{"ADMIN", "USER"},
		},
		"2": {
			ID:       "2",
			Username: "user",
			Password: "password456", // In a real app, this would be hashed
			Roles:    []string{"USER"},
		},
	}

	// Setup session store
	sessionStore := sessions.NewCookieStore([]byte("your-session-secret"))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 24, // 1 day
		HttpOnly: true,
	}

	// Create auth providers
	jwtProvider := auth.NewJWTProvider(
		"your-jwt-secret",
		auth.WithExpiration(1*time.Hour),
		auth.WithIssuer("gRain-example"),
	)

	sessionProvider := auth.NewSessionProvider("gRain-session", sessionStore)

	basicAuthProvider := auth.NewBasicAuthProvider(
		func(username, password string) (auth.Identity, error) {
			// Find user by username
			for _, user := range users {
				if user.Username == username && user.Password == password {
					return auth.NewIdentity(user.ID, user.Username, user.Roles, nil), nil
				}
			}
			return nil, auth.ErrInvalidToken
		},
		"gRain API",
	)

	// Create auth manager
	authManager := auth.NewAuthManager()
	authManager.Register(jwtProvider)
	authManager.Register(sessionProvider)
	authManager.Register(basicAuthProvider)

	// Auth middleware
	authMiddleware := auth.AuthMiddleware(
		authManager,
		auth.WithSkipPaths("/login", "/token"),
		auth.WithFailureHandler(func(c *gin.Context, err error) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":  "Authentication failed",
				"detail": err.Error(),
			})
		}),
	)

	// Apply auth middleware
	r.Use(authMiddleware)

	// Login endpoint - JWT
	r.POST("/token", func(c *gin.Context) {
		var loginReq LoginRequest
		if err := c.ShouldBindJSON(&loginReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Find user
		var user User
		found := false
		for _, u := range users {
			if u.Username == loginReq.Username && u.Password == loginReq.Password {
				user = u
				found = true
				break
			}
		}

		if !found {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Create JWT token
		token, err := jwtProvider.GenerateToken(map[string]interface{}{
			"sub":   user.ID,
			"name":  user.Username,
			"roles": user.Roles,
		})

		if err != nil {
			log.Printf("Error generating token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating token"})
			return
		}

		c.JSON(http.StatusOK, TokenResponse{
			Token:     token,
			ExpiresIn: 3600, // 1 hour
			Type:      "Bearer",
		})
	})

	// Login endpoint - Session
	r.POST("/login", func(c *gin.Context) {
		var loginReq LoginRequest
		if err := c.ShouldBindJSON(&loginReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Find user
		var user User
		found := false
		for _, u := range users {
			if u.Username == loginReq.Username && u.Password == loginReq.Password {
				user = u
				found = true
				break
			}
		}

		if !found {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Create identity
		identity := auth.NewIdentity(user.ID, user.Username, user.Roles, nil)

		// Create session
		if err := sessionProvider.CreateSession(c, identity); err != nil {
			log.Printf("Error creating session: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Login successful",
			"user": UserResponse{
				ID:       user.ID,
				Username: user.Username,
				Roles:    user.Roles,
			},
		})
	})

	// Public endpoint
	r.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "This is a public endpoint",
		})
	})

	// Protected endpoint - any authenticated user
	r.GET("/profile", auth.RequireAuthMiddleware(), func(c *gin.Context) {
		identity, _ := auth.GetIdentityGin(c)

		c.JSON(http.StatusOK, gin.H{
			"message": "This is a protected endpoint",
			"user": UserResponse{
				ID:       identity.GetID(),
				Username: identity.GetUsername(),
				Roles:    identity.GetRoles(),
			},
		})
	})

	// Admin endpoint - requires ADMIN role
	r.GET("/admin", auth.RequireRolesMiddleware("ADMIN"), func(c *gin.Context) {
		identity, _ := auth.GetIdentityGin(c)

		c.JSON(http.StatusOK, gin.H{
			"message": "This is an admin endpoint",
			"user": UserResponse{
				ID:       identity.GetID(),
				Username: identity.GetUsername(),
				Roles:    identity.GetRoles(),
			},
		})
	})

	// Logout endpoint
	r.POST("/logout", func(c *gin.Context) {
		// Invalidate session
		err := sessionProvider.InvalidateSession(c)
		if err != nil {
			log.Printf("Error invalidating session: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Logout successful",
		})
	})

	// Start server
	fmt.Println("Server running on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
