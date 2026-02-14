package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCloudAdapter_NotImplemented(t *testing.T) {
	adapter := NewCloudAdapter("http://localhost", "test-key")
	ctx := context.Background()

	t.Run("GetUser returns not implemented", func(t *testing.T) {
		_, err := adapter.GetUser(ctx, "user-id")
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("CreateUser returns not implemented", func(t *testing.T) {
		user := &User{
			ID:              "user-id",
			UserType:        UserTypeAuthenticated,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}
		err := adapter.CreateUser(ctx, user)
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("UpdateUser returns not implemented", func(t *testing.T) {
		user := &User{
			ID:              "user-id",
			UserType:        UserTypeAuthenticated,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}
		err := adapter.UpdateUser(ctx, user)
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("StoreAPIKey returns not implemented", func(t *testing.T) {
		key := &APIKey{
			ID:           "key-id",
			UserID:       "user-id",
			Provider:     "openai",
			KeyHash:      "hash",
			EncryptedKey: []byte("encrypted"),
			StorageType:  "keychain",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			IsActive:     true,
		}
		err := adapter.StoreAPIKey(ctx, key)
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("GetAPIKey returns not implemented", func(t *testing.T) {
		_, err := adapter.GetAPIKey(ctx, "user-id", "openai")
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("ListAPIKeys returns not implemented", func(t *testing.T) {
		_, err := adapter.ListAPIKeys(ctx, "user-id")
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("DeleteAPIKey returns not implemented", func(t *testing.T) {
		err := adapter.DeleteAPIKey(ctx, "user-id", "openai")
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("UpdateAPIKeyLastUsed returns not implemented", func(t *testing.T) {
		err := adapter.UpdateAPIKeyLastUsed(ctx, "user-id", "openai")
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("CreateConversation returns not implemented", func(t *testing.T) {
		conv := &Conversation{
			ID:           "conv-id",
			UserID:       "user-id",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			MessageCount: 0,
			IsArchived:   false,
		}
		err := adapter.CreateConversation(ctx, conv)
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("GetConversation returns not implemented", func(t *testing.T) {
		_, err := adapter.GetConversation(ctx, "conv-id")
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("ListConversations returns not implemented", func(t *testing.T) {
		_, err := adapter.ListConversations(ctx, "user-id")
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("UpdateConversation returns not implemented", func(t *testing.T) {
		conv := &Conversation{
			ID:           "conv-id",
			UserID:       "user-id",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			MessageCount: 0,
			IsArchived:   false,
		}
		err := adapter.UpdateConversation(ctx, conv)
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("DeleteConversation returns not implemented", func(t *testing.T) {
		err := adapter.DeleteConversation(ctx, "conv-id")
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("GetSettings returns not implemented", func(t *testing.T) {
		_, err := adapter.GetSettings(ctx, "user-id")
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("CreateSettings returns not implemented", func(t *testing.T) {
		settings := &Settings{
			UserID:              "user-id",
			Theme:               "dark",
			Language:            "en",
			NotificationEnabled: true,
			AutoSaveEnabled:     true,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		err := adapter.CreateSettings(ctx, settings)
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("UpdateSettings returns not implemented", func(t *testing.T) {
		settings := &Settings{
			UserID:              "user-id",
			Theme:               "dark",
			Language:            "en",
			NotificationEnabled: true,
			AutoSaveEnabled:     true,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		err := adapter.UpdateSettings(ctx, settings)
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("Ping returns not implemented", func(t *testing.T) {
		err := adapter.Ping(ctx)
		assert.ErrorIs(t, err, ErrCloudAdapterNotImplemented)
	})

	t.Run("Close succeeds", func(t *testing.T) {
		err := adapter.Close()
		assert.NoError(t, err)
	})
}
