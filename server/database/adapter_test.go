package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseAdapterInterface(t *testing.T) {
	var _ DatabaseAdapter = (*LocalAdapter)(nil)
	var _ DatabaseAdapter = (*CloudAdapter)(nil)
}

func testAdapterCRUD(t *testing.T, adapter DatabaseAdapter) {
	ctx := context.Background()

	t.Run("User CRUD", func(t *testing.T) {
		user := &User{
			ID:              "test-user-1",
			UserType:        UserTypeGuest,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}

		err := adapter.CreateUser(ctx, user)
		require.NoError(t, err)

		retrieved, err := adapter.GetUser(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, retrieved.ID)
		assert.Equal(t, user.UserType, retrieved.UserType)

		user.UserType = UserTypeAuthenticated
		cloudUserID := "cloud-123"
		user.CloudUserID = &cloudUserID
		err = adapter.UpdateUser(ctx, user)
		require.NoError(t, err)

		updated, err := adapter.GetUser(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, UserTypeAuthenticated, updated.UserType)
		assert.NotNil(t, updated.CloudUserID)
		assert.Equal(t, cloudUserID, *updated.CloudUserID)

		_, err = adapter.GetUser(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("API Key CRUD", func(t *testing.T) {
		user := &User{
			ID:              "test-user-api-keys",
			UserType:        UserTypeGuest,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}
		err := adapter.CreateUser(ctx, user)
		require.NoError(t, err)

		key := &APIKey{
			ID:           "key-1",
			UserID:       user.ID,
			Provider:     "openai",
			KeyHash:      "hash123",
			EncryptedKey: []byte("encrypted"),
			StorageType:  "encrypted_db",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			IsActive:     true,
		}

		err = adapter.StoreAPIKey(ctx, key)
		require.NoError(t, err)

		retrieved, err := adapter.GetAPIKey(ctx, user.ID, "openai")
		require.NoError(t, err)
		assert.Equal(t, key.Provider, retrieved.Provider)
		assert.Equal(t, key.KeyHash, retrieved.KeyHash)

		keys, err := adapter.ListAPIKeys(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, keys, 1)

		err = adapter.UpdateAPIKeyLastUsed(ctx, user.ID, "openai")
		require.NoError(t, err)

		err = adapter.DeleteAPIKey(ctx, user.ID, "openai")
		require.NoError(t, err)

		_, err = adapter.GetAPIKey(ctx, user.ID, "openai")
		assert.ErrorIs(t, err, ErrAPIKeyNotFound)
	})

	t.Run("Conversation CRUD", func(t *testing.T) {
		user := &User{
			ID:              "test-user-convos",
			UserType:        UserTypeGuest,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}
		err := adapter.CreateUser(ctx, user)
		require.NoError(t, err)

		conv := &Conversation{
			ID:           "conv-1",
			UserID:       user.ID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			MessageCount: 0,
			IsArchived:   false,
		}

		err = adapter.CreateConversation(ctx, conv)
		require.NoError(t, err)

		retrieved, err := adapter.GetConversation(ctx, conv.ID)
		require.NoError(t, err)
		assert.Equal(t, conv.ID, retrieved.ID)
		assert.Equal(t, conv.UserID, retrieved.UserID)

		convs, err := adapter.ListConversations(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, convs, 1)

		title := "Updated Title"
		conv.Title = &title
		conv.MessageCount = 5
		err = adapter.UpdateConversation(ctx, conv)
		require.NoError(t, err)

		updated, err := adapter.GetConversation(ctx, conv.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, updated.MessageCount)
		assert.NotNil(t, updated.Title)
		assert.Equal(t, title, *updated.Title)

		err = adapter.DeleteConversation(ctx, conv.ID)
		require.NoError(t, err)

		_, err = adapter.GetConversation(ctx, conv.ID)
		assert.ErrorIs(t, err, ErrConversationNotFound)
	})

	t.Run("Settings CRUD", func(t *testing.T) {
		user := &User{
			ID:              "test-user-settings",
			UserType:        UserTypeGuest,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}
		err := adapter.CreateUser(ctx, user)
		require.NoError(t, err)

		settings := &Settings{
			UserID:              user.ID,
			Theme:               "dark",
			Language:            "en",
			NotificationEnabled: true,
			AutoSaveEnabled:     true,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		err = adapter.CreateSettings(ctx, settings)
		require.NoError(t, err)

		retrieved, err := adapter.GetSettings(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "dark", retrieved.Theme)
		assert.Equal(t, "en", retrieved.Language)

		settings.Theme = "light"
		err = adapter.UpdateSettings(ctx, settings)
		require.NoError(t, err)

		updated, err := adapter.GetSettings(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "light", updated.Theme)
	})

	t.Run("Ping", func(t *testing.T) {
		err := adapter.Ping(ctx)
		assert.NoError(t, err)
	})
}
