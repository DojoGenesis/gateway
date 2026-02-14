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

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	migrationPath := filepath.Join("..", "migrations", "20260207_v0.0.30_local_auth.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)

	_, err = db.Exec(string(migrationSQL))
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestLocalAdapter_CRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	adapter := NewLocalAdapter(db)
	testAdapterCRUD(t, adapter)
}

func TestLocalAdapter_UserOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	adapter := NewLocalAdapter(db)
	ctx := context.Background()

	t.Run("Create and retrieve user", func(t *testing.T) {
		user := &User{
			ID:              "user-123",
			UserType:        UserTypeGuest,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}

		err := adapter.CreateUser(ctx, user)
		require.NoError(t, err)

		retrieved, err := adapter.GetUser(ctx, "user-123")
		require.NoError(t, err)
		assert.Equal(t, "user-123", retrieved.ID)
		assert.Equal(t, UserTypeGuest, retrieved.UserType)
	})

	t.Run("Update user migration status", func(t *testing.T) {
		user := &User{
			ID:              "user-migrate",
			UserType:        UserTypeGuest,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}

		err := adapter.CreateUser(ctx, user)
		require.NoError(t, err)

		user.MigrationStatus = MigrationStatusCompleted
		cloudID := "cloud-user-123"
		user.CloudUserID = &cloudID

		err = adapter.UpdateUser(ctx, user)
		require.NoError(t, err)

		retrieved, err := adapter.GetUser(ctx, "user-migrate")
		require.NoError(t, err)
		assert.Equal(t, MigrationStatusCompleted, retrieved.MigrationStatus)
		assert.NotNil(t, retrieved.CloudUserID)
		assert.Equal(t, "cloud-user-123", *retrieved.CloudUserID)
	})

	t.Run("Get non-existent user", func(t *testing.T) {
		_, err := adapter.GetUser(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestLocalAdapter_APIKeyOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	adapter := NewLocalAdapter(db)
	ctx := context.Background()

	user := &User{
		ID:              "user-keys",
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}
	err := adapter.CreateUser(ctx, user)
	require.NoError(t, err)

	t.Run("Store and retrieve API key", func(t *testing.T) {
		keyName := "My OpenAI Key"
		key := &APIKey{
			ID:           "key-openai",
			UserID:       "user-keys",
			Provider:     "openai",
			KeyName:      &keyName,
			KeyHash:      "sha256hash",
			EncryptedKey: []byte("encrypted-data"),
			StorageType:  "encrypted_db",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			IsActive:     true,
		}

		err := adapter.StoreAPIKey(ctx, key)
		require.NoError(t, err)

		retrieved, err := adapter.GetAPIKey(ctx, "user-keys", "openai")
		require.NoError(t, err)
		assert.Equal(t, "openai", retrieved.Provider)
		assert.Equal(t, "sha256hash", retrieved.KeyHash)
		assert.NotNil(t, retrieved.KeyName)
		assert.Equal(t, "My OpenAI Key", *retrieved.KeyName)
	})

	t.Run("Upsert API key", func(t *testing.T) {
		key := &APIKey{
			ID:           "key-anthropic",
			UserID:       "user-keys",
			Provider:     "anthropic",
			KeyHash:      "hash1",
			EncryptedKey: []byte("encrypted1"),
			StorageType:  "keychain",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			IsActive:     true,
		}

		err := adapter.StoreAPIKey(ctx, key)
		require.NoError(t, err)

		key.KeyHash = "hash2"
		key.EncryptedKey = []byte("encrypted2")
		key.UpdatedAt = time.Now()

		err = adapter.StoreAPIKey(ctx, key)
		require.NoError(t, err)

		retrieved, err := adapter.GetAPIKey(ctx, "user-keys", "anthropic")
		require.NoError(t, err)
		assert.Equal(t, "hash2", retrieved.KeyHash)
	})

	t.Run("List API keys", func(t *testing.T) {
		keys, err := adapter.ListAPIKeys(ctx, "user-keys")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(keys), 2)
	})

	t.Run("Update last used timestamp", func(t *testing.T) {
		time.Sleep(10 * time.Millisecond)

		before, err := adapter.GetAPIKey(ctx, "user-keys", "openai")
		require.NoError(t, err)

		err = adapter.UpdateAPIKeyLastUsed(ctx, "user-keys", "openai")
		require.NoError(t, err)

		after, err := adapter.GetAPIKey(ctx, "user-keys", "openai")
		require.NoError(t, err)

		if before.LastUsedAt == nil {
			assert.NotNil(t, after.LastUsedAt)
		} else {
			assert.True(t, after.LastUsedAt.After(*before.LastUsedAt))
		}
	})

	t.Run("Soft delete API key", func(t *testing.T) {
		err := adapter.DeleteAPIKey(ctx, "user-keys", "openai")
		require.NoError(t, err)

		_, err = adapter.GetAPIKey(ctx, "user-keys", "openai")
		assert.ErrorIs(t, err, ErrAPIKeyNotFound)

		var isActive bool
		err = db.QueryRow("SELECT is_active FROM api_keys WHERE user_id = ? AND provider = ?", "user-keys", "openai").Scan(&isActive)
		require.NoError(t, err)
		assert.False(t, isActive)
	})
}

func TestLocalAdapter_ConversationOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	adapter := NewLocalAdapter(db)
	ctx := context.Background()

	user := &User{
		ID:              "user-convos",
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}
	err := adapter.CreateUser(ctx, user)
	require.NoError(t, err)

	t.Run("Create conversation", func(t *testing.T) {
		title := "My Conversation"
		conv := &Conversation{
			ID:           "conv-1",
			UserID:       "user-convos",
			Title:        &title,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			MessageCount: 0,
			IsArchived:   false,
		}

		err := adapter.CreateConversation(ctx, conv)
		require.NoError(t, err)

		retrieved, err := adapter.GetConversation(ctx, "conv-1")
		require.NoError(t, err)
		assert.Equal(t, "conv-1", retrieved.ID)
		assert.NotNil(t, retrieved.Title)
		assert.Equal(t, "My Conversation", *retrieved.Title)
	})

	t.Run("Update conversation", func(t *testing.T) {
		conv, err := adapter.GetConversation(ctx, "conv-1")
		require.NoError(t, err)

		conv.MessageCount = 10
		now := time.Now()
		conv.LastMessageAt = &now

		err = adapter.UpdateConversation(ctx, conv)
		require.NoError(t, err)

		updated, err := adapter.GetConversation(ctx, "conv-1")
		require.NoError(t, err)
		assert.Equal(t, 10, updated.MessageCount)
		assert.NotNil(t, updated.LastMessageAt)
	})

	t.Run("List conversations", func(t *testing.T) {
		conv2 := &Conversation{
			ID:           "conv-2",
			UserID:       "user-convos",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			MessageCount: 0,
			IsArchived:   false,
		}
		err := adapter.CreateConversation(ctx, conv2)
		require.NoError(t, err)

		convs, err := adapter.ListConversations(ctx, "user-convos")
		require.NoError(t, err)
		assert.Len(t, convs, 2)
	})

	t.Run("Delete conversation", func(t *testing.T) {
		err := adapter.DeleteConversation(ctx, "conv-2")
		require.NoError(t, err)

		_, err = adapter.GetConversation(ctx, "conv-2")
		assert.ErrorIs(t, err, ErrConversationNotFound)
	})
}

func TestLocalAdapter_SettingsOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	adapter := NewLocalAdapter(db)
	ctx := context.Background()

	user := &User{
		ID:              "user-settings",
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}
	err := adapter.CreateUser(ctx, user)
	require.NoError(t, err)

	t.Run("Create settings", func(t *testing.T) {
		defaultModel := "gpt-4"
		settings := &Settings{
			UserID:              "user-settings",
			Theme:               "dark",
			Language:            "en",
			DefaultModel:        &defaultModel,
			NotificationEnabled: true,
			AutoSaveEnabled:     true,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		err := adapter.CreateSettings(ctx, settings)
		require.NoError(t, err)

		retrieved, err := adapter.GetSettings(ctx, "user-settings")
		require.NoError(t, err)
		assert.Equal(t, "dark", retrieved.Theme)
		assert.NotNil(t, retrieved.DefaultModel)
		assert.Equal(t, "gpt-4", *retrieved.DefaultModel)
	})

	t.Run("Update settings", func(t *testing.T) {
		settings, err := adapter.GetSettings(ctx, "user-settings")
		require.NoError(t, err)

		settings.Theme = "light"
		settings.NotificationEnabled = false

		err = adapter.UpdateSettings(ctx, settings)
		require.NoError(t, err)

		updated, err := adapter.GetSettings(ctx, "user-settings")
		require.NoError(t, err)
		assert.Equal(t, "light", updated.Theme)
		assert.False(t, updated.NotificationEnabled)
	})

	t.Run("Get non-existent settings", func(t *testing.T) {
		_, err := adapter.GetSettings(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrSettingsNotFound)
	})
}

func TestLocalAdapter_DataIsolation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	adapter := NewLocalAdapter(db)
	ctx := context.Background()

	user1 := &User{
		ID:              "user-1",
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}
	user2 := &User{
		ID:              "user-2",
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}

	err := adapter.CreateUser(ctx, user1)
	require.NoError(t, err)
	err = adapter.CreateUser(ctx, user2)
	require.NoError(t, err)

	key1 := &APIKey{
		ID:           "key-user1",
		UserID:       "user-1",
		Provider:     "openai",
		KeyHash:      "hash1",
		EncryptedKey: []byte("encrypted1"),
		StorageType:  "encrypted_db",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	key2 := &APIKey{
		ID:           "key-user2",
		UserID:       "user-2",
		Provider:     "openai",
		KeyHash:      "hash2",
		EncryptedKey: []byte("encrypted2"),
		StorageType:  "encrypted_db",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}

	err = adapter.StoreAPIKey(ctx, key1)
	require.NoError(t, err)
	err = adapter.StoreAPIKey(ctx, key2)
	require.NoError(t, err)

	t.Run("User 1 can only see their own keys", func(t *testing.T) {
		keys, err := adapter.ListAPIKeys(ctx, "user-1")
		require.NoError(t, err)
		assert.Len(t, keys, 1)
		assert.Equal(t, "hash1", keys[0].KeyHash)
	})

	t.Run("User 2 can only see their own keys", func(t *testing.T) {
		keys, err := adapter.ListAPIKeys(ctx, "user-2")
		require.NoError(t, err)
		assert.Len(t, keys, 1)
		assert.Equal(t, "hash2", keys[0].KeyHash)
	})

	t.Run("Users cannot access each other's data", func(t *testing.T) {
		conv1 := &Conversation{
			ID:           "conv-user1",
			UserID:       "user-1",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			MessageCount: 0,
			IsArchived:   false,
		}
		err := adapter.CreateConversation(ctx, conv1)
		require.NoError(t, err)

		convs, err := adapter.ListConversations(ctx, "user-2")
		require.NoError(t, err)
		assert.Len(t, convs, 0)
	})
}

func TestLocalAdapter_Ping(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	adapter := NewLocalAdapter(db)
	ctx := context.Background()

	err := adapter.Ping(ctx)
	assert.NoError(t, err)
}

func TestLocalAdapter_Close(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	adapter := NewLocalAdapter(db)

	err := adapter.Close()
	assert.NoError(t, err)
}

func TestLocalAdapter_ErrorConditions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	adapter := NewLocalAdapter(db)
	ctx := context.Background()

	user := &User{
		ID:              "error-user",
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}
	err := adapter.CreateUser(ctx, user)
	require.NoError(t, err)

	t.Run("Update non-existent user", func(t *testing.T) {
		nonExistent := &User{
			ID:              "non-existent",
			UserType:        UserTypeGuest,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}
		err := adapter.UpdateUser(ctx, nonExistent)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("Delete non-existent API key", func(t *testing.T) {
		err := adapter.DeleteAPIKey(ctx, "error-user", "non-existent-provider")
		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
	})

	t.Run("Update non-existent conversation", func(t *testing.T) {
		conv := &Conversation{
			ID:           "non-existent-conv",
			UserID:       "error-user",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			MessageCount: 0,
			IsArchived:   false,
		}
		err := adapter.UpdateConversation(ctx, conv)
		assert.ErrorIs(t, err, ErrConversationNotFound)
	})

	t.Run("Delete non-existent conversation", func(t *testing.T) {
		err := adapter.DeleteConversation(ctx, "non-existent-conv")
		assert.ErrorIs(t, err, ErrConversationNotFound)
	})

	t.Run("Update non-existent settings", func(t *testing.T) {
		settings := &Settings{
			UserID:              "non-existent-user",
			Theme:               "dark",
			Language:            "en",
			NotificationEnabled: true,
			AutoSaveEnabled:     true,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		err := adapter.UpdateSettings(ctx, settings)
		assert.ErrorIs(t, err, ErrSettingsNotFound)
	})
}
