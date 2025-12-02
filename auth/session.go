package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "filesonthego_session"
	// AuthTokenHeader is the name of the Authorization header
	AuthTokenHeader = "Authorization"
	// BearerPrefix is the prefix for Bearer tokens
	BearerPrefix = "Bearer "
)

// SessionConfig holds session configuration
type SessionConfig struct {
	CookieName     string
	CookieDomain   string
	CookiePath     string
	CookieSecure   bool        // Set to true in production (HTTPS)
	CookieHTTPOnly bool        // Prevent JavaScript access
	CookieSameSite http.SameSite
	MaxAge         time.Duration
}

// SessionManager manages user sessions
type SessionManager struct {
	jwtManager *JWTManager
	config     SessionConfig
}

// NewSessionManager creates a new session manager
func NewSessionManager(jwtManager *JWTManager, config SessionConfig) *SessionManager {
	// Set defaults
	if config.CookieName == "" {
		config.CookieName = SessionCookieName
	}
	if config.CookiePath == "" {
		config.CookiePath = "/"
	}
	if config.MaxAge == 0 {
		config.MaxAge = 24 * time.Hour
	}

	return &SessionManager{
		jwtManager: jwtManager,
		config:     config,
	}
}

// SetSession sets a session cookie with the JWT token
func (m *SessionManager) SetSession(c *gin.Context, token string) {
	c.SetCookie(
		m.config.CookieName,
		token,
		int(m.config.MaxAge.Seconds()),
		m.config.CookiePath,
		m.config.CookieDomain,
		m.config.CookieSecure,
		m.config.CookieHTTPOnly,
	)

	// Also set SameSite attribute
	c.SetSameSite(m.config.CookieSameSite)
}

// GetToken retrieves the token from the request
// Checks Authorization header first, then cookie
func (m *SessionManager) GetToken(c *gin.Context) (string, error) {
	// Check Authorization header first
	authHeader := c.GetHeader(AuthTokenHeader)
	if authHeader != "" && len(authHeader) > len(BearerPrefix) {
		if authHeader[:len(BearerPrefix)] == BearerPrefix {
			return authHeader[len(BearerPrefix):], nil
		}
	}

	// Check cookie
	token, err := c.Cookie(m.config.CookieName)
	if err != nil {
		return "", ErrTokenNotFound
	}

	if token == "" {
		return "", ErrTokenNotFound
	}

	return token, nil
}

// GetClaims retrieves and validates the JWT claims from the request
func (m *SessionManager) GetClaims(c *gin.Context) (*JWTClaims, error) {
	token, err := m.GetToken(c)
	if err != nil {
		return nil, err
	}

	return m.jwtManager.ValidateToken(token)
}

// ClearSession clears the session cookie
func (m *SessionManager) ClearSession(c *gin.Context) {
	c.SetCookie(
		m.config.CookieName,
		"",
		-1, // MaxAge -1 deletes the cookie
		m.config.CookiePath,
		m.config.CookieDomain,
		m.config.CookieSecure,
		m.config.CookieHTTPOnly,
	)
}

// RefreshSession refreshes the session with a new token
func (m *SessionManager) RefreshSession(c *gin.Context) error {
	token, err := m.GetToken(c)
	if err != nil {
		return err
	}

	newToken, err := m.jwtManager.RefreshToken(token)
	if err != nil {
		return err
	}

	m.SetSession(c, newToken)
	return nil
}

// IsAuthenticated checks if the request has a valid session
func (m *SessionManager) IsAuthenticated(c *gin.Context) bool {
	_, err := m.GetClaims(c)
	return err == nil
}

// RequireAuth is middleware that requires authentication
func (m *SessionManager) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := m.GetClaims(c)
		if err != nil {
			if errors.Is(err, ErrTokenNotFound) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			} else if errors.Is(err, ErrExpiredToken) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			}
			c.Abort()
			return
		}

		// Store claims in context for use in handlers
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)
		c.Set("is_admin", claims.IsAdmin)
		c.Set("claims", claims)

		c.Next()
	}
}

// RequireAdmin is middleware that requires admin privileges
func (m *SessionManager) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := m.GetClaims(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		if !claims.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
			c.Abort()
			return
		}

		// Store claims in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)
		c.Set("is_admin", claims.IsAdmin)
		c.Set("claims", claims)

		c.Next()
	}
}

// GetUserID retrieves the user ID from the context
func GetUserID(c *gin.Context) (string, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", errors.New("user ID not found in context")
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return "", errors.New("invalid user ID in context")
	}

	return userIDStr, nil
}

// GetUserClaims retrieves the JWT claims from the context
func GetUserClaims(c *gin.Context) (*JWTClaims, error) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, errors.New("claims not found in context")
	}

	jwtClaims, ok := claims.(*JWTClaims)
	if !ok {
		return nil, errors.New("invalid claims in context")
	}

	return jwtClaims, nil
}
