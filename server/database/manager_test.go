package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupTestManager(t *testing.T) (*DatabaseManager, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_manager.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	migrationPath := filepath.Join("..", "migrations", "20260207_v0.0.30_local_auth.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)

	_, err = db.Exec(string(migrationSQL))
	require.NoError(t, err)

	localAdapter := NewLocalAdapter(db)
	cloudAdapter := NewCloudAdapter("", "")

	manager := NewDatabaseManager(localAdapter, cloudAdapter, false)

	cleanup := func() {
		manager.Close()
	}

	return manager, cleanup
}

func TestDatabaseManager_AdapterRouting(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Guest user uses local adapter", func(t *testing.T) {
		user := &User{
			ID:              "guest-user",
			UserType:        UserTypeGuest,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}

		err := manager.CreateUser(ctx, user)
		require.NoError(t, err)

		retrieved, err := manager.GetUser(ctx, "guest-user", UserTypeGuest)
		require.NoError(t, err)
		assert.Equal(t, "guest-user", retrieved.ID)
		assert.Equal(t, UserTypeGuest, retrieved.UserType)
	})

	t.Run("Authenticated user uses local adapter when cloud disabled", func(t *testing.T) {
		user := &User{
			ID:              "auth-user",
			UserType:        UserTypeAuthenticated,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}

		err := manager.CreateUser(ctx, user)
		require.NoError(t, err)

		retrieved, err := manager.GetUser(ctx, "auth-user", UserTypeAuthenticated)
		require.NoError(t, err)
		assert.Equal(t, "auth-user", retrieved.ID)
	})
}

func TestDatabaseManager_CloudAdapterRouting(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_cloud.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	migrationPath := filepath.Join("..", "migrations", "20260207_v0.0.30_local_auth.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)

	_, err = db.Exec(string(migrationSQL))
	require.NoError(t, err)

	localAdapter := NewLocalAdapter(db)
	cloudAdapter := NewCloudAdapter("", "")

	manager := NewDatabaseManager(localAdapter, cloudAdapter, true)
	defer manager.Close()

	ctx := context.Background()

	t.Run("Authenticated user uses cloud adapter when enabled", func(t *testing.T) {
		user := &User{
			ID:              "cloud-user",
			UserType:        UserTypeAuthenticated,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}

		err := manager.CreateUser(ctx, user)
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("Guest user still uses local adapter", func(t *testing.T) {
		user := &User{
			ID:              "guest-cloud",
			UserType:        UserTypeGuest,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}

		err := manager.CreateUser(ctx, user)
		require.NoError(t, err)

		retrieved, err := manager.GetUser(ctx, "guest-cloud", UserTypeGuest)
		require.NoError(t, err)
		assert.Equal(t, UserTypeGuest, retrieved.UserType)
	})
}

func TestDatabaseManager_APIKeyOperations(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	user := &User{
		ID:              "user-api",
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}
	err := manager.CreateUser(ctx, user)
	require.NoError(t, err)

	t.Run("Store and retrieve API key", func(t *testing.T) {
		key := &APIKey{
			ID:           "key-1",
			UserID:       "user-api",
			Provider:     "openai",
			KeyHash:      "hash123",
			EncryptedKey: []byte("encrypted"),
			StorageType:  "encrypted_db",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			IsActive:     true,
		}

		err := manager.StoreAPIKey(ctx, key, UserTypeGuest)
		require.NoError(t, err)

		retrieved, err := manager.GetAPIKey(ctx, "user-api", "openai", UserTypeGuest)
		require.NoError(t, err)
		assert.Equal(t, "openai", retrieved.Provider)
	})

	t.Run("List API keys", func(t *testing.T) {
		keys, err := manager.ListAPIKeys(ctx, "user-api", UserTypeGuest)
		require.NoError(t, err)
		assert.Len(t, keys, 1)
	})

	t.Run("Update API key last used", func(t *testing.T) {
		err := manager.UpdateAPIKeyLastUsed(ctx, "user-api", "openai", UserTypeGuest)
		require.NoError(t, err)
	})

	t.Run("Delete API key", func(t *testing.T) {
		err := manager.DeleteAPIKey(ctx, "user-api", "openai", UserTypeGuest)
		require.NoError(t, err)

		_, err = manager.GetAPIKey(ctx, "user-api", "openai", UserTypeGuest)
		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
	})
}

func TestDatabaseManager_ConversationOperations(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	user := &User{
		ID:              "user-conv",
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}
	err := manager.CreateUser(ctx, user)
	require.NoError(t, err)

	t.Run("Create and retrieve conversation", func(t *testing.T) {
		conv := &Conversation{
			ID:           "conv-1",
			UserID:       "user-conv",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			MessageCount: 0,
			IsArchived:   false,
		}

		err := manager.CreateConversation(ctx, conv, UserTypeGuest)
		require.NoError(t, err)

		retrieved, err := manager.GetConversation(ctx, "conv-1", UserTypeGuest)
		require.NoError(t, err)
		assert.Equal(t, "conv-1", retrieved.ID)
	})

	t.Run("List conversations", func(t *testing.T) {
		convs, err := manager.ListConversations(ctx, "user-conv", UserTypeGuest)
		require.NoError(t, err)
		assert.Len(t, convs, 1)
	})

	t.Run("Update conversation", func(t *testing.T) {
		conv, err := manager.GetConversation(ctx, "conv-1", UserTypeGuest)
		require.NoError(t, err)

		conv.MessageCount = 5
		err = manager.UpdateConversation(ctx, conv, UserTypeGuest)
		require.NoError(t, err)

		updated, err := manager.GetConversation(ctx, "conv-1", UserTypeGuest)
		require.NoError(t, err)
		assert.Equal(t, 5, updated.MessageCount)
	})

	t.Run("Delete conversation", func(t *testing.T) {
		err := manager.DeleteConversation(ctx, "conv-1", UserTypeGuest)
		require.NoError(t, err)

		_, err = manager.GetConversation(ctx, "conv-1", UserTypeGuest)
		assert.ErrorIs(t, err, ErrConversationNotFound)
	})
}

func TestDatabaseManager_SettingsOperations(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	user := &User{
		ID:              "user-settings",
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}
	err := manager.CreateUser(ctx, user)
	require.NoError(t, err)

	t.Run("Create and retrieve settings", func(t *testing.T) {
		settings := &Settings{
			UserID:              "user-settings",
			Theme:               "dark",
			Language:            "en",
			NotificationEnabled: true,
			AutoSaveEnabled:     true,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		err := manager.CreateSettings(ctx, settings, UserTypeGuest)
		require.NoError(t, err)

		retrieved, err := manager.GetSettings(ctx, "user-settings", UserTypeGuest)
		require.NoError(t, err)
		assert.Equal(t, "dark", retrieved.Theme)
	})

	t.Run("Update settings", func(t *testing.T) {
		settings, err := manager.GetSettings(ctx, "user-settings", UserTypeGuest)
		require.NoError(t, err)

		settings.Theme = "light"
		err = manager.UpdateSettings(ctx, settings, UserTypeGuest)
		require.NoError(t, err)

		updated, err := manager.GetSettings(ctx, "user-settings", UserTypeGuest)
		require.NoError(t, err)
		assert.Equal(t, "light", updated.Theme)
	})
}

func TestDatabaseManager_DataIsolation(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	guest := &User{
		ID:              "guest-isolation",
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}
	auth := &User{
		ID:              "auth-isolation",
		UserType:        UserTypeAuthenticated,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}

	err := manager.CreateUser(ctx, guest)
	require.NoError(t, err)
	err = manager.CreateUser(ctx, auth)
	require.NoError(t, err)

	guestKey := &APIKey{
		ID:           "guest-key",
		UserID:       "guest-isolation",
		Provider:     "openai",
		KeyHash:      "guest-hash",
		EncryptedKey: []byte("guest-encrypted"),
		StorageType:  "encrypted_db",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	authKey := &APIKey{
		ID:           "auth-key",
		UserID:       "auth-isolation",
		Provider:     "openai",
		KeyHash:      "auth-hash",
		EncryptedKey: []byte("auth-encrypted"),
		StorageType:  "encrypted_db",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}

	err = manager.StoreAPIKey(ctx, guestKey, UserTypeGuest)
	require.NoError(t, err)
	err = manager.StoreAPIKey(ctx, authKey, UserTypeAuthenticated)
	require.NoError(t, err)

	t.Run("Guest user cannot access authenticated user data", func(t *testing.T) {
		keys, err := manager.ListAPIKeys(ctx, "guest-isolation", UserTypeGuest)
		require.NoError(t, err)
		assert.Len(t, keys, 1)
		assert.Equal(t, "guest-hash", keys[0].KeyHash)
	})

	t.Run("Authenticated user cannot access guest user data", func(t *testing.T) {
		keys, err := manager.ListAPIKeys(ctx, "auth-isolation", UserTypeAuthenticated)
		require.NoError(t, err)
		assert.Len(t, keys, 1)
		assert.Equal(t, "auth-hash", keys[0].KeyHash)
	})
}

func TestDatabaseManager_Ping(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	err := manager.Ping(ctx, UserTypeGuest)
	assert.NoError(t, err)

	err = manager.Ping(ctx, UserTypeAuthenticated)
	assert.NoError(t, err)
}

func TestDatabaseManager_Close(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	err := manager.Close()
	assert.NoError(t, err)
}

func TestDatabaseManager_UpdateUser(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	user := &User{
		ID:              "update-user",
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}

	err := manager.CreateUser(ctx, user)
	require.NoError(t, err)

	t.Run("Update user successfully", func(t *testing.T) {
		user.MigrationStatus = MigrationStatusCompleted
		err := manager.UpdateUser(ctx, user)
		require.NoError(t, err)

		updated, err := manager.GetUser(ctx, "update-user", UserTypeGuest)
		require.NoError(t, err)
		assert.Equal(t, MigrationStatusCompleted, updated.MigrationStatus)
	})
}

func TestDatabaseManager_ErrorHandling(t *testing.T) {
	manager := NewDatabaseManager(nil, nil, false)

	ctx := context.Background()

	t.Run("Invalid adapter error", func(t *testing.T) {
		user := &User{
			ID:              "test",
			UserType:        UserTypeGuest,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}

		err := manager.CreateUser(ctx, user)
		assert.ErrorIs(t, err, ErrInvalidAdapter)
	})

	t.Run("Invalid user type", func(t *testing.T) {
		mgr, cleanup := setupTestManager(t)
		defer cleanup()

		user := &User{
			ID:              "test",
			UserType:        UserType("invalid"),
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}

		err := mgr.CreateUser(ctx, user)
		assert.ErrorIs(t, err, ErrInvalidUserType)
	})
}
