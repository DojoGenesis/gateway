package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type LocalAdapter struct {
	db *sql.DB
}

func NewLocalAdapter(db *sql.DB) *LocalAdapter {
	return &LocalAdapter{db: db}
}

func (a *LocalAdapter) GetUser(ctx context.Context, userID string) (*User, error) {
	query := `
		SELECT id, user_type, created_at, last_accessed_at, cloud_user_id, migration_status, metadata
		FROM local_users
		WHERE id = ?
	`

	user := &User{}
	err := a.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.UserType,
		&user.CreatedAt,
		&user.LastAccessedAt,
		&user.CloudUserID,
		&user.MigrationStatus,
		&user.Metadata,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (a *LocalAdapter) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO local_users (id, user_type, created_at, last_accessed_at, cloud_user_id, migration_status, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := a.db.ExecContext(ctx, query,
		user.ID,
		user.UserType,
		user.CreatedAt,
		user.LastAccessedAt,
		user.CloudUserID,
		user.MigrationStatus,
		user.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (a *LocalAdapter) UpdateUser(ctx context.Context, user *User) error {
	query := `
		UPDATE local_users
		SET user_type = ?, last_accessed_at = ?, cloud_user_id = ?, migration_status = ?, metadata = ?
		WHERE id = ?
	`

	result, err := a.db.ExecContext(ctx, query,
		user.UserType,
		user.LastAccessedAt,
		user.CloudUserID,
		user.MigrationStatus,
		user.Metadata,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (a *LocalAdapter) StoreAPIKey(ctx context.Context, key *APIKey) error {
	query := `
		INSERT INTO api_keys (id, user_id, provider, key_name, key_hash, encrypted_key, storage_type, created_at, updated_at, last_used_at, is_active, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, provider) DO UPDATE SET
			key_name = excluded.key_name,
			key_hash = excluded.key_hash,
			encrypted_key = excluded.encrypted_key,
			storage_type = excluded.storage_type,
			updated_at = excluded.updated_at,
			is_active = excluded.is_active,
			metadata = excluded.metadata
	`

	_, err := a.db.ExecContext(ctx, query,
		key.ID,
		key.UserID,
		key.Provider,
		key.KeyName,
		key.KeyHash,
		key.EncryptedKey,
		key.StorageType,
		key.CreatedAt,
		key.UpdatedAt,
		key.LastUsedAt,
		key.IsActive,
		key.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to store api key: %w", err)
	}

	return nil
}

func (a *LocalAdapter) GetAPIKey(ctx context.Context, userID, provider string) (*APIKey, error) {
	query := `
		SELECT id, user_id, provider, key_name, key_hash, encrypted_key, storage_type, created_at, updated_at, last_used_at, is_active, metadata
		FROM api_keys
		WHERE user_id = ? AND provider = ? AND is_active = 1
	`

	key := &APIKey{}
	err := a.db.QueryRowContext(ctx, query, userID, provider).Scan(
		&key.ID,
		&key.UserID,
		&key.Provider,
		&key.KeyName,
		&key.KeyHash,
		&key.EncryptedKey,
		&key.StorageType,
		&key.CreatedAt,
		&key.UpdatedAt,
		&key.LastUsedAt,
		&key.IsActive,
		&key.Metadata,
	)

	if err == sql.ErrNoRows {
		return nil, ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}

	return key, nil
}

func (a *LocalAdapter) ListAPIKeys(ctx context.Context, userID string) ([]*APIKey, error) {
	query := `
		SELECT id, user_id, provider, key_name, key_hash, encrypted_key, storage_type, created_at, updated_at, last_used_at, is_active, metadata
		FROM api_keys
		WHERE user_id = ? AND is_active = 1
		ORDER BY created_at DESC
	`

	rows, err := a.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list api keys: %w", err)
	}
	defer rows.Close()

	keys := []*APIKey{}
	for rows.Next() {
		key := &APIKey{}
		err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.Provider,
			&key.KeyName,
			&key.KeyHash,
			&key.EncryptedKey,
			&key.StorageType,
			&key.CreatedAt,
			&key.UpdatedAt,
			&key.LastUsedAt,
			&key.IsActive,
			&key.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan api key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate api keys: %w", err)
	}

	return keys, nil
}

// FindAPIKeyByProvider finds any active API key for a given provider across all users.
// This is used by the in-process provider to dynamically resolve API keys
// added through the Dev Mode UI, regardless of which user added them.
func (a *LocalAdapter) FindAPIKeyByProvider(ctx context.Context, provider string) (*APIKey, error) {
	query := `
		SELECT id, user_id, provider, key_name, key_hash, encrypted_key, storage_type, created_at, updated_at, last_used_at, is_active, metadata
		FROM api_keys
		WHERE provider = ? AND is_active = 1
		ORDER BY updated_at DESC
		LIMIT 1
	`

	key := &APIKey{}
	err := a.db.QueryRowContext(ctx, query, provider).Scan(
		&key.ID,
		&key.UserID,
		&key.Provider,
		&key.KeyName,
		&key.KeyHash,
		&key.EncryptedKey,
		&key.StorageType,
		&key.CreatedAt,
		&key.UpdatedAt,
		&key.LastUsedAt,
		&key.IsActive,
		&key.Metadata,
	)

	if err == sql.ErrNoRows {
		return nil, ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find api key by provider: %w", err)
	}

	return key, nil
}

// ListAllActiveAPIKeys returns all active API keys across all users.
// Used during bootstrap to configure plugins with stored keys.
func (a *LocalAdapter) ListAllActiveAPIKeys(ctx context.Context) ([]*APIKey, error) {
	query := `
		SELECT id, user_id, provider, key_name, key_hash, encrypted_key, storage_type, created_at, updated_at, last_used_at, is_active, metadata
		FROM api_keys
		WHERE is_active = 1
		ORDER BY updated_at DESC
	`

	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all active api keys: %w", err)
	}
	defer rows.Close()

	keys := []*APIKey{}
	for rows.Next() {
		key := &APIKey{}
		err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.Provider,
			&key.KeyName,
			&key.KeyHash,
			&key.EncryptedKey,
			&key.StorageType,
			&key.CreatedAt,
			&key.UpdatedAt,
			&key.LastUsedAt,
			&key.IsActive,
			&key.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan api key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate api keys: %w", err)
	}

	return keys, nil
}

func (a *LocalAdapter) DeleteAPIKey(ctx context.Context, userID, provider string) error {
	query := `
		UPDATE api_keys
		SET is_active = 0, updated_at = ?
		WHERE user_id = ? AND provider = ?
	`

	result, err := a.db.ExecContext(ctx, query, time.Now(), userID, provider)
	if err != nil {
		return fmt.Errorf("failed to delete api key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

func (a *LocalAdapter) UpdateAPIKeyLastUsed(ctx context.Context, userID, provider string) error {
	query := `
		UPDATE api_keys
		SET last_used_at = ?
		WHERE user_id = ? AND provider = ? AND is_active = 1
	`

	_, err := a.db.ExecContext(ctx, query, time.Now(), userID, provider)
	if err != nil {
		return fmt.Errorf("failed to update api key last used: %w", err)
	}

	return nil
}

func (a *LocalAdapter) CreateConversation(ctx context.Context, conv *Conversation) error {
	query := `
		INSERT INTO conversations (id, user_id, title, project_id, created_at, updated_at, last_message_at, message_count, is_archived, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := a.db.ExecContext(ctx, query,
		conv.ID,
		conv.UserID,
		conv.Title,
		conv.ProjectID,
		conv.CreatedAt,
		conv.UpdatedAt,
		conv.LastMessageAt,
		conv.MessageCount,
		conv.IsArchived,
		conv.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	return nil
}

func (a *LocalAdapter) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	query := `
		SELECT id, user_id, title, project_id, created_at, updated_at, last_message_at, message_count, is_archived, metadata
		FROM conversations
		WHERE id = ?
	`

	conv := &Conversation{}
	err := a.db.QueryRowContext(ctx, query, id).Scan(
		&conv.ID,
		&conv.UserID,
		&conv.Title,
		&conv.ProjectID,
		&conv.CreatedAt,
		&conv.UpdatedAt,
		&conv.LastMessageAt,
		&conv.MessageCount,
		&conv.IsArchived,
		&conv.Metadata,
	)

	if err == sql.ErrNoRows {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	return conv, nil
}

func (a *LocalAdapter) ListConversations(ctx context.Context, userID string) ([]*Conversation, error) {
	query := `
		SELECT id, user_id, title, project_id, created_at, updated_at, last_message_at, message_count, is_archived, metadata
		FROM conversations
		WHERE user_id = ?
		ORDER BY last_message_at DESC
	`

	rows, err := a.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	convs := []*Conversation{}
	for rows.Next() {
		conv := &Conversation{}
		err := rows.Scan(
			&conv.ID,
			&conv.UserID,
			&conv.Title,
			&conv.ProjectID,
			&conv.CreatedAt,
			&conv.UpdatedAt,
			&conv.LastMessageAt,
			&conv.MessageCount,
			&conv.IsArchived,
			&conv.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		convs = append(convs, conv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate conversations: %w", err)
	}

	return convs, nil
}

func (a *LocalAdapter) UpdateConversation(ctx context.Context, conv *Conversation) error {
	query := `
		UPDATE conversations
		SET title = ?, project_id = ?, updated_at = ?, last_message_at = ?, message_count = ?, is_archived = ?, metadata = ?
		WHERE id = ?
	`

	result, err := a.db.ExecContext(ctx, query,
		conv.Title,
		conv.ProjectID,
		conv.UpdatedAt,
		conv.LastMessageAt,
		conv.MessageCount,
		conv.IsArchived,
		conv.Metadata,
		conv.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrConversationNotFound
	}

	return nil
}

func (a *LocalAdapter) DeleteConversation(ctx context.Context, id string) error {
	query := `DELETE FROM conversations WHERE id = ?`

	result, err := a.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrConversationNotFound
	}

	return nil
}

func (a *LocalAdapter) GetSettings(ctx context.Context, userID string) (*Settings, error) {
	query := `
		SELECT user_id, theme, language, default_model, default_provider, notification_enabled, auto_save_enabled, preferences, created_at, updated_at
		FROM user_settings
		WHERE user_id = ?
	`

	settings := &Settings{}
	err := a.db.QueryRowContext(ctx, query, userID).Scan(
		&settings.UserID,
		&settings.Theme,
		&settings.Language,
		&settings.DefaultModel,
		&settings.DefaultProvider,
		&settings.NotificationEnabled,
		&settings.AutoSaveEnabled,
		&settings.Preferences,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrSettingsNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	return settings, nil
}

func (a *LocalAdapter) CreateSettings(ctx context.Context, settings *Settings) error {
	query := `
		INSERT INTO user_settings (user_id, theme, language, default_model, default_provider, notification_enabled, auto_save_enabled, preferences, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := a.db.ExecContext(ctx, query,
		settings.UserID,
		settings.Theme,
		settings.Language,
		settings.DefaultModel,
		settings.DefaultProvider,
		settings.NotificationEnabled,
		settings.AutoSaveEnabled,
		settings.Preferences,
		settings.CreatedAt,
		settings.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create settings: %w", err)
	}

	return nil
}

func (a *LocalAdapter) UpdateSettings(ctx context.Context, settings *Settings) error {
	query := `
		UPDATE user_settings
		SET theme = ?, language = ?, default_model = ?, default_provider = ?, notification_enabled = ?, auto_save_enabled = ?, preferences = ?, updated_at = ?
		WHERE user_id = ?
	`

	result, err := a.db.ExecContext(ctx, query,
		settings.Theme,
		settings.Language,
		settings.DefaultModel,
		settings.DefaultProvider,
		settings.NotificationEnabled,
		settings.AutoSaveEnabled,
		settings.Preferences,
		settings.UpdatedAt,
		settings.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrSettingsNotFound
	}

	return nil
}

func (a *LocalAdapter) CreateMessage(ctx context.Context, msg *Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO messages (id, conversation_id, role, content, model, provider, tokens_used, created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := a.db.ExecContext(ctx, query,
		msg.ID,
		msg.ConversationID,
		msg.Role,
		msg.Content,
		msg.Model,
		msg.Provider,
		msg.TokensUsed,
		msg.CreatedAt,
		msg.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

func (a *LocalAdapter) ListMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, conversation_id, role, content, model, provider, tokens_used, created_at, metadata
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at ASC
		LIMIT ? OFFSET ?
	`

	rows, err := a.db.QueryContext(ctx, query, conversationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	msgs := []*Message{}
	for rows.Next() {
		msg := &Message{}
		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.Role,
			&msg.Content,
			&msg.Model,
			&msg.Provider,
			&msg.TokensUsed,
			&msg.CreatedAt,
			&msg.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		msgs = append(msgs, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate messages: %w", err)
	}

	return msgs, nil
}

func (a *LocalAdapter) GetMessage(ctx context.Context, id string) (*Message, error) {
	query := `
		SELECT id, conversation_id, role, content, model, provider, tokens_used, created_at, metadata
		FROM messages
		WHERE id = ?
	`

	msg := &Message{}
	err := a.db.QueryRowContext(ctx, query, id).Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.Role,
		&msg.Content,
		&msg.Model,
		&msg.Provider,
		&msg.TokensUsed,
		&msg.CreatedAt,
		&msg.Metadata,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return msg, nil
}

// EnsureMessagesTable creates the messages table if it does not exist.
// This is called from the schema initialization path.
func EnsureMessagesTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS messages (
			id              TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			role            TEXT NOT NULL CHECK(role IN ('user', 'assistant', 'system')),
			content         TEXT NOT NULL,
			model           TEXT,
			provider        TEXT,
			tokens_used     INTEGER,
			created_at      DATETIME NOT NULL,
			metadata        TEXT,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id);
		CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at ASC);
		CREATE INDEX IF NOT EXISTS idx_messages_role ON messages(role);
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}
	return nil
}

func (a *LocalAdapter) Ping(ctx context.Context) error {
	return a.db.PingContext(ctx)
}

func (a *LocalAdapter) Close() error {
	return a.db.Close()
}
