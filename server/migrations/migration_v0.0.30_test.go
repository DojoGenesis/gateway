package migrations

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestMigration_v0_0_30_LocalAuth(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_dojo.db")
	migrationPath := "20260207_v0.0.30_local_auth.sql"

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("Failed to read migration file: %v", err)
	}

	if _, err := db.Exec(string(migrationSQL)); err != nil {
		t.Fatalf("Failed to apply migration: %v", err)
	}

	t.Run("VerifySchemaVersionTracking", func(t *testing.T) {
		var version string
		var appliedAt string
		var description string

		err := db.QueryRow(`
			SELECT version, applied_at, description 
			FROM schema_migrations 
			WHERE version = '20260207_v0.0.30_local_auth'
		`).Scan(&version, &appliedAt, &description)

		if err != nil {
			t.Fatalf("Failed to query schema_migrations: %v", err)
		}

		if version != "20260207_v0.0.30_local_auth" {
			t.Errorf("Expected version '20260207_v0.0.30_local_auth', got '%s'", version)
		}

		if description == "" {
			t.Error("Migration description should not be empty")
		}
	})

	t.Run("VerifyAllTablesCreated", func(t *testing.T) {
		expectedTables := []string{
			"schema_migrations",
			"local_users",
			"api_keys",
			"conversations",
			"user_settings",
			"migration_log",
		}

		for _, tableName := range expectedTables {
			var name string
			err := db.QueryRow(`
				SELECT name FROM sqlite_master 
				WHERE type='table' AND name=?
			`, tableName).Scan(&name)

			if err == sql.ErrNoRows {
				t.Errorf("Table %s was not created", tableName)
			} else if err != nil {
				t.Fatalf("Error checking table %s: %v", tableName, err)
			}
		}
	})

	t.Run("VerifyIndexesCreated", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_local_users_type",
			"idx_local_users_migration_status",
			"idx_local_users_cloud_id",
			"idx_local_users_last_accessed",
			"idx_api_keys_user",
			"idx_api_keys_provider",
			"idx_api_keys_active",
			"idx_api_keys_last_used",
			"idx_api_keys_user_provider",
			"idx_conversations_user",
			"idx_conversations_project",
			"idx_conversations_last_message",
			"idx_conversations_archived",
			"idx_conversations_user_archived",
			"idx_user_settings_updated",
			"idx_migration_log_user",
			"idx_migration_log_status",
			"idx_migration_log_started",
			"idx_migration_log_user_status",
		}

		rows, err := db.Query(`
			SELECT name FROM sqlite_master 
			WHERE type='index' AND name LIKE 'idx_%'
		`)
		if err != nil {
			t.Fatalf("Failed to query indexes: %v", err)
		}
		defer rows.Close()

		foundIndexes := make(map[string]bool)
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatalf("Failed to scan index name: %v", err)
			}
			foundIndexes[name] = true
		}

		for _, idxName := range expectedIndexes {
			if !foundIndexes[idxName] {
				t.Errorf("Index %s was not created", idxName)
			}
		}

		t.Logf("Created %d indexes", len(foundIndexes))
	})

	t.Run("InsertLocalUser", func(t *testing.T) {
		userID := "test-user-123"
		now := time.Now().Format(time.RFC3339)

		_, err := db.Exec(`
			INSERT INTO local_users (id, user_type, created_at, last_accessed_at, migration_status)
			VALUES (?, ?, ?, ?, ?)
		`, userID, "guest", now, now, "none")

		if err != nil {
			t.Fatalf("Failed to insert local user: %v", err)
		}

		var retrievedID string
		var userType string
		var migrationStatus string

		err = db.QueryRow(`
			SELECT id, user_type, migration_status 
			FROM local_users 
			WHERE id = ?
		`, userID).Scan(&retrievedID, &userType, &migrationStatus)

		if err != nil {
			t.Fatalf("Failed to retrieve local user: %v", err)
		}

		if retrievedID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, retrievedID)
		}

		if userType != "guest" {
			t.Errorf("Expected user type 'guest', got '%s'", userType)
		}

		if migrationStatus != "none" {
			t.Errorf("Expected migration status 'none', got '%s'", migrationStatus)
		}
	})

	t.Run("InsertAPIKey", func(t *testing.T) {
		userID := "test-user-api"
		now := time.Now().Format(time.RFC3339)

		_, err := db.Exec(`
			INSERT INTO local_users (id, user_type, created_at, last_accessed_at)
			VALUES (?, ?, ?, ?)
		`, userID, "guest", now, now)

		if err != nil {
			t.Fatalf("Failed to insert user for API key test: %v", err)
		}

		keyID := "key-123"
		_, err = db.Exec(`
			INSERT INTO api_keys (
				id, user_id, provider, key_name, key_hash, 
				encrypted_key, storage_type, created_at, updated_at, is_active
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, keyID, userID, "openai", "My OpenAI Key", "hash123",
			[]byte("encrypted-key-data"), "keychain", now, now, 1)

		if err != nil {
			t.Fatalf("Failed to insert API key: %v", err)
		}

		var provider string
		var keyName string
		var isActive bool

		err = db.QueryRow(`
			SELECT provider, key_name, is_active 
			FROM api_keys 
			WHERE id = ?
		`, keyID).Scan(&provider, &keyName, &isActive)

		if err != nil {
			t.Fatalf("Failed to retrieve API key: %v", err)
		}

		if provider != "openai" {
			t.Errorf("Expected provider 'openai', got '%s'", provider)
		}

		if !isActive {
			t.Error("Expected is_active to be true")
		}
	})

	t.Run("InsertConversation", func(t *testing.T) {
		userID := "test-user-conv"
		now := time.Now().Format(time.RFC3339)

		_, err := db.Exec(`
			INSERT INTO local_users (id, user_type, created_at, last_accessed_at)
			VALUES (?, ?, ?, ?)
		`, userID, "guest", now, now)

		if err != nil {
			t.Fatalf("Failed to insert user for conversation test: %v", err)
		}

		convID := "conv-123"
		_, err = db.Exec(`
			INSERT INTO conversations (
				id, user_id, title, created_at, updated_at, 
				last_message_at, message_count, is_archived
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, convID, userID, "Test Conversation", now, now, now, 5, 0)

		if err != nil {
			t.Fatalf("Failed to insert conversation: %v", err)
		}

		var title string
		var messageCount int

		err = db.QueryRow(`
			SELECT title, message_count 
			FROM conversations 
			WHERE id = ?
		`, convID).Scan(&title, &messageCount)

		if err != nil {
			t.Fatalf("Failed to retrieve conversation: %v", err)
		}

		if title != "Test Conversation" {
			t.Errorf("Expected title 'Test Conversation', got '%s'", title)
		}

		if messageCount != 5 {
			t.Errorf("Expected message count 5, got %d", messageCount)
		}
	})

	t.Run("InsertUserSettings", func(t *testing.T) {
		userID := "test-user-settings"
		now := time.Now().Format(time.RFC3339)

		_, err := db.Exec(`
			INSERT INTO local_users (id, user_type, created_at, last_accessed_at)
			VALUES (?, ?, ?, ?)
		`, userID, "guest", now, now)

		if err != nil {
			t.Fatalf("Failed to insert user for settings test: %v", err)
		}

		_, err = db.Exec(`
			INSERT INTO user_settings (
				user_id, theme, language, default_model, 
				created_at, updated_at
			)
			VALUES (?, ?, ?, ?, ?, ?)
		`, userID, "dark", "en", "gpt-4", now, now)

		if err != nil {
			t.Fatalf("Failed to insert user settings: %v", err)
		}

		var theme string
		var language string
		var defaultModel string

		err = db.QueryRow(`
			SELECT theme, language, default_model 
			FROM user_settings 
			WHERE user_id = ?
		`, userID).Scan(&theme, &language, &defaultModel)

		if err != nil {
			t.Fatalf("Failed to retrieve user settings: %v", err)
		}

		if theme != "dark" {
			t.Errorf("Expected theme 'dark', got '%s'", theme)
		}

		if defaultModel != "gpt-4" {
			t.Errorf("Expected default model 'gpt-4', got '%s'", defaultModel)
		}
	})

	t.Run("InsertMigrationLog", func(t *testing.T) {
		userID := "test-user-migration"
		now := time.Now().Format(time.RFC3339)

		_, err := db.Exec(`
			INSERT INTO local_users (id, user_type, created_at, last_accessed_at)
			VALUES (?, ?, ?, ?)
		`, userID, "guest", now, now)

		if err != nil {
			t.Fatalf("Failed to insert user for migration test: %v", err)
		}

		migrationID := "migration-123"
		_, err = db.Exec(`
			INSERT INTO migration_log (
				id, user_id, migration_type, started_at, 
				status, records_migrated, records_total, progress_percent
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, migrationID, userID, "full", now, "running", 50, 100, 50.0)

		if err != nil {
			t.Fatalf("Failed to insert migration log: %v", err)
		}

		var migrationType string
		var status string
		var progressPercent float64

		err = db.QueryRow(`
			SELECT migration_type, status, progress_percent 
			FROM migration_log 
			WHERE id = ?
		`, migrationID).Scan(&migrationType, &status, &progressPercent)

		if err != nil {
			t.Fatalf("Failed to retrieve migration log: %v", err)
		}

		if status != "running" {
			t.Errorf("Expected status 'running', got '%s'", status)
		}

		if progressPercent != 50.0 {
			t.Errorf("Expected progress 50.0, got %f", progressPercent)
		}
	})

	t.Run("VerifyForeignKeyConstraints", func(t *testing.T) {
		_, err := db.Exec("PRAGMA foreign_keys = ON")
		if err != nil {
			t.Fatalf("Failed to enable foreign keys: %v", err)
		}

		now := time.Now().Format(time.RFC3339)

		result, err := db.Exec(`
			INSERT INTO api_keys (
				id, user_id, provider, key_name, key_hash, 
				encrypted_key, storage_type, created_at, updated_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "key-orphan", "nonexistent-user", "openai", "Test", "hash",
			[]byte("data"), "keychain", now, now)

		if err == nil {
			t.Error("Expected foreign key constraint violation, but insert succeeded")
			if result != nil {
				t.Logf("Unexpected successful insert result: %v", result)
			}
		}
	})

	t.Run("VerifyUniqueConstraints", func(t *testing.T) {
		userID := "test-user-unique"
		now := time.Now().Format(time.RFC3339)

		_, err := db.Exec(`
			INSERT INTO local_users (id, user_type, created_at, last_accessed_at)
			VALUES (?, ?, ?, ?)
		`, userID, "guest", now, now)

		if err != nil {
			t.Fatalf("Failed to insert user for unique constraint test: %v", err)
		}

		_, err = db.Exec(`
			INSERT INTO api_keys (
				id, user_id, provider, key_name, key_hash, 
				encrypted_key, storage_type, created_at, updated_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "key1", userID, "openai", "First", "hash1",
			[]byte("data1"), "keychain", now, now)

		if err != nil {
			t.Fatalf("Failed to insert first API key: %v", err)
		}

		_, err = db.Exec(`
			INSERT INTO api_keys (
				id, user_id, provider, key_name, key_hash, 
				encrypted_key, storage_type, created_at, updated_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "key2", userID, "openai", "Second", "hash2",
			[]byte("data2"), "keychain", now, now)

		if err == nil {
			t.Error("Expected unique constraint violation for (user_id, provider), but insert succeeded")
		}
	})

	t.Run("MigrationIdempotency", func(t *testing.T) {
		_, err := db.Exec(string(migrationSQL))
		if err != nil {
			t.Fatalf("Migration should be idempotent, but second run failed: %v", err)
		}

		var count int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM schema_migrations 
			WHERE version = '20260207_v0.0.30_local_auth'
		`).Scan(&count)

		if err != nil {
			t.Fatalf("Failed to count migrations: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 migration record, got %d", count)
		}
	})

	t.Run("Rollback", func(t *testing.T) {
		rollbackPath := "20260207_v0.0.30_rollback.sql"
		rollbackSQL, err := os.ReadFile(rollbackPath)
		if err != nil {
			t.Fatalf("Failed to read rollback file: %v", err)
		}

		if _, err := db.Exec(string(rollbackSQL)); err != nil {
			t.Fatalf("Failed to apply rollback: %v", err)
		}

		tablesToCheck := []string{
			"local_users",
			"api_keys",
			"conversations",
			"user_settings",
			"migration_log",
		}

		for _, tableName := range tablesToCheck {
			var name string
			err := db.QueryRow(`
				SELECT name FROM sqlite_master 
				WHERE type='table' AND name=?
			`, tableName).Scan(&name)

			if err != sql.ErrNoRows {
				t.Errorf("Table %s should have been dropped by rollback", tableName)
			}
		}

		var count int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM schema_migrations 
			WHERE version = '20260207_v0.0.30_local_auth'
		`).Scan(&count)

		if err != nil {
			t.Fatalf("Failed to query schema_migrations: %v", err)
		}

		if count != 0 {
			t.Errorf("Migration record should have been deleted, found %d records", count)
		}

		t.Log("Rollback completed successfully - all tables dropped and migration record removed")
	})
}
