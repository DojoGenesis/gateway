package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/server/database"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupMigrationHandlerTest(t *testing.T) (*sql.DB, *database.MigrationManager, *MigrationHandlers, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_migration_handler.db")

	// Use WAL mode and busy timeout to prevent SQLITE_BUSY during concurrent access.
	// The migration handler spawns background goroutines that need concurrent DB access,
	// so MaxOpenConns must remain >1 to avoid deadlocks.
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)

	migrationSQL, err := os.ReadFile("../migrations/20260207_v0.0.30_local_auth.sql")
	require.NoError(t, err)

	_, err = db.Exec(string(migrationSQL))
	require.NoError(t, err)

	cloudAdapter := database.NewCloudAdapter("url", "key")
	manager := database.NewMigrationManager(db, cloudAdapter)
	handlers := NewMigrationHandlers(manager)

	cleanup := func() {
		db.Close()
	}

	return db, manager, handlers, cleanup
}

func TestHandleStartMigration(t *testing.T) {
	db, _, handlers, cleanup := setupMigrationHandlerTest(t)
	defer cleanup()

	adapter := database.NewLocalAdapter(db)
	ctx := context.Background()

	userID := uuid.New().String()
	cloudUserID := uuid.New().String()
	user := &database.User{
		ID:              userID,
		UserType:        database.UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: database.MigrationStatusNone,
	}
	require.NoError(t, adapter.CreateUser(ctx, user))

	key := &database.APIKey{
		ID:           uuid.New().String(),
		UserID:       userID,
		Provider:     "openai",
		KeyHash:      "hash1",
		EncryptedKey: []byte("encrypted"),
		StorageType:  "encrypted_db",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	require.NoError(t, adapter.StoreAPIKey(ctx, key))

	gin.SetMode(gin.TestMode)

	t.Run("unauthorized - no user context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := StartMigrationRequest{CloudUserID: cloudUserID}
		bodyBytes, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/migration/start", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handlers.HandleStartMigration(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid request body", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", userID)

		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/migration/start", bytes.NewReader([]byte("invalid json")))
		c.Request.Header.Set("Content-Type", "application/json")

		handlers.HandleStartMigration(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("successful migration start", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", userID)

		reqBody := StartMigrationRequest{CloudUserID: cloudUserID}
		bodyBytes, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/migration/start", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handlers.HandleStartMigration(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp StartMigrationResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.MigrationID)
		assert.Equal(t, "started", resp.Status)
		assert.NotNil(t, resp.Summary)
		assert.Equal(t, 1, resp.Summary.APIKeys)
	})

	t.Run("migration already in progress", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", userID)

		reqBody := StartMigrationRequest{CloudUserID: cloudUserID}
		bodyBytes, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/migration/start", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handlers.HandleStartMigration(c)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "migration already")
	})

	t.Run("no data to migrate", func(t *testing.T) {
		emptyUserID := uuid.New().String()
		emptyUser := &database.User{
			ID:              emptyUserID,
			UserType:        database.UserTypeGuest,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: database.MigrationStatusNone,
		}
		require.NoError(t, adapter.CreateUser(ctx, emptyUser))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", emptyUserID)

		reqBody := StartMigrationRequest{CloudUserID: cloudUserID}
		bodyBytes, _ := json.Marshal(reqBody)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/migration/start", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handlers.HandleStartMigration(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "no data to migrate")
	})
}

func TestHandleGetMigrationStatus(t *testing.T) {
	db, _, handlers, cleanup := setupMigrationHandlerTest(t)
	defer cleanup()

	adapter := database.NewLocalAdapter(db)
	ctx := context.Background()

	userID := uuid.New().String()
	user := &database.User{
		ID:              userID,
		UserType:        database.UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: database.MigrationStatusNone,
	}
	require.NoError(t, adapter.CreateUser(ctx, user))

	migrationID := uuid.New().String()
	_, err := db.ExecContext(ctx, `
		INSERT INTO migration_log (id, user_id, migration_type, started_at, status, records_total, records_migrated, progress_percent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, migrationID, userID, database.MigrationTypeFull, time.Now(), database.MigrationLogStatusRunning, 100, 50, 50.0)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)

	t.Run("unauthorized - no user context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: migrationID}}

		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/migration/status/"+migrationID, nil)

		handlers.HandleGetMigrationStatus(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing migration ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", userID)

		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/migration/status/", nil)

		handlers.HandleGetMigrationStatus(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("successful status retrieval", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", userID)
		c.Params = gin.Params{{Key: "id", Value: migrationID}}

		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/migration/status/"+migrationID, nil)

		handlers.HandleGetMigrationStatus(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp GetMigrationStatusResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotNil(t, resp.Progress)
		assert.Equal(t, migrationID, resp.Progress.MigrationID)
		assert.Equal(t, database.MigrationLogStatusRunning, resp.Progress.Status)
		assert.Equal(t, 50, resp.Progress.RecordsMigrated)
		assert.Equal(t, 100, resp.Progress.RecordsTotal)
		assert.Equal(t, 50.0, resp.Progress.ProgressPercent)
	})

	t.Run("migration not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", userID)
		nonExistentID := uuid.New().String()
		c.Params = gin.Params{{Key: "id", Value: nonExistentID}}

		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/migration/status/"+nonExistentID, nil)

		handlers.HandleGetMigrationStatus(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleGetLatestMigration(t *testing.T) {
	db, _, handlers, cleanup := setupMigrationHandlerTest(t)
	defer cleanup()

	adapter := database.NewLocalAdapter(db)
	ctx := context.Background()

	userID := uuid.New().String()
	user := &database.User{
		ID:              userID,
		UserType:        database.UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: database.MigrationStatusNone,
	}
	require.NoError(t, adapter.CreateUser(ctx, user))

	gin.SetMode(gin.TestMode)

	t.Run("unauthorized - no user context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/migration/latest", nil)

		handlers.HandleGetLatestMigration(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("no migrations", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", userID)

		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/migration/latest", nil)

		handlers.HandleGetLatestMigration(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp GetLatestMigrationResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Nil(t, resp.Progress)
	})

	t.Run("successful latest migration retrieval", func(t *testing.T) {
		migrationID := uuid.New().String()
		_, err := db.ExecContext(ctx, `
			INSERT INTO migration_log (id, user_id, migration_type, started_at, status, records_total, records_migrated, progress_percent)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, migrationID, userID, database.MigrationTypeFull, time.Now(), database.MigrationLogStatusCompleted, 100, 100, 100.0)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", userID)

		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/migration/latest", nil)

		handlers.HandleGetLatestMigration(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp GetLatestMigrationResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotNil(t, resp.Progress)
		assert.Equal(t, migrationID, resp.Progress.MigrationID)
		assert.Equal(t, database.MigrationLogStatusCompleted, resp.Progress.Status)
	})
}

func TestHandleCancelMigration(t *testing.T) {
	db, _, handlers, cleanup := setupMigrationHandlerTest(t)
	defer cleanup()

	adapter := database.NewLocalAdapter(db)
	ctx := context.Background()

	userID := uuid.New().String()
	user := &database.User{
		ID:              userID,
		UserType:        database.UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: database.MigrationStatusNone,
	}
	require.NoError(t, adapter.CreateUser(ctx, user))

	migrationID := uuid.New().String()
	_, err := db.ExecContext(ctx, `
		INSERT INTO migration_log (id, user_id, migration_type, started_at, status, records_total, records_migrated, progress_percent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, migrationID, userID, database.MigrationTypeFull, time.Now(), database.MigrationLogStatusRunning, 100, 30, 30.0)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)

	t.Run("unauthorized - no user context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: migrationID}}

		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/migration/cancel/"+migrationID, nil)

		handlers.HandleCancelMigration(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing migration ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", userID)

		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/migration/cancel/", nil)

		handlers.HandleCancelMigration(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("successful cancellation", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", userID)
		c.Params = gin.Params{{Key: "id", Value: migrationID}}

		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/migration/cancel/"+migrationID, nil)

		handlers.HandleCancelMigration(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "cancelled successfully")
	})

	t.Run("migration not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", userID)
		nonExistentID := uuid.New().String()
		c.Params = gin.Params{{Key: "id", Value: nonExistentID}}

		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/migration/cancel/"+nonExistentID, nil)

		handlers.HandleCancelMigration(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleGetDataSummary(t *testing.T) {
	db, _, handlers, cleanup := setupMigrationHandlerTest(t)
	defer cleanup()

	adapter := database.NewLocalAdapter(db)
	ctx := context.Background()

	userID := uuid.New().String()
	user := &database.User{
		ID:              userID,
		UserType:        database.UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: database.MigrationStatusNone,
	}
	require.NoError(t, adapter.CreateUser(ctx, user))

	key := &database.APIKey{
		ID:           uuid.New().String(),
		UserID:       userID,
		Provider:     "openai",
		KeyHash:      "hash1",
		EncryptedKey: []byte("encrypted"),
		StorageType:  "encrypted_db",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	require.NoError(t, adapter.StoreAPIKey(ctx, key))

	conv := &database.Conversation{
		ID:        uuid.New().String(),
		UserID:    userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, adapter.CreateConversation(ctx, conv))

	gin.SetMode(gin.TestMode)

	t.Run("unauthorized - no user context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/migration/summary", nil)

		handlers.HandleGetDataSummary(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("successful summary retrieval", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", userID)

		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/migration/summary", nil)

		handlers.HandleGetDataSummary(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp GetDataSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotNil(t, resp.Summary)
		assert.Equal(t, 1, resp.Summary.APIKeys)
		assert.Equal(t, 1, resp.Summary.Conversations)
		assert.Equal(t, 2, resp.Summary.Total)
	})
}
