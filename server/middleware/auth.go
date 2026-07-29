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

// The signing secret, where it is read from, and the production startup gate
// all live in jwt_secret.go. Every sign and verify path below reads it through
// currentJWTSecret() — do not reintroduce a package-level copy here.

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

		userID, claims, err := validateTokenWithClaims(token)
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

		// Attribute machine traffic distinctly from human traffic. claims is
		// nil only for the development-only legacy shim tokens, which are never
		// service credentials.
		if claims != nil && claims.Role == ServiceRole {
			c.Set("auth_type", "service")
			c.Set("service_name", ServiceNameFromSubject(userID))
			c.Set("token_id", claims.ID)
			// jti is an identifier, not a credential — logging it is what lets
			// the operator find the token to revoke. The token itself is never
			// logged.
			slog.Info("service token authenticated",
				"subject", userID,
				"jti", claims.ID,
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
			)
		} else {
			c.Set("auth_type", "user")
		}

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
	subject, _, err := validateTokenWithClaims(tokenString)
	return subject, err
}

// validateTokenWithClaims is validateToken plus the parsed claims, which callers
// need in order to tell a machine credential from a human session.
//
// The returned claims are nil for the development-only legacy shim tokens,
// which are not JWTs at all.
func validateTokenWithClaims(tokenString string) (string, *GatewayClaims, error) {
	// Development-only: accept legacy test tokens.
	//
	// UNCHANGED by the service-token work, and deliberately NOT extended to it:
	// production sets ENVIRONMENT=production, which makes this block dead code
	// there. A service token is a real signed JWT and takes the same signature
	// verification path as every human session below.
	if isDevelopment() {
		if tokenString == "test-token" {
			return "test-user", nil, nil
		}
		if strings.HasPrefix(tokenString, "user-") {
			return tokenString, nil, nil
		}
		if strings.HasPrefix(tokenString, "admin-") {
			return tokenString, nil, nil
		}
	}

	// Parse and validate JWT
	claims := &GatewayClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return currentJWTSecret(), nil
	})
	if err != nil {
		return "", nil, err
	}

	if !token.Valid {
		return "", nil, jwt.ErrSignatureInvalid
	}

	// Extract user ID from subject claim
	subject, err := claims.GetSubject()
	if err != nil || subject == "" {
		return "", nil, jwt.ErrTokenInvalidSubject
	}

	// Machine credentials carry extra constraints — a mandatory jti, a bounded
	// lifetime, a namespaced subject, and the revocation denylist. These run
	// AFTER the signature check above, never instead of it, and can only
	// reject a token the signature check already accepted.
	if claims.Role == ServiceRole {
		if err := validateServiceClaims(claims, subject); err != nil {
			return "", nil, err
		}
	}

	return subject, claims, nil
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
		return currentJWTSecret(), nil
	})
	if err != nil {
		return false, err
	}

	// Least privilege: a machine credential is NEVER an admin. Two independent
	// checks, so that neither a future widening of the role test below nor a
	// mis-minted token carrying role="admin" on a service subject can produce
	// an admin machine credential.
	if claims.Role == ServiceRole {
		return false, nil
	}
	if subject, _ := claims.GetSubject(); IsServiceSubject(subject) {
		return false, nil
	}

	return claims.Role == "admin", nil
}

// GetJWTSecret returns the configured JWT secret for use by token-issuing handlers.
// This is the only function in this file that should be exported for issuance.
func GetJWTSecret() []byte {
	return currentJWTSecret()
}

// ValidateRefreshToken validates a refresh token and returns the user ID.
// Returns error if the token is expired, malformed, or not a refresh token.
func ValidateRefreshToken(tokenString string) (string, error) {
	claims := &GatewayClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return currentJWTSecret(), nil
	})
	if err != nil || !token.Valid {
		return "", jwt.ErrSignatureInvalid
	}
	if claims.Role != "refresh" {
		return "", jwt.ErrTokenInvalidClaims
	}
	subject, err := claims.GetSubject()
	if err != nil || subject == "" {
		return "", jwt.ErrTokenInvalidSubject
	}
	return subject, nil
}
