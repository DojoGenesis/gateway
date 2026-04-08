package handlers

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/DojoGenesis/gateway/server/database"
	"github.com/gin-gonic/gin"
)

func sanitizeFilename(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
}

func getUserContext(c *gin.Context) (string, database.UserType, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", "", fmt.Errorf("user_id not found in context")
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return "", "", fmt.Errorf("user_id is not a string")
	}

	userType, exists := c.Get("user_type")
	if !exists {
		return "", "", fmt.Errorf("user_type not found in context")
	}

	var ut database.UserType
	switch v := userType.(type) {
	case string:
		ut = database.UserType(v)
	case database.UserType:
		ut = v
	default:
		return "", "", fmt.Errorf("user_type is not valid")
	}

	if ut != database.UserTypeGuest && ut != database.UserTypeAuthenticated {
		return "", "", database.ErrInvalidUserType
	}

	return userIDStr, ut, nil
}
