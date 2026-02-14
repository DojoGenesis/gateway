package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// jwtSecret is loaded once from the environment.
// In production, JWT_SECRET must be set to a strong random value.
var jwtSecret = []byte(getEnvDefault("JWT_SECRET", "dev-secret-change-in-production"))

// isDevelopment returns true when ENVIRONMENT != "production".
func isDevelopment() bool {
	return os.Getenv("ENVIRONMENT") != "production"
}

// GatewayClaims extends jwt.RegisteredClaims with application-specific fields.
type GatewayClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role,omitempty"` // "admin", "user", etc.
}

// AuthMiddleware requires a valid JWT Bearer token.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authorization header is required",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid authorization header format. Expected: Bearer <token>",
			})
			c.Abort()
			return
		}

		scheme := parts[0]
		token := parts[1]

		if scheme != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid authorization scheme. Expected: Bearer",
			})
			c.Abort()
			return
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Token is required",
			})
			c.Abort()
			return
		}

		userID, err := validateToken(token)
		if err != nil {
			slog.Warn("token validation failed", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or expired token",
			})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("token", token)

		c.Next()
	}
}

// AdminAuthMiddleware requires a valid JWT with role=admin.
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Admin authorization required",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		scheme := parts[0]
		token := parts[1]

		if scheme != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid authorization scheme",
			})
			c.Abort()
			return
		}

		userID, err := validateToken(token)
		if err != nil {
			slog.Warn("admin token validation failed", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or expired token",
			})
			c.Abort()
			return
		}

		isAdmin, err := validateAdminRole(token, userID)
		if err != nil || !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Admin privileges required",
			})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("token", token)
		c.Set("is_admin", true)

		c.Next()
	}
}

// OptionalAuthMiddleware validates JWT if present, otherwise assigns a guest ID.
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token := parts[1]
				if userID, err := validateToken(token); err == nil {
					c.Set("user_id", userID)
					c.Set("token", token)
					c.Set("user_type", "authenticated")
					c.Next()
					return
				}
				// If token is a valid UUID, treat it as a persistent guest ID
				if _, err := uuid.Parse(token); err == nil {
					c.Set("guest_user_id", token)
					c.Set("user_id", token)
					c.Set("user_type", "guest")
					c.Next()
					return
				}
			}
		}

		guestID := uuid.New().String()
		c.Set("guest_user_id", guestID)
		c.Set("user_id", guestID)
		c.Set("user_type", "guest")

		c.Next()
	}
}

// validateToken parses and validates a JWT token.
// In development mode, also accepts legacy test tokens for backward compatibility.
func validateToken(tokenString string) (string, error) {
	// Development-only: accept legacy test tokens
	if isDevelopment() {
		if tokenString == "test-token" {
			return "test-user", nil
		}
		if strings.HasPrefix(tokenString, "user-") {
			return tokenString, nil
		}
		if strings.HasPrefix(tokenString, "admin-") {
			return tokenString, nil
		}
	}

	// Parse and validate JWT
	claims := &GatewayClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})
	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", jwt.ErrSignatureInvalid
	}

	// Extract user ID from subject claim
	subject, err := claims.GetSubject()
	if err != nil || subject == "" {
		return "", jwt.ErrTokenInvalidSubject
	}

	return subject, nil
}

// validateAdminRole checks if the token holder has admin privileges.
func validateAdminRole(tokenString string, userID string) (bool, error) {
	// Development-only: legacy token support
	if isDevelopment() {
		if tokenString == "test-token" {
			return false, nil
		}
		if strings.HasPrefix(tokenString, "admin-") {
			return true, nil
		}
	}

	// Parse JWT claims to check role
	claims := &GatewayClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})
	if err != nil {
		return false, err
	}

	return claims.Role == "admin", nil
}

func getEnvDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
