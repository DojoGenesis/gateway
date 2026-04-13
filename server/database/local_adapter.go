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

// ─── Prompt Templates ────────────────────────────────────────────────────────

func (a *LocalAdapter) CreatePromptTemplate(ctx context.Context, tmpl *PromptTemplate) error {
	query := `
		INSERT INTO prompt_templates (id, user_id, title, description, system_prompt, is_public, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := a.db.ExecContext(ctx, query,
		tmpl.ID,
		tmpl.UserID,
		tmpl.Title,
		tmpl.Description,
		tmpl.SystemPrompt,
		tmpl.IsPublic,
		tmpl.CreatedAt,
		tmpl.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create prompt template: %w", err)
	}
	return nil
}

func (a *LocalAdapter) GetPromptTemplate(ctx context.Context, id string) (*PromptTemplate, error) {
	query := `
		SELECT id, user_id, title, description, system_prompt, is_public, created_at, updated_at
		FROM prompt_templates
		WHERE id = ?
	`

	tmpl := &PromptTemplate{}
	err := a.db.QueryRowContext(ctx, query, id).Scan(
		&tmpl.ID,
		&tmpl.UserID,
		&tmpl.Title,
		&tmpl.Description,
		&tmpl.SystemPrompt,
		&tmpl.IsPublic,
		&tmpl.CreatedAt,
		&tmpl.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrTemplateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt template: %w", err)
	}

	return tmpl, nil
}

func (a *LocalAdapter) ListPromptTemplates(ctx context.Context, userID string, includePublic bool) ([]*PromptTemplate, error) {
	var query string
	var args []interface{}

	if includePublic {
		query = `
			SELECT id, user_id, title, description, system_prompt, is_public, created_at, updated_at
			FROM prompt_templates
			WHERE user_id = ? OR is_public = 1
			ORDER BY updated_at DESC
		`
		args = []interface{}{userID}
	} else {
		query = `
			SELECT id, user_id, title, description, system_prompt, is_public, created_at, updated_at
			FROM prompt_templates
			WHERE user_id = ?
			ORDER BY updated_at DESC
		`
		args = []interface{}{userID}
	}

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompt templates: %w", err)
	}
	defer rows.Close()

	tmpls := []*PromptTemplate{}
	for rows.Next() {
		tmpl := &PromptTemplate{}
		err := rows.Scan(
			&tmpl.ID,
			&tmpl.UserID,
			&tmpl.Title,
			&tmpl.Description,
			&tmpl.SystemPrompt,
			&tmpl.IsPublic,
			&tmpl.CreatedAt,
			&tmpl.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan prompt template: %w", err)
		}
		tmpls = append(tmpls, tmpl)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate prompt templates: %w", err)
	}

	return tmpls, nil
}

func (a *LocalAdapter) UpdatePromptTemplate(ctx context.Context, tmpl *PromptTemplate) error {
	query := `
		UPDATE prompt_templates
		SET title = ?, description = ?, system_prompt = ?, is_public = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := a.db.ExecContext(ctx, query,
		tmpl.Title,
		tmpl.Description,
		tmpl.SystemPrompt,
		tmpl.IsPublic,
		tmpl.UpdatedAt,
		tmpl.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update prompt template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

func (a *LocalAdapter) DeletePromptTemplate(ctx context.Context, id string) error {
	query := `DELETE FROM prompt_templates WHERE id = ?`

	result, err := a.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete prompt template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

// EnsurePromptTemplatesTable creates the prompt_templates table if it does not exist.
// This is called from the schema initialization path.
func EnsurePromptTemplatesTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS prompt_templates (
			id            TEXT PRIMARY KEY,
			user_id       TEXT NOT NULL,
			title         TEXT NOT NULL,
			description   TEXT DEFAULT '',
			system_prompt TEXT NOT NULL,
			is_public     BOOLEAN DEFAULT 0,
			created_at    DATETIME NOT NULL,
			updated_at    DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_prompt_templates_user ON prompt_templates(user_id);
		CREATE INDEX IF NOT EXISTS idx_prompt_templates_public ON prompt_templates(is_public) WHERE is_public = 1;
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create prompt_templates table: %w", err)
	}
	return nil
}

// SeedDefaultTemplates inserts pre-seeded public system templates if they don't exist.
// Templates are keyed by title + user_id = "system" to remain idempotent on restart.
func SeedDefaultTemplates(db *sql.DB) error {
	type seedEntry struct {
		title        string
		description  string
		systemPrompt string
	}

	seeds := []seedEntry{
		{
			title:       "Grant Writing Assistant",
			description: "Expert grant writer specializing in community development and social equity funding.",
			systemPrompt: "You are an expert grant writer specializing in community development and social equity funding. " +
				"Help draft compelling narratives that connect data to impact. Reference specific metrics and cite data sources. " +
				"Focus on logic models, theory of change, and measurable outcomes.",
		},
		{
			title:       "Atlas Data Analyst",
			description: "Spatial equity data analyst with expertise in the Madison Equity Atlas methodology.",
			systemPrompt: "You are a spatial equity data analyst. You have expertise in the Madison Equity Atlas methodology " +
				"including the Neighborhood Attendance Risk Index (NARI), Blinder-Oaxaca decomposition, and OLS regression for " +
				"chronic absenteeism drivers. Interpret results in plain language for policymakers. When discussing statistics, " +
				"always note confidence levels and practical significance.",
		},
		{
			title:       "Policy Brief Drafter",
			description: "Policy brief writer for Common Wealth Development (CWD), writing for Madison and Dane County officials.",
			systemPrompt: "You are a policy brief writer for Common Wealth Development (CWD), a community development organization " +
				"in Madison, WI. Write for the audience of Madison City Council members and Dane County Board supervisors. " +
				"Use clear, accessible language. Structure briefs with: Executive Summary, Background, Key Findings, " +
				"Recommendations, and Call to Action.",
		},
		{
			title:       "Meeting Summarizer",
			description: "Summarizes meeting notes into a structured format with decisions, action items, and next steps.",
			systemPrompt: "Summarize meeting notes into a structured format: 1) Key Decisions Made, 2) Action Items (with owners " +
				"and deadlines), 3) Open Questions, 4) Next Steps. Be concise. Use bullet points. Flag any decisions that seem " +
				"to contradict prior commitments.",
		},
	}

	now := time.Now()
	for _, s := range seeds {
		// Check if a system template with this title already exists.
		var count int
		err := db.QueryRow(
			`SELECT COUNT(*) FROM prompt_templates WHERE user_id = 'system' AND title = ?`,
			s.title,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check existing seed template %q: %w", s.title, err)
		}
		if count > 0 {
			continue
		}

		_, err = db.Exec(
			`INSERT INTO prompt_templates (id, user_id, title, description, system_prompt, is_public, created_at, updated_at)
			 VALUES (?, 'system', ?, ?, ?, 1, ?, ?)`,
			uuid.New().String(),
			s.title,
			s.description,
			s.systemPrompt,
			now,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to seed template %q: %w", s.title, err)
		}
	}

	return nil
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

// ─── Document / RAG ──────────────────────────────────────────────────────────

// EnsureDocumentsTable creates the documents, document_chunks, and FTS tables if they do not exist.
// FTS5 support is probed at runtime; if unavailable we fall back to plain LIKE search.
func EnsureDocumentsTable(db *sql.DB) error {
	// Core documents table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
			id           TEXT PRIMARY KEY,
			user_id      TEXT NOT NULL,
			filename     TEXT NOT NULL,
			content_type TEXT NOT NULL DEFAULT 'text/plain',
			size_bytes   INTEGER NOT NULL DEFAULT 0,
			chunk_count  INTEGER NOT NULL DEFAULT 0,
			status       TEXT NOT NULL DEFAULT 'processing',
			created_at   DATETIME NOT NULL,
			metadata     TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_documents_user ON documents(user_id);
	`)
	if err != nil {
		return fmt.Errorf("failed to create documents table: %w", err)
	}

	// Chunks table (regular, for retrieval)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS document_chunks (
			id          TEXT PRIMARY KEY,
			document_id TEXT NOT NULL,
			chunk_index INTEGER NOT NULL,
			content     TEXT NOT NULL,
			created_at  DATETIME NOT NULL,
			FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_chunks_document ON document_chunks(document_id);
	`)
	if err != nil {
		return fmt.Errorf("failed to create document_chunks table: %w", err)
	}

	// Attempt FTS5 virtual table creation. modernc.org/sqlite ships with FTS5.
	// If it fails for any reason we leave it absent and fall back to LIKE.
	_, ftsErr := db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS document_chunks_fts USING fts5(
			content,
			chunk_id   UNINDEXED,
			document_id UNINDEXED,
			chunk_index UNINDEXED
		);
	`)
	if ftsErr != nil {
		// FTS5 not available — log and continue. Searches will use LIKE.
		_ = ftsErr
	}

	return nil
}

func (a *LocalAdapter) CreateDocument(ctx context.Context, doc *Document) error {
	_, err := a.db.ExecContext(ctx, `
		INSERT INTO documents (id, user_id, filename, content_type, size_bytes, chunk_count, status, created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doc.ID, doc.UserID, doc.Filename, doc.ContentType,
		doc.SizeBytes, doc.ChunkCount, doc.Status, doc.CreatedAt, doc.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}
	return nil
}

func (a *LocalAdapter) GetDocument(ctx context.Context, id string) (*Document, error) {
	doc := &Document{}
	err := a.db.QueryRowContext(ctx, `
		SELECT id, user_id, filename, content_type, size_bytes, chunk_count, status, created_at, metadata
		FROM documents WHERE id = ?`, id,
	).Scan(&doc.ID, &doc.UserID, &doc.Filename, &doc.ContentType,
		&doc.SizeBytes, &doc.ChunkCount, &doc.Status, &doc.CreatedAt, &doc.Metadata)
	if err == sql.ErrNoRows {
		return nil, ErrDocumentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	return doc, nil
}

func (a *LocalAdapter) ListDocuments(ctx context.Context, userID string) ([]*Document, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT id, user_id, filename, content_type, size_bytes, chunk_count, status, created_at, metadata
		FROM documents WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer rows.Close()

	docs := []*Document{}
	for rows.Next() {
		doc := &Document{}
		if err := rows.Scan(&doc.ID, &doc.UserID, &doc.Filename, &doc.ContentType,
			&doc.SizeBytes, &doc.ChunkCount, &doc.Status, &doc.CreatedAt, &doc.Metadata); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

func (a *LocalAdapter) DeleteDocument(ctx context.Context, id string) error {
	// Delete FTS entries first (no CASCADE on virtual table)
	_, _ = a.db.ExecContext(ctx, `DELETE FROM document_chunks_fts WHERE document_id = ?`, id)

	result, err := a.db.ExecContext(ctx, `DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrDocumentNotFound
	}
	return nil
}

func (a *LocalAdapter) UpdateDocumentStatus(ctx context.Context, id string, status string, chunkCount int) error {
	result, err := a.db.ExecContext(ctx,
		`UPDATE documents SET status = ?, chunk_count = ? WHERE id = ?`,
		status, chunkCount, id)
	if err != nil {
		return fmt.Errorf("failed to update document status: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrDocumentNotFound
	}
	return nil
}

// hasFTS5 probes whether the document_chunks_fts virtual table exists.
func (a *LocalAdapter) hasFTS5(ctx context.Context) bool {
	var name string
	err := a.db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name='document_chunks_fts'`,
	).Scan(&name)
	return err == nil && name != ""
}

func (a *LocalAdapter) CreateDocumentChunks(ctx context.Context, chunks []*DocumentChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmtChunks, err := tx.PrepareContext(ctx, `
		INSERT INTO document_chunks (id, document_id, chunk_index, content, created_at)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare chunk insert: %w", err)
	}
	defer stmtChunks.Close()

	// FTS5 insert — prepared separately; may be nil if FTS5 unavailable
	var stmtFTS *sql.Stmt
	if a.hasFTS5(ctx) {
		stmtFTS, err = tx.PrepareContext(ctx, `
			INSERT INTO document_chunks_fts (content, chunk_id, document_id, chunk_index)
			VALUES (?, ?, ?, ?)`)
		if err != nil {
			// FTS5 write failed — proceed without FTS (search falls back to LIKE)
			stmtFTS = nil
		}
	}
	if stmtFTS != nil {
		defer stmtFTS.Close()
	}

	for _, c := range chunks {
		if c.ID == "" {
			c.ID = uuid.New().String()
		}
		if c.CreatedAt.IsZero() {
			c.CreatedAt = time.Now()
		}
		if _, err := stmtChunks.ExecContext(ctx, c.ID, c.DocumentID, c.ChunkIndex, c.Content, c.CreatedAt); err != nil {
			return fmt.Errorf("failed to insert chunk %d: %w", c.ChunkIndex, err)
		}
		if stmtFTS != nil {
			if _, err := stmtFTS.ExecContext(ctx, c.Content, c.ID, c.DocumentID, c.ChunkIndex); err != nil {
				// Non-fatal: FTS insert failure shouldn't abort the whole upload
				_ = err
			}
		}
	}

	return tx.Commit()
}

func (a *LocalAdapter) SearchDocumentChunks(ctx context.Context, userID string, query string, limit int) ([]*DocumentChunk, error) {
	if limit <= 0 {
		limit = 5
	}

	var (
		rows *sql.Rows
		err  error
	)

	if a.hasFTS5(ctx) {
		rows, err = a.db.QueryContext(ctx, `
			SELECT dc.id, dc.document_id, dc.chunk_index, dc.content, dc.created_at
			FROM document_chunks_fts fts
			JOIN document_chunks dc ON dc.id = fts.chunk_id
			JOIN documents d ON d.id = dc.document_id
			WHERE d.user_id = ? AND document_chunks_fts MATCH ?
			ORDER BY rank
			LIMIT ?`,
			userID, query, limit)
	} else {
		// Fallback: simple LIKE search
		rows, err = a.db.QueryContext(ctx, `
			SELECT dc.id, dc.document_id, dc.chunk_index, dc.content, dc.created_at
			FROM document_chunks dc
			JOIN documents d ON d.id = dc.document_id
			WHERE d.user_id = ? AND dc.content LIKE ?
			LIMIT ?`,
			userID, "%"+query+"%", limit)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}
	defer rows.Close()

	chunks := []*DocumentChunk{}
	for rows.Next() {
		c := &DocumentChunk{}
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.ChunkIndex, &c.Content, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

func (a *LocalAdapter) GetDocumentChunks(ctx context.Context, documentID string) ([]*DocumentChunk, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT id, document_id, chunk_index, content, created_at
		FROM document_chunks WHERE document_id = ? ORDER BY chunk_index ASC`, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document chunks: %w", err)
	}
	defer rows.Close()

	chunks := []*DocumentChunk{}
	for rows.Next() {
		c := &DocumentChunk{}
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.ChunkIndex, &c.Content, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

func (a *LocalAdapter) Ping(ctx context.Context) error {
	return a.db.PingContext(ctx)
}

func (a *LocalAdapter) Close() error {
	return a.db.Close()
}
