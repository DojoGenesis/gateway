package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PortalUser holds the fields visible to admin management endpoints.
type PortalUser struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreatePortalUser inserts a new user record with hashed credentials.
// Returns the generated user ID (UUID v4).
func CreatePortalUser(db *sql.DB, email, passwordHash, displayName string) (string, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO local_users (id, user_type, created_at, last_accessed_at, email, password_hash, display_name)
		VALUES (?, 'authenticated', ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query, id, now, now, email, passwordHash, displayName)
	if err != nil {
		return "", fmt.Errorf("failed to create portal user: %w", err)
	}

	return id, nil
}

// GetPortalUserByEmail retrieves a user by email for login.
// Returns (userID, passwordHash, displayName, error).
// Returns sql.ErrNoRows if not found.
func GetPortalUserByEmail(db *sql.DB, email string) (id, passwordHash, displayName string, err error) {
	query := `
		SELECT id, password_hash, display_name
		FROM local_users
		WHERE email = ? AND user_type = 'authenticated'
	`

	err = db.QueryRow(query, email).Scan(&id, &passwordHash, &displayName)
	if err != nil {
		return "", "", "", err
	}

	return id, passwordHash, displayName, nil
}

// GetPortalUserByID retrieves a user by ID for token refresh.
// Returns (email, displayName, error).
func GetPortalUserByID(db *sql.DB, userID string) (email, displayName string, err error) {
	query := `
		SELECT email, display_name
		FROM local_users
		WHERE id = ? AND user_type = 'authenticated'
	`

	err = db.QueryRow(query, userID).Scan(&email, &displayName)
	if err != nil {
		return "", "", err
	}

	return email, displayName, nil
}

// ListPortalUsers returns all authenticated portal users for admin management.
func ListPortalUsers(db *sql.DB) ([]PortalUser, error) {
	query := `
		SELECT id, COALESCE(email, ''), COALESCE(display_name, ''), COALESCE(is_active, 1), created_at
		FROM local_users
		WHERE user_type = 'authenticated'
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list portal users: %w", err)
	}
	defer rows.Close()

	var users []PortalUser
	for rows.Next() {
		var u PortalUser
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan portal user: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate portal users: %w", err)
	}

	return users, nil
}

// DeactivatePortalUser sets is_active=0 for the given user.
func DeactivatePortalUser(db *sql.DB, userID string) error {
	result, err := db.Exec(
		`UPDATE local_users SET is_active = 0 WHERE id = ? AND user_type = 'authenticated'`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to deactivate portal user: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ActivatePortalUser sets is_active=1 for the given user.
func ActivatePortalUser(db *sql.DB, userID string) error {
	result, err := db.Exec(
		`UPDATE local_users SET is_active = 1 WHERE id = ? AND user_type = 'authenticated'`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to activate portal user: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
