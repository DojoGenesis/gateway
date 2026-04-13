package database

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrAPIKeyNotFound         = errors.New("api key not found")
	ErrConversationNotFound   = errors.New("conversation not found")
	ErrSettingsNotFound       = errors.New("settings not found")
	ErrDuplicateAPIKey        = errors.New("api key already exists for provider")
	ErrInvalidUserType        = errors.New("invalid user type")
	ErrInvalidMigrationStatus = errors.New("invalid migration status")
)

type UserType string

const (
	UserTypeGuest         UserType = "guest"
	UserTypeAuthenticated UserType = "authenticated"
)

type MigrationStatus string

const (
	MigrationStatusNone       MigrationStatus = "none"
	MigrationStatusPending    MigrationStatus = "pending"
	MigrationStatusInProgress MigrationStatus = "in_progress"
	MigrationStatusCompleted  MigrationStatus = "completed"
	MigrationStatusFailed     MigrationStatus = "failed"
)

type User struct {
	ID              string          `json:"id"`
	UserType        UserType        `json:"user_type"`
	CreatedAt       time.Time       `json:"created_at"`
	LastAccessedAt  time.Time       `json:"last_accessed_at"`
	CloudUserID     *string         `json:"cloud_user_id,omitempty"`
	MigrationStatus MigrationStatus `json:"migration_status"`
	Metadata        *string         `json:"metadata,omitempty"`
}

type APIKey struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	Provider     string     `json:"provider"`
	KeyName      *string    `json:"key_name,omitempty"`
	KeyHash      string     `json:"key_hash"`
	EncryptedKey []byte     `json:"-"`
	StorageType  string     `json:"storage_type"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	IsActive     bool       `json:"is_active"`
	Metadata     *string    `json:"metadata,omitempty"`
}

type Conversation struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	Title         *string    `json:"title,omitempty"`
	ProjectID     *string    `json:"project_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	MessageCount  int        `json:"message_count"`
	IsArchived    bool       `json:"is_archived"`
	Metadata      *string    `json:"metadata,omitempty"`
}

type Settings struct {
	UserID              string    `json:"user_id"`
	Theme               string    `json:"theme"`
	Language            string    `json:"language"`
	DefaultModel        *string   `json:"default_model,omitempty"`
	DefaultProvider     *string   `json:"default_provider,omitempty"`
	NotificationEnabled bool      `json:"notification_enabled"`
	AutoSaveEnabled     bool      `json:"auto_save_enabled"`
	Preferences         *string   `json:"preferences,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"` // "user", "assistant", "system"
	Content        string    `json:"content"`
	Model          *string   `json:"model,omitempty"`
	Provider       *string   `json:"provider,omitempty"`
	TokensUsed     *int      `json:"tokens_used,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	Metadata       *string   `json:"metadata,omitempty"`
}

type DatabaseAdapter interface {
	GetUser(ctx context.Context, userID string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error

	StoreAPIKey(ctx context.Context, key *APIKey) error
	GetAPIKey(ctx context.Context, userID, provider string) (*APIKey, error)
	ListAPIKeys(ctx context.Context, userID string) ([]*APIKey, error)
	DeleteAPIKey(ctx context.Context, userID, provider string) error
	UpdateAPIKeyLastUsed(ctx context.Context, userID, provider string) error

	CreateConversation(ctx context.Context, conv *Conversation) error
	GetConversation(ctx context.Context, id string) (*Conversation, error)
	ListConversations(ctx context.Context, userID string) ([]*Conversation, error)
	UpdateConversation(ctx context.Context, conv *Conversation) error
	DeleteConversation(ctx context.Context, id string) error

	GetSettings(ctx context.Context, userID string) (*Settings, error)
	CreateSettings(ctx context.Context, settings *Settings) error
	UpdateSettings(ctx context.Context, settings *Settings) error

	CreateMessage(ctx context.Context, msg *Message) error
	ListMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error)
	GetMessage(ctx context.Context, id string) (*Message, error)

	Ping(ctx context.Context) error
	Close() error
}
