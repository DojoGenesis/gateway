package server

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/DojoGenesis/gateway/server/database"
	"github.com/DojoGenesis/gateway/server/middleware"
)

// ─── Request / Response types ───────────────────────────────────────────────

type authRegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type authLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authTokenResponse struct {
	UserID       string `json:"user_id"`
	DisplayName  string `json:"display_name"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// ─── Handlers ───────────────────────────────────────────────────────────────

// handleAuthRegister handles POST /auth/register.
func (s *Server) handleAuthRegister(c *gin.Context) {
	if !s.cfg.RegistrationEnabled {
		s.errorResponse(c, http.StatusForbidden, "registration_disabled", "User registration is currently disabled")
		return
	}

	var req authRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Validate fields
	req.Email = strings.TrimSpace(req.Email)
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if req.Email == "" || req.DisplayName == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Email and display_name are required")
		return
	}

	if len(req.Password) < 8 || len(req.Password) > 72 {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Password must be between 8 and 72 characters")
		return
	}

	// Hash password with bcrypt cost 12
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to process password")
		return
	}

	// Create user
	userID, err := database.CreatePortalUser(s.authDB, req.Email, string(hash), req.DisplayName)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			s.errorResponse(c, http.StatusConflict, "email_taken", "A user with this email already exists")
			return
		}
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to create user")
		return
	}

	// Issue tokens
	accessToken, err := issueToken(userID, "user", s.cfg.AccessTokenTTL)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to issue access token")
		return
	}

	refreshToken, err := issueToken(userID, "refresh", s.cfg.RefreshTokenTTL)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to issue refresh token")
		return
	}

	c.JSON(http.StatusCreated, authTokenResponse{
		UserID:       userID,
		DisplayName:  req.DisplayName,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
	})
}

// handleAuthLogin handles POST /auth/login.
func (s *Server) handleAuthLogin(c *gin.Context) {
	var req authLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)

	if req.Email == "" || req.Password == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Email and password are required")
		return
	}

	// Look up user by email
	userID, storedHash, displayName, err := database.GetPortalUserByEmail(s.authDB, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			s.errorResponse(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
			return
		}
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to look up user")
		return
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password)); err != nil {
		s.errorResponse(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	// Issue tokens
	accessToken, err := issueToken(userID, "user", s.cfg.AccessTokenTTL)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to issue access token")
		return
	}

	refreshToken, err := issueToken(userID, "refresh", s.cfg.RefreshTokenTTL)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to issue refresh token")
		return
	}

	c.JSON(http.StatusOK, authTokenResponse{
		UserID:       userID,
		DisplayName:  displayName,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
	})
}

// handleAuthRefresh handles POST /auth/refresh.
func (s *Server) handleAuthRefresh(c *gin.Context) {
	var req authRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if req.RefreshToken == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}

	// Validate refresh token
	userID, err := middleware.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		s.errorResponse(c, http.StatusUnauthorized, "invalid_token", "Invalid or expired refresh token")
		return
	}

	// Verify user still exists
	_, _, err = database.GetPortalUserByID(s.authDB, userID)
	if err != nil {
		s.errorResponse(c, http.StatusUnauthorized, "invalid_token", "User not found")
		return
	}

	// Issue new access token only
	accessToken, err := issueToken(userID, "user", s.cfg.AccessTokenTTL)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to issue access token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"expires_in":   86400,
	})
}

// ─── Token Issuance ─────────────────────────────────────────────────────────

// issueToken creates a signed JWT for the given user ID and role.
// ttl controls token lifetime (e.g. 24*time.Hour for access, 7*24*time.Hour for refresh).
func issueToken(userID, role string, ttl time.Duration) (string, error) {
	claims := middleware.GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
		Role: role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(middleware.GetJWTSecret())
}
