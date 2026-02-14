package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or expired token",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("token", token)

		c.Next()
	}
}

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

func validateToken(token string) (string, error) {
	if token == "test-token" {
		return "test-user", nil
	}

	if strings.HasPrefix(token, "user-") {
		return token, nil
	}

	return "", fmt.Errorf("invalid token format")
}

func validateAdminRole(token string, userID string) (bool, error) {
	if token == "test-token" {
		return false, nil
	}

	if strings.HasPrefix(token, "admin-") {
		return true, nil
	}

	return false, nil
}
