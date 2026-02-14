package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error   string            `json:"error"`
	Details string            `json:"details,omitempty"`
	Code    string            `json:"code,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type PaginationParams struct {
	Limit  int `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset int `form:"offset" binding:"omitempty,min=0"`
}

type PaginationMetadata struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func respondError(c *gin.Context, statusCode int, message string, details ...string) {
	resp := ErrorResponse{
		Error: message,
	}
	if len(details) > 0 {
		resp.Details = details[0]
	}
	c.JSON(statusCode, resp)
}

func respondValidationError(c *gin.Context, fieldErrors map[string]string) {
	c.JSON(400, ErrorResponse{
		Error:  "Validation failed",
		Code:   "VALIDATION_ERROR",
		Fields: fieldErrors,
	})
}

func sanitizeInput(input string) string {
	input = strings.ReplaceAll(input, "\x00", "")

	input = strings.TrimSpace(input)

	if len(input) > 1000 {
		input = input[:1000]
	}

	return input
}

func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "23505") ||
		strings.Contains(errStr, "UNIQUE constraint failed")
}
