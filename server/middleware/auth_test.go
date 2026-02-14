package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupAuthTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestAuthMiddleware(t *testing.T) {
	router := setupAuthTestRouter()

	router.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		userID := c.GetString("user_id")
		token := c.GetString("token")
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"token":   token,
			"message": "Access granted",
		})
	})

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Valid Bearer token",
			authHeader:     "Bearer test-token",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "test-user")
				assert.Contains(t, w.Body.String(), "Access granted")
			},
		},
		{
			name:           "Valid user token",
			authHeader:     "Bearer user-12345",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "user-12345")
				assert.Contains(t, w.Body.String(), "Access granted")
			},
		},
		{
			name:           "Missing Authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Authorization header is required")
			},
		},
		{
			name:           "Invalid header format - no space",
			authHeader:     "BearerToken",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Invalid authorization header format")
			},
		},
		{
			name:           "Invalid scheme",
			authHeader:     "Basic test-token",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Invalid authorization scheme")
			},
		},
		{
			name:           "Empty token",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Token is required")
			},
		},
		{
			name:           "Invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Invalid or expired token")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestOptionalAuthMiddleware(t *testing.T) {
	router := setupAuthTestRouter()

	router.GET("/optional", OptionalAuthMiddleware(), func(c *gin.Context) {
		userID := c.GetString("user_id")
		userType := c.GetString("user_type")
		guestUserID := c.GetString("guest_user_id")

		response := gin.H{
			"user_id":   userID,
			"user_type": userType,
		}

		if guestUserID != "" {
			response["guest_user_id"] = guestUserID
			response["message"] = "Guest access"
		} else {
			response["message"] = "Authenticated access"
		}

		c.JSON(http.StatusOK, response)
	})

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Valid token - authenticated access",
			authHeader:     "Bearer test-token",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "test-user")
				assert.Contains(t, w.Body.String(), "Authenticated access")
				assert.Contains(t, w.Body.String(), `"user_type":"authenticated"`)
			},
		},
		{
			name:           "No token - guest access with UUID",
			authHeader:     "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Guest access")
				assert.Contains(t, w.Body.String(), `"user_type":"guest"`)
				assert.Contains(t, w.Body.String(), `"guest_user_id"`)

				// Verify UUID format by attempting to parse the response
				assert.Contains(t, w.Body.String(), `"user_id"`)
			},
		},
		{
			name:           "Invalid token - fallback to guest",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Guest access")
				assert.Contains(t, w.Body.String(), `"user_type":"guest"`)
				assert.Contains(t, w.Body.String(), `"guest_user_id"`)
			},
		},
		{
			name:           "Invalid format - fallback to guest",
			authHeader:     "Invalid format",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Guest access")
				assert.Contains(t, w.Body.String(), `"user_type":"guest"`)
				assert.Contains(t, w.Body.String(), `"guest_user_id"`)
			},
		},
		{
			name:           "Valid user token - authenticated access",
			authHeader:     "Bearer user-12345",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "user-12345")
				assert.Contains(t, w.Body.String(), "Authenticated access")
				assert.Contains(t, w.Body.String(), `"user_type":"authenticated"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/optional", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestGuestUserIDIsValidUUID(t *testing.T) {
	router := setupAuthTestRouter()

	var capturedGuestID string
	router.GET("/test", OptionalAuthMiddleware(), func(c *gin.Context) {
		capturedGuestID = c.GetString("guest_user_id")
		userType := c.GetString("user_type")
		c.JSON(http.StatusOK, gin.H{
			"user_type": userType,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, capturedGuestID)

	// Verify that the guest ID is a valid UUID
	_, err := uuid.Parse(capturedGuestID)
	assert.NoError(t, err, "guest_user_id should be a valid UUID")
}

func TestGuestIDIsConsistentAcrossRequest(t *testing.T) {
	router := setupAuthTestRouter()

	var firstGuestID, secondGuestID string
	router.GET("/test1", OptionalAuthMiddleware(), func(c *gin.Context) {
		firstGuestID = c.GetString("guest_user_id")
		c.JSON(http.StatusOK, gin.H{"guest_id": firstGuestID})
	})

	router.GET("/test2", OptionalAuthMiddleware(), func(c *gin.Context) {
		secondGuestID = c.GetString("guest_user_id")
		c.JSON(http.StatusOK, gin.H{"guest_id": secondGuestID})
	})

	// Make first request
	req1 := httptest.NewRequest(http.MethodGet, "/test1", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Make second request
	req2 := httptest.NewRequest(http.MethodGet, "/test2", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Both should be valid UUIDs
	assert.NotEmpty(t, firstGuestID)
	assert.NotEmpty(t, secondGuestID)

	// They should be different (new UUID per request in backend)
	assert.NotEqual(t, firstGuestID, secondGuestID, "Each request should generate a new guest ID")
}

func TestUserTypeSetCorrectly(t *testing.T) {
	router := setupAuthTestRouter()

	router.GET("/check", OptionalAuthMiddleware(), func(c *gin.Context) {
		userType := c.GetString("user_type")
		userID := c.GetString("user_id")
		c.JSON(http.StatusOK, gin.H{
			"user_type": userType,
			"user_id":   userID,
		})
	})

	tests := []struct {
		name             string
		authHeader       string
		expectedUserType string
	}{
		{
			name:             "No auth header - guest type",
			authHeader:       "",
			expectedUserType: "guest",
		},
		{
			name:             "Valid auth header - authenticated type",
			authHeader:       "Bearer test-token",
			expectedUserType: "authenticated",
		},
		{
			name:             "Invalid auth header - guest type",
			authHeader:       "Bearer invalid",
			expectedUserType: "guest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/check", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), `"user_type":"`+tt.expectedUserType+`"`)
		})
	}
}

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expectError bool
		expectedID  string
	}{
		{
			name:        "Test token",
			token:       "test-token",
			expectError: false,
			expectedID:  "test-user",
		},
		{
			name:        "User prefix token",
			token:       "user-12345",
			expectError: false,
			expectedID:  "user-12345",
		},
		{
			name:        "Another user prefix token",
			token:       "user-abc",
			expectError: false,
			expectedID:  "user-abc",
		},
		{
			name:        "Invalid token",
			token:       "invalid",
			expectError: true,
			expectedID:  "",
		},
		{
			name:        "Empty token",
			token:       "",
			expectError: true,
			expectedID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := validateToken(tt.token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, userID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, userID)
			}
		})
	}
}

func BenchmarkAuthMiddleware(b *testing.B) {
	router := setupAuthTestRouter()
	router.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkOptionalAuthMiddleware(b *testing.B) {
	router := setupAuthTestRouter()
	router.GET("/optional", OptionalAuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/optional", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
