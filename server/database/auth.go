package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

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
