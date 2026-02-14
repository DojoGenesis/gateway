package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupMigrationTestDB(t *testing.T) (*sql.DB, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_migration.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	migrationSQL, err := os.ReadFile("../migrations/20260207_v0.0.30_local_auth.sql")
	require.NoError(t, err)

	_, err = db.Exec(string(migrationSQL))
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestGetDataSummary(t *testing.T) {
	db, cleanup := setupMigrationTestDB(t)
	defer cleanup()

	adapter := NewLocalAdapter(db)
	manager := NewMigrationManager(db, nil)
	ctx := context.Background()

	userID := uuid.New().String()
	user := &User{
		ID:              userID,
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}
	require.NoError(t, adapter.CreateUser(ctx, user))

	tests := []struct {
		name      string
		setup     func()
		expected  *DataSummary
		wantError bool
	}{
		{
			name:  "empty data",
			setup: func() {},
			expected: &DataSummary{
				APIKeys:       0,
				Conversations: 0,
				Settings:      false,
				Total:         0,
			},
			wantError: false,
		},
		{
			name: "with api keys",
			setup: func() {
				key := &APIKey{
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
			},
			expected: &DataSummary{
				APIKeys:       1,
				Conversations: 0,
				Settings:      false,
				Total:         1,
			},
			wantError: false,
		},
		{
			name: "with conversations",
			setup: func() {
				conv := &Conversation{
					ID:        uuid.New().String(),
					UserID:    userID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				require.NoError(t, adapter.CreateConversation(ctx, conv))
			},
			expected: &DataSummary{
				APIKeys:       1,
				Conversations: 1,
				Settings:      false,
				Total:         2,
			},
			wantError: false,
		},
		{
			name: "with settings",
			setup: func() {
				settings := &Settings{
					UserID:              userID,
					Theme:               "dark",
					Language:            "en",
					NotificationEnabled: true,
					AutoSaveEnabled:     true,
					CreatedAt:           time.Now(),
					UpdatedAt:           time.Now(),
				}
				require.NoError(t, adapter.CreateSettings(ctx, settings))
			},
			expected: &DataSummary{
				APIKeys:       1,
				Conversations: 1,
				Settings:      true,
				Total:         3,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			summary, err := manager.GetDataSummary(ctx, userID)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.APIKeys, summary.APIKeys)
				assert.Equal(t, tt.expected.Conversations, summary.Conversations)
				assert.Equal(t, tt.expected.Settings, summary.Settings)
				assert.Equal(t, tt.expected.Total, summary.Total)
			}
		})
	}
}

func TestStartMigration(t *testing.T) {
	db, cleanup := setupMigrationTestDB(t)
	defer cleanup()

	adapter := NewLocalAdapter(db)
	ctx := context.Background()

	userID := uuid.New().String()
	cloudUserID := uuid.New().String()
	user := &User{
		ID:              userID,
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}
	require.NoError(t, adapter.CreateUser(ctx, user))

	key := &APIKey{
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

	t.Run("no cloud adapter", func(t *testing.T) {
		manager := NewMigrationManager(db, nil)
		migrationID, err := manager.StartMigration(ctx, userID, cloudUserID)
		assert.Error(t, err)
		assert.Equal(t, ErrCloudAdapterRequired, err)
		assert.Empty(t, migrationID)
	})

	t.Run("no data to migrate", func(t *testing.T) {
		emptyUserID := uuid.New().String()
		emptyUser := &User{
			ID:              emptyUserID,
			UserType:        UserTypeGuest,
			CreatedAt:       time.Now(),
			LastAccessedAt:  time.Now(),
			MigrationStatus: MigrationStatusNone,
		}
		require.NoError(t, adapter.CreateUser(ctx, emptyUser))

		cloudAdapter := NewCloudAdapter("url", "key")
		manager := NewMigrationManager(db, cloudAdapter)
		migrationID, err := manager.StartMigration(ctx, emptyUserID, cloudUserID)
		assert.Error(t, err)
		assert.Equal(t, ErrNoDataToMigrate, err)
		assert.Empty(t, migrationID)
	})

	t.Run("successful migration start", func(t *testing.T) {
		cloudAdapter := NewCloudAdapter("url", "key")
		manager := NewMigrationManager(db, cloudAdapter)

		migrationID, err := manager.StartMigration(ctx, userID, cloudUserID)
		assert.NoError(t, err)
		assert.NotEmpty(t, migrationID)

		time.Sleep(100 * time.Millisecond)

		progress, err := manager.GetMigrationStatus(ctx, migrationID)
		assert.NoError(t, err)
		assert.NotNil(t, progress)
		assert.Equal(t, migrationID, progress.MigrationID)
	})

	t.Run("migration already in progress", func(t *testing.T) {
		cloudAdapter := NewCloudAdapter("url", "key")
		manager := NewMigrationManager(db, cloudAdapter)

		_, err := db.ExecContext(ctx, `
			INSERT INTO migration_log (id, user_id, migration_type, started_at, status, records_total, records_migrated, progress_percent)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New().String(), userID, MigrationTypeFull, time.Now(), MigrationLogStatusRunning, 10, 0, 0.0)
		require.NoError(t, err)

		migrationID, err := manager.StartMigration(ctx, userID, cloudUserID)
		assert.Error(t, err)
		assert.Equal(t, ErrMigrationInProgress, err)
		assert.Empty(t, migrationID)
	})
}

func TestGetMigrationStatus(t *testing.T) {
	db, cleanup := setupMigrationTestDB(t)
	defer cleanup()

	manager := NewMigrationManager(db, nil)
	ctx := context.Background()

	t.Run("migration not found", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		progress, err := manager.GetMigrationStatus(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Equal(t, ErrMigrationNotFound, err)
		assert.Nil(t, progress)
	})

	t.Run("successful retrieval", func(t *testing.T) {
		migrationID := uuid.New().String()
		userID := uuid.New().String()
		now := time.Now()

		user := &User{
			ID:              userID,
			UserType:        UserTypeGuest,
			CreatedAt:       now,
			LastAccessedAt:  now,
			MigrationStatus: MigrationStatusNone,
		}
		require.NoError(t, NewLocalAdapter(db).CreateUser(ctx, user))

		_, err := db.ExecContext(ctx, `
			INSERT INTO migration_log (id, user_id, migration_type, started_at, status, records_total, records_migrated, progress_percent)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, migrationID, userID, MigrationTypeFull, now, MigrationLogStatusRunning, 100, 50, 50.0)
		require.NoError(t, err)

		progress, err := manager.GetMigrationStatus(ctx, migrationID)
		assert.NoError(t, err)
		assert.NotNil(t, progress)
		assert.Equal(t, migrationID, progress.MigrationID)
		assert.Equal(t, MigrationLogStatusRunning, progress.Status)
		assert.Equal(t, 50, progress.RecordsMigrated)
		assert.Equal(t, 100, progress.RecordsTotal)
		assert.Equal(t, 50.0, progress.ProgressPercent)
	})

	t.Run("completed migration with errors", func(t *testing.T) {
		migrationID := uuid.New().String()
		userID := uuid.New().String()
		now := time.Now()
		errorsJSON := `["error 1", "error 2"]`

		user := &User{
			ID:              userID,
			UserType:        UserTypeGuest,
			CreatedAt:       now,
			LastAccessedAt:  now,
			MigrationStatus: MigrationStatusNone,
		}
		require.NoError(t, NewLocalAdapter(db).CreateUser(ctx, user))

		_, err := db.ExecContext(ctx, `
			INSERT INTO migration_log (id, user_id, migration_type, started_at, completed_at, status, records_total, records_migrated, progress_percent, errors)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, migrationID, userID, MigrationTypeFull, now, now.Add(time.Minute), MigrationLogStatusFailed, 100, 75, 75.0, errorsJSON)
		require.NoError(t, err)

		progress, err := manager.GetMigrationStatus(ctx, migrationID)
		assert.NoError(t, err)
		assert.NotNil(t, progress)
		assert.Equal(t, MigrationLogStatusFailed, progress.Status)
		assert.NotNil(t, progress.CompletedAt)
		assert.Len(t, progress.Errors, 2)
	})
}

func TestGetLatestMigration(t *testing.T) {
	db, cleanup := setupMigrationTestDB(t)
	defer cleanup()

	manager := NewMigrationManager(db, nil)
	ctx := context.Background()
	userID := uuid.New().String()

	t.Run("no migrations", func(t *testing.T) {
		progress, err := manager.GetLatestMigration(ctx, userID)
		assert.Error(t, err)
		assert.Equal(t, ErrMigrationNotFound, err)
		assert.Nil(t, progress)
	})

	t.Run("multiple migrations - returns latest", func(t *testing.T) {
		oldMigrationID := uuid.New().String()
		newMigrationID := uuid.New().String()
		now := time.Now()

		user := &User{
			ID:              userID,
			UserType:        UserTypeGuest,
			CreatedAt:       now,
			LastAccessedAt:  now,
			MigrationStatus: MigrationStatusNone,
		}
		require.NoError(t, NewLocalAdapter(db).CreateUser(ctx, user))

		_, err := db.ExecContext(ctx, `
			INSERT INTO migration_log (id, user_id, migration_type, started_at, status, records_total, records_migrated, progress_percent)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, oldMigrationID, userID, MigrationTypeFull, now.Add(-time.Hour), MigrationLogStatusCompleted, 50, 50, 100.0)
		require.NoError(t, err)

		_, err = db.ExecContext(ctx, `
			INSERT INTO migration_log (id, user_id, migration_type, started_at, status, records_total, records_migrated, progress_percent)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, newMigrationID, userID, MigrationTypeFull, now, MigrationLogStatusRunning, 100, 25, 25.0)
		require.NoError(t, err)

		progress, err := manager.GetLatestMigration(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, progress)
		assert.Equal(t, newMigrationID, progress.MigrationID)
		assert.Equal(t, MigrationLogStatusRunning, progress.Status)
	})
}

func TestCancelMigration(t *testing.T) {
	db, cleanup := setupMigrationTestDB(t)
	defer cleanup()

	manager := NewMigrationManager(db, nil)
	ctx := context.Background()

	t.Run("migration not found", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		err := manager.CancelMigration(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Equal(t, ErrMigrationNotFound, err)
	})

	t.Run("successful cancellation", func(t *testing.T) {
		migrationID := uuid.New().String()
		userID := uuid.New().String()
		now := time.Now()

		user := &User{
			ID:              userID,
			UserType:        UserTypeGuest,
			CreatedAt:       now,
			LastAccessedAt:  now,
			MigrationStatus: MigrationStatusNone,
		}
		require.NoError(t, NewLocalAdapter(db).CreateUser(ctx, user))

		_, err := db.ExecContext(ctx, `
			INSERT INTO migration_log (id, user_id, migration_type, started_at, status, records_total, records_migrated, progress_percent)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, migrationID, userID, MigrationTypeFull, now, MigrationLogStatusRunning, 100, 30, 30.0)
		require.NoError(t, err)

		err = manager.CancelMigration(ctx, migrationID)
		assert.NoError(t, err)

		progress, err := manager.GetMigrationStatus(ctx, migrationID)
		assert.NoError(t, err)
		assert.Equal(t, MigrationLogStatusCancelled, progress.Status)
		assert.NotNil(t, progress.CompletedAt)
	})

	t.Run("cannot cancel completed migration", func(t *testing.T) {
		migrationID := uuid.New().String()
		userID := uuid.New().String()
		now := time.Now()

		user := &User{
			ID:              userID,
			UserType:        UserTypeGuest,
			CreatedAt:       now,
			LastAccessedAt:  now,
			MigrationStatus: MigrationStatusNone,
		}
		require.NoError(t, NewLocalAdapter(db).CreateUser(ctx, user))

		_, err := db.ExecContext(ctx, `
			INSERT INTO migration_log (id, user_id, migration_type, started_at, completed_at, status, records_total, records_migrated, progress_percent)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, migrationID, userID, MigrationTypeFull, now, now.Add(time.Minute), MigrationLogStatusCompleted, 100, 100, 100.0)
		require.NoError(t, err)

		err = manager.CancelMigration(ctx, migrationID)
		assert.Error(t, err)
		assert.Equal(t, ErrMigrationNotFound, err)
	})
}

func TestMigrationDataIntegrity(t *testing.T) {
	db, cleanup := setupMigrationTestDB(t)
	defer cleanup()

	adapter := NewLocalAdapter(db)
	ctx := context.Background()

	userID := uuid.New().String()
	cloudUserID := uuid.New().String()
	user := &User{
		ID:              userID,
		UserType:        UserTypeGuest,
		CreatedAt:       time.Now(),
		LastAccessedAt:  time.Now(),
		MigrationStatus: MigrationStatusNone,
	}
	require.NoError(t, adapter.CreateUser(ctx, user))

	providers := []string{"openai", "anthropic", "deepseek", "google", "cohere", "mistral", "other"}
	numAPIKeys := len(providers)
	for i := 0; i < numAPIKeys; i++ {
		key := &APIKey{
			ID:           uuid.New().String(),
			UserID:       userID,
			Provider:     providers[i],
			KeyHash:      uuid.New().String(),
			EncryptedKey: []byte("encrypted"),
			StorageType:  "encrypted_db",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			IsActive:     true,
		}
		require.NoError(t, adapter.StoreAPIKey(ctx, key))
	}

	numConversations := 20
	for i := 0; i < numConversations; i++ {
		conv := &Conversation{
			ID:        uuid.New().String(),
			UserID:    userID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, adapter.CreateConversation(ctx, conv))
	}

	settings := &Settings{
		UserID:              userID,
		Theme:               "dark",
		Language:            "en",
		NotificationEnabled: true,
		AutoSaveEnabled:     true,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	require.NoError(t, adapter.CreateSettings(ctx, settings))

	cloudAdapter := NewCloudAdapter("url", "key")
	manager := NewMigrationManager(db, cloudAdapter)

	summary, err := manager.GetDataSummary(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, numAPIKeys, summary.APIKeys)
	assert.Equal(t, numConversations, summary.Conversations)
	assert.True(t, summary.Settings)
	assert.Equal(t, numAPIKeys+numConversations+1, summary.Total)

	migrationID, err := manager.StartMigration(ctx, userID, cloudUserID)
	require.NoError(t, err)
	require.NotEmpty(t, migrationID)

	time.Sleep(200 * time.Millisecond)

	progress, err := manager.GetMigrationStatus(ctx, migrationID)
	require.NoError(t, err)
	assert.NotNil(t, progress)
	assert.Equal(t, numAPIKeys+numConversations+1, progress.RecordsTotal)
}
