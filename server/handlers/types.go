package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Success *bool             `json:"success,omitempty"`
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

// respondErrorWithCode sends an error response with a semantic error code
func respondErrorWithCode(c *gin.Context, statusCode int, code, message string, details ...string) {
	resp := ErrorResponse{
		Error: message,
		Code:  code,
	}
	if len(details) > 0 {
		resp.Details = details[0]
	}
	c.JSON(statusCode, resp)
}

// respondNotFound sends a 404 Not Found error
func respondNotFound(c *gin.Context, resource string) {
	respondErrorWithCode(c, http.StatusNotFound, "NOT_FOUND",
		fmt.Sprintf("%s not found", resource))
}

// respondUnauthorized sends a 401 Unauthorized error
func respondUnauthorized(c *gin.Context, message string) {
	respondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// respondForbidden sends a 403 Forbidden error
func respondForbidden(c *gin.Context, message string) {
	respondErrorWithCode(c, http.StatusForbidden, "FORBIDDEN", message)
}

// respondConflict sends a 409 Conflict error
func respondConflict(c *gin.Context, message string) {
	respondErrorWithCode(c, http.StatusConflict, "CONFLICT", message)
}

// respondBadRequest sends a 400 Bad Request error
func respondBadRequest(c *gin.Context, message string, details ...string) {
	respondError(c, http.StatusBadRequest, message, details...)
}

// respondInternalError sends a 500 Internal Server Error
func respondInternalError(c *gin.Context, message string) {
	respondError(c, http.StatusInternalServerError, message)
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}

// respondErrorWithSuccess sends an error response that includes "success": false
// for backwards compatibility with handlers that previously used gin.H{"success": false, "error": "..."}.
func respondErrorWithSuccess(c *gin.Context, statusCode int, message string, details ...string) {
	resp := ErrorResponse{
		Success: boolPtr(false),
		Error:   message,
	}
	if len(details) > 0 {
		resp.Details = details[0]
	}
	c.JSON(statusCode, resp)
}

// respondBadRequestWithSuccess sends a 400 Bad Request error with "success": false.
func respondBadRequestWithSuccess(c *gin.Context, message string, details ...string) {
	respondErrorWithSuccess(c, http.StatusBadRequest, message, details...)
}

// respondInternalErrorWithSuccess sends a 500 Internal Server Error with "success": false.
func respondInternalErrorWithSuccess(c *gin.Context, message string) {
	respondErrorWithSuccess(c, http.StatusInternalServerError, message)
}

// respondNotFoundWithSuccess sends a 404 Not Found error with "success": false.
func respondNotFoundWithSuccess(c *gin.Context, message string) {
	respondErrorWithSuccess(c, http.StatusNotFound, message)
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
