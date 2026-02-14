package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMigrationInProgress  = errors.New("migration already in progress")
	ErrMigrationNotFound    = errors.New("migration not found")
	ErrNoDataToMigrate      = errors.New("no data to migrate")
	ErrCloudAdapterRequired = errors.New("cloud adapter required for migration")
	ErrMigrationAlreadyDone = errors.New("migration already completed")
)

type MigrationType string

const (
	MigrationTypeFull      MigrationType = "full"
	MigrationTypePartial   MigrationType = "partial"
	MigrationTypeSelective MigrationType = "selective"
)

type MigrationLogStatus string

const (
	MigrationLogStatusRunning   MigrationLogStatus = "running"
	MigrationLogStatusCompleted MigrationLogStatus = "completed"
	MigrationLogStatusFailed    MigrationLogStatus = "failed"
	MigrationLogStatusCancelled MigrationLogStatus = "cancelled"
)

type MigrationLog struct {
	ID              string             `json:"id"`
	UserID          string             `json:"user_id"`
	MigrationType   MigrationType      `json:"migration_type"`
	StartedAt       time.Time          `json:"started_at"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
	Status          MigrationLogStatus `json:"status"`
	RecordsMigrated int                `json:"records_migrated"`
	RecordsTotal    int                `json:"records_total"`
	ProgressPercent float64            `json:"progress_percent"`
	Errors          []string           `json:"errors,omitempty"`
	Metadata        map[string]int     `json:"metadata,omitempty"`
}

type MigrationProgress struct {
	MigrationID     string             `json:"migration_id"`
	Status          MigrationLogStatus `json:"status"`
	RecordsMigrated int                `json:"records_migrated"`
	RecordsTotal    int                `json:"records_total"`
	ProgressPercent float64            `json:"progress_percent"`
	StartedAt       time.Time          `json:"started_at"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
	Errors          []string           `json:"errors,omitempty"`
}

type DataSummary struct {
	APIKeys       int  `json:"api_keys"`
	Conversations int  `json:"conversations"`
	Settings      bool `json:"settings"`
	Total         int  `json:"total"`
}

type MigrationManager struct {
	localDB      *sql.DB
	cloudAdapter DatabaseAdapter
}

func NewMigrationManager(localDB *sql.DB, cloudAdapter DatabaseAdapter) *MigrationManager {
	return &MigrationManager{
		localDB:      localDB,
		cloudAdapter: cloudAdapter,
	}
}

func (m *MigrationManager) GetDataSummary(ctx context.Context, userID string) (*DataSummary, error) {
	summary := &DataSummary{}

	countQuery := `SELECT COUNT(*) FROM api_keys WHERE user_id = ? AND is_active = 1`
	err := m.localDB.QueryRowContext(ctx, countQuery, userID).Scan(&summary.APIKeys)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to count api keys: %w", err)
	}

	countQuery = `SELECT COUNT(*) FROM conversations WHERE user_id = ?`
	err = m.localDB.QueryRowContext(ctx, countQuery, userID).Scan(&summary.Conversations)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to count conversations: %w", err)
	}

	var settingsCount int
	countQuery = `SELECT COUNT(*) FROM user_settings WHERE user_id = ?`
	err = m.localDB.QueryRowContext(ctx, countQuery, userID).Scan(&settingsCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to count settings: %w", err)
	}
	summary.Settings = settingsCount > 0

	summary.Total = summary.APIKeys + summary.Conversations
	if summary.Settings {
		summary.Total++
	}

	return summary, nil
}

func (m *MigrationManager) StartMigration(ctx context.Context, userID, cloudUserID string) (string, error) {
	if m.cloudAdapter == nil {
		return "", ErrCloudAdapterRequired
	}

	existingMigration, err := m.checkExistingMigration(ctx, userID)
	if err != nil && err != ErrMigrationNotFound {
		return "", fmt.Errorf("failed to check existing migration: %w", err)
	}
	if existingMigration != nil {
		if existingMigration.Status == MigrationLogStatusRunning {
			return "", ErrMigrationInProgress
		}
		if existingMigration.Status == MigrationLogStatusCompleted {
			return "", ErrMigrationAlreadyDone
		}
	}

	summary, err := m.GetDataSummary(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get data summary: %w", err)
	}

	if summary.Total == 0 {
		return "", ErrNoDataToMigrate
	}

	migrationID := uuid.New().String()
	now := time.Now()

	insertQuery := `
		INSERT INTO migration_log (id, user_id, migration_type, started_at, status, records_total, records_migrated, progress_percent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = m.localDB.ExecContext(ctx, insertQuery,
		migrationID,
		userID,
		MigrationTypeFull,
		now,
		MigrationLogStatusRunning,
		summary.Total,
		0,
		0.0,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create migration log: %w", err)
	}

	go m.performMigration(context.Background(), migrationID, userID, cloudUserID, summary.Total)

	return migrationID, nil
}

func (m *MigrationManager) performMigration(ctx context.Context, migrationID, userID, cloudUserID string, totalRecords int) {
	var migrationErrors []string
	recordsMigrated := 0

	tx, err := m.localDB.BeginTx(ctx, nil)
	if err != nil {
		m.updateMigrationStatus(ctx, migrationID, MigrationLogStatusFailed, recordsMigrated, totalRecords, []string{fmt.Sprintf("failed to start transaction: %v", err)})
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			m.updateMigrationStatus(ctx, migrationID, MigrationLogStatusFailed, recordsMigrated, totalRecords, append(migrationErrors, fmt.Sprintf("panic during migration: %v", r)))
		}
	}()

	metadata := make(map[string]int)

	localAdapter := NewLocalAdapter(m.localDB)

	user, err := localAdapter.GetUser(ctx, userID)
	if err != nil {
		tx.Rollback()
		m.updateMigrationStatus(ctx, migrationID, MigrationLogStatusFailed, recordsMigrated, totalRecords, []string{fmt.Sprintf("failed to get user: %v", err)})
		return
	}

	cloudUser := &User{
		ID:              cloudUserID,
		UserType:        UserTypeAuthenticated,
		CreatedAt:       user.CreatedAt,
		LastAccessedAt:  time.Now(),
		CloudUserID:     &cloudUserID,
		MigrationStatus: MigrationStatusCompleted,
		Metadata:        user.Metadata,
	}

	err = m.cloudAdapter.CreateUser(ctx, cloudUser)
	if err != nil && !errors.Is(err, ErrCloudAdapterNotImplemented) {
		migrationErrors = append(migrationErrors, fmt.Sprintf("failed to create cloud user: %v", err))
	}

	apiKeys, err := localAdapter.ListAPIKeys(ctx, userID)
	if err != nil && err != ErrAPIKeyNotFound {
		migrationErrors = append(migrationErrors, fmt.Sprintf("failed to list api keys: %v", err))
	} else {
		for _, key := range apiKeys {
			cloudKey := &APIKey{
				ID:           key.ID,
				UserID:       cloudUserID,
				Provider:     key.Provider,
				KeyName:      key.KeyName,
				KeyHash:      key.KeyHash,
				EncryptedKey: key.EncryptedKey,
				StorageType:  key.StorageType,
				CreatedAt:    key.CreatedAt,
				UpdatedAt:    key.UpdatedAt,
				LastUsedAt:   key.LastUsedAt,
				IsActive:     key.IsActive,
				Metadata:     key.Metadata,
			}

			err := m.cloudAdapter.StoreAPIKey(ctx, cloudKey)
			if err != nil && !errors.Is(err, ErrCloudAdapterNotImplemented) {
				migrationErrors = append(migrationErrors, fmt.Sprintf("failed to migrate api key %s: %v", key.Provider, err))
			} else if err == nil || errors.Is(err, ErrCloudAdapterNotImplemented) {
				recordsMigrated++
			}
		}
		metadata["api_keys"] = len(apiKeys)
		m.updateMigrationProgress(ctx, migrationID, recordsMigrated, totalRecords)
	}

	conversations, err := localAdapter.ListConversations(ctx, userID)
	if err != nil && err != ErrConversationNotFound {
		migrationErrors = append(migrationErrors, fmt.Sprintf("failed to list conversations: %v", err))
	} else {
		for _, conv := range conversations {
			cloudConv := &Conversation{
				ID:            conv.ID,
				UserID:        cloudUserID,
				Title:         conv.Title,
				ProjectID:     conv.ProjectID,
				CreatedAt:     conv.CreatedAt,
				UpdatedAt:     conv.UpdatedAt,
				LastMessageAt: conv.LastMessageAt,
				MessageCount:  conv.MessageCount,
				IsArchived:    conv.IsArchived,
				Metadata:      conv.Metadata,
			}

			err := m.cloudAdapter.CreateConversation(ctx, cloudConv)
			if err != nil && !errors.Is(err, ErrCloudAdapterNotImplemented) {
				migrationErrors = append(migrationErrors, fmt.Sprintf("failed to migrate conversation %s: %v", conv.ID, err))
			} else if err == nil || errors.Is(err, ErrCloudAdapterNotImplemented) {
				recordsMigrated++
			}
		}
		metadata["conversations"] = len(conversations)
		m.updateMigrationProgress(ctx, migrationID, recordsMigrated, totalRecords)
	}

	settings, err := localAdapter.GetSettings(ctx, userID)
	if err != nil && err != ErrSettingsNotFound {
		migrationErrors = append(migrationErrors, fmt.Sprintf("failed to get settings: %v", err))
	} else if err == nil {
		cloudSettings := &Settings{
			UserID:              cloudUserID,
			Theme:               settings.Theme,
			Language:            settings.Language,
			DefaultModel:        settings.DefaultModel,
			DefaultProvider:     settings.DefaultProvider,
			NotificationEnabled: settings.NotificationEnabled,
			AutoSaveEnabled:     settings.AutoSaveEnabled,
			Preferences:         settings.Preferences,
			CreatedAt:           settings.CreatedAt,
			UpdatedAt:           time.Now(),
		}

		err := m.cloudAdapter.CreateSettings(ctx, cloudSettings)
		if err != nil && !errors.Is(err, ErrCloudAdapterNotImplemented) {
			migrationErrors = append(migrationErrors, fmt.Sprintf("failed to migrate settings: %v", err))
		} else if err == nil || errors.Is(err, ErrCloudAdapterNotImplemented) {
			recordsMigrated++
		}
		metadata["settings"] = 1
		m.updateMigrationProgress(ctx, migrationID, recordsMigrated, totalRecords)
	}

	if err := tx.Commit(); err != nil {
		m.updateMigrationStatus(ctx, migrationID, MigrationLogStatusFailed, recordsMigrated, totalRecords, append(migrationErrors, fmt.Sprintf("failed to commit transaction: %v", err)))
		return
	}

	user.MigrationStatus = MigrationStatusCompleted
	user.CloudUserID = &cloudUserID
	if err := localAdapter.UpdateUser(ctx, user); err != nil {
		migrationErrors = append(migrationErrors, fmt.Sprintf("failed to update user migration status: %v", err))
	}

	finalStatus := MigrationLogStatusCompleted
	if len(migrationErrors) > 0 {
		finalStatus = MigrationLogStatusFailed
	}

	m.updateMigrationStatusWithMetadata(ctx, migrationID, finalStatus, recordsMigrated, totalRecords, migrationErrors, metadata)
}

func (m *MigrationManager) updateMigrationProgress(ctx context.Context, migrationID string, recordsMigrated, totalRecords int) {
	progressPercent := 0.0
	if totalRecords > 0 {
		progressPercent = float64(recordsMigrated) / float64(totalRecords) * 100.0
	}

	updateQuery := `
		UPDATE migration_log
		SET records_migrated = ?, progress_percent = ?
		WHERE id = ?
	`
	_, err := m.localDB.ExecContext(ctx, updateQuery, recordsMigrated, progressPercent, migrationID)
	if err != nil {
		slog.Warn("failed to update migration progress", "error", err)
	}
}

func (m *MigrationManager) updateMigrationStatus(ctx context.Context, migrationID string, status MigrationLogStatus, recordsMigrated, totalRecords int, errors []string) {
	m.updateMigrationStatusWithMetadata(ctx, migrationID, status, recordsMigrated, totalRecords, errors, nil)
}

func (m *MigrationManager) updateMigrationStatusWithMetadata(ctx context.Context, migrationID string, status MigrationLogStatus, recordsMigrated, totalRecords int, errorList []string, metadata map[string]int) {
	now := time.Now()
	progressPercent := 0.0
	if totalRecords > 0 {
		progressPercent = float64(recordsMigrated) / float64(totalRecords) * 100.0
	}

	var errorsJSON *string
	if len(errorList) > 0 {
		errorsBytes, err := json.Marshal(errorList)
		if err == nil {
			errorsStr := string(errorsBytes)
			errorsJSON = &errorsStr
		}
	}

	var metadataJSON *string
	if metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err == nil {
			metadataStr := string(metadataBytes)
			metadataJSON = &metadataStr
		}
	}

	updateQuery := `
		UPDATE migration_log
		SET status = ?, records_migrated = ?, progress_percent = ?, completed_at = ?, errors = ?, metadata = ?
		WHERE id = ?
	`
	_, err := m.localDB.ExecContext(ctx, updateQuery, status, recordsMigrated, progressPercent, now, errorsJSON, metadataJSON, migrationID)
	if err != nil {
		slog.Warn("failed to update migration status", "error", err)
	}
}

func (m *MigrationManager) GetMigrationStatus(ctx context.Context, migrationID string) (*MigrationProgress, error) {
	query := `
		SELECT id, status, records_migrated, records_total, progress_percent, started_at, completed_at, errors
		FROM migration_log
		WHERE id = ?
	`

	var progress MigrationProgress
	var completedAt sql.NullTime
	var errorsJSON sql.NullString

	err := m.localDB.QueryRowContext(ctx, query, migrationID).Scan(
		&progress.MigrationID,
		&progress.Status,
		&progress.RecordsMigrated,
		&progress.RecordsTotal,
		&progress.ProgressPercent,
		&progress.StartedAt,
		&completedAt,
		&errorsJSON,
	)

	if err == sql.ErrNoRows {
		return nil, ErrMigrationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}

	if completedAt.Valid {
		progress.CompletedAt = &completedAt.Time
	}

	if errorsJSON.Valid && errorsJSON.String != "" {
		var errors []string
		if err := json.Unmarshal([]byte(errorsJSON.String), &errors); err == nil {
			progress.Errors = errors
		}
	}

	return &progress, nil
}

func (m *MigrationManager) GetLatestMigration(ctx context.Context, userID string) (*MigrationProgress, error) {
	query := `
		SELECT id, status, records_migrated, records_total, progress_percent, started_at, completed_at, errors
		FROM migration_log
		WHERE user_id = ?
		ORDER BY started_at DESC
		LIMIT 1
	`

	var progress MigrationProgress
	var completedAt sql.NullTime
	var errorsJSON sql.NullString

	err := m.localDB.QueryRowContext(ctx, query, userID).Scan(
		&progress.MigrationID,
		&progress.Status,
		&progress.RecordsMigrated,
		&progress.RecordsTotal,
		&progress.ProgressPercent,
		&progress.StartedAt,
		&completedAt,
		&errorsJSON,
	)

	if err == sql.ErrNoRows {
		return nil, ErrMigrationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest migration: %w", err)
	}

	if completedAt.Valid {
		progress.CompletedAt = &completedAt.Time
	}

	if errorsJSON.Valid && errorsJSON.String != "" {
		var errors []string
		if err := json.Unmarshal([]byte(errorsJSON.String), &errors); err == nil {
			progress.Errors = errors
		}
	}

	return &progress, nil
}

func (m *MigrationManager) CancelMigration(ctx context.Context, migrationID string) error {
	query := `
		UPDATE migration_log
		SET status = ?, completed_at = ?
		WHERE id = ? AND status = ?
	`

	result, err := m.localDB.ExecContext(ctx, query, MigrationLogStatusCancelled, time.Now(), migrationID, MigrationLogStatusRunning)
	if err != nil {
		return fmt.Errorf("failed to cancel migration: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrMigrationNotFound
	}

	return nil
}

func (m *MigrationManager) checkExistingMigration(ctx context.Context, userID string) (*MigrationProgress, error) {
	return m.GetLatestMigration(ctx, userID)
}
