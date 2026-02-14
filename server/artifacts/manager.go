package artifacts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type ArtifactManager struct {
	db *sql.DB
}

func NewArtifactManager(db *sql.DB) (*ArtifactManager, error) {
	am := &ArtifactManager{
		db: db,
	}
	return am, nil
}

func (am *ArtifactManager) CreateArtifact(ctx context.Context, projectID, sessionID string, artifactType ArtifactType, name, description, content string) (*Artifact, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if artifactType == "" {
		return nil, fmt.Errorf("type is required")
	}

	now := time.Now()
	artifact := &Artifact{
		ID:            uuid.New().String(),
		ProjectID:     projectID,
		SessionID:     sessionID,
		Type:          artifactType,
		Name:          name,
		Description:   description,
		LatestVersion: 1,
		CreatedAt:     now,
		UpdatedAt:     now,
		Metadata:      make(map[string]interface{}),
	}

	metadataJSON, err := json.Marshal(artifact.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	tx, err := am.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	artifactQuery := `
		INSERT INTO artifacts (id, project_id, session_id, type, name, description, latest_version, created_at, updated_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.ExecContext(ctx, artifactQuery,
		artifact.ID,
		artifact.ProjectID,
		artifact.SessionID,
		artifact.Type,
		artifact.Name,
		artifact.Description,
		artifact.LatestVersion,
		artifact.CreatedAt,
		artifact.UpdatedAt,
		string(metadataJSON),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create artifact: %w", err)
	}

	version := &ArtifactVersion{
		ID:            uuid.New().String(),
		ArtifactID:    artifact.ID,
		Version:       1,
		Content:       content,
		CommitMessage: "Initial version",
		CreatedAt:     now,
		CreatedBy:     CreatedByAgent,
		Metadata:      make(map[string]interface{}),
	}

	versionMetadataJSON, err := json.Marshal(version.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal version metadata: %w", err)
	}

	versionQuery := `
		INSERT INTO artifact_versions (id, artifact_id, version, content, diff, commit_message, created_at, created_by, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.ExecContext(ctx, versionQuery,
		version.ID,
		version.ArtifactID,
		version.Version,
		version.Content,
		"",
		version.CommitMessage,
		version.CreatedAt,
		version.CreatedBy,
		string(versionMetadataJSON),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create artifact version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return artifact, nil
}

func (am *ArtifactManager) UpdateArtifact(ctx context.Context, artifactID, newContent, commitMessage string) (*Artifact, error) {
	if artifactID == "" {
		return nil, fmt.Errorf("artifact_id is required")
	}

	artifact, err := am.GetArtifact(ctx, artifactID)
	if err != nil {
		return nil, err
	}

	previousVersion, err := am.GetArtifactVersion(ctx, artifactID, artifact.LatestVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous version: %w", err)
	}

	diff, err := CalculateDiff(previousVersion.Content, newContent)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate diff: %w", err)
	}

	newVersionNum := artifact.LatestVersion + 1
	now := time.Now()

	tx, err := am.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	updateArtifactQuery := `
		UPDATE artifacts 
		SET latest_version = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = tx.ExecContext(ctx, updateArtifactQuery, newVersionNum, now, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to update artifact: %w", err)
	}

	version := &ArtifactVersion{
		ID:            uuid.New().String(),
		ArtifactID:    artifactID,
		Version:       newVersionNum,
		Content:       newContent,
		Diff:          diff,
		CommitMessage: commitMessage,
		CreatedAt:     now,
		CreatedBy:     CreatedByAgent,
		Metadata:      make(map[string]interface{}),
	}

	versionMetadataJSON, err := json.Marshal(version.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal version metadata: %w", err)
	}

	insertVersionQuery := `
		INSERT INTO artifact_versions (id, artifact_id, version, content, diff, commit_message, created_at, created_by, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.ExecContext(ctx, insertVersionQuery,
		version.ID,
		version.ArtifactID,
		version.Version,
		version.Content,
		version.Diff,
		version.CommitMessage,
		version.CreatedAt,
		version.CreatedBy,
		string(versionMetadataJSON),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert new version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	artifact.LatestVersion = newVersionNum
	artifact.UpdatedAt = now

	return artifact, nil
}

func (am *ArtifactManager) GetArtifact(ctx context.Context, id string) (*Artifact, error) {
	query := `SELECT id, project_id, session_id, type, name, description, latest_version, created_at, updated_at, metadata 
	          FROM artifacts WHERE id = ?`

	var artifact Artifact
	var metadataJSON string
	var sessionID sql.NullString
	var description sql.NullString

	err := am.db.QueryRowContext(ctx, query, id).Scan(
		&artifact.ID,
		&artifact.ProjectID,
		&sessionID,
		&artifact.Type,
		&artifact.Name,
		&description,
		&artifact.LatestVersion,
		&artifact.CreatedAt,
		&artifact.UpdatedAt,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("artifact not found: %s", id)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve artifact: %w", err)
	}

	if sessionID.Valid {
		artifact.SessionID = sessionID.String
	}

	if description.Valid {
		artifact.Description = description.String
	}

	if err := json.Unmarshal([]byte(metadataJSON), &artifact.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &artifact, nil
}

func (am *ArtifactManager) GetArtifactVersion(ctx context.Context, artifactID string, version int) (*ArtifactVersion, error) {
	query := `SELECT id, artifact_id, version, content, diff, commit_message, created_at, created_by, metadata 
	          FROM artifact_versions WHERE artifact_id = ? AND version = ?`

	var av ArtifactVersion
	var metadataJSON string
	var diff sql.NullString
	var commitMessage sql.NullString
	var createdBy sql.NullString

	err := am.db.QueryRowContext(ctx, query, artifactID, version).Scan(
		&av.ID,
		&av.ArtifactID,
		&av.Version,
		&av.Content,
		&diff,
		&commitMessage,
		&av.CreatedAt,
		&createdBy,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("artifact version not found: %s v%d", artifactID, version)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve artifact version: %w", err)
	}

	if diff.Valid {
		av.Diff = diff.String
	}

	if commitMessage.Valid {
		av.CommitMessage = commitMessage.String
	}

	if createdBy.Valid {
		av.CreatedBy = createdBy.String
	}

	if err := json.Unmarshal([]byte(metadataJSON), &av.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &av, nil
}

func (am *ArtifactManager) ListArtifacts(ctx context.Context, projectID string, artifactType ArtifactType, limit int) ([]Artifact, error) {
	var query string
	var args []interface{}

	if projectID != "" && artifactType != "" {
		query = `SELECT id, project_id, session_id, type, name, description, latest_version, created_at, updated_at, metadata 
		         FROM artifacts WHERE project_id = ? AND type = ? ORDER BY updated_at DESC LIMIT ?`
		args = append(args, projectID, artifactType, limit)
	} else if projectID != "" {
		query = `SELECT id, project_id, session_id, type, name, description, latest_version, created_at, updated_at, metadata 
		         FROM artifacts WHERE project_id = ? ORDER BY updated_at DESC LIMIT ?`
		args = append(args, projectID, limit)
	} else if artifactType != "" {
		query = `SELECT id, project_id, session_id, type, name, description, latest_version, created_at, updated_at, metadata 
		         FROM artifacts WHERE type = ? ORDER BY updated_at DESC LIMIT ?`
		args = append(args, artifactType, limit)
	} else {
		query = `SELECT id, project_id, session_id, type, name, description, latest_version, created_at, updated_at, metadata 
		         FROM artifacts ORDER BY updated_at DESC LIMIT ?`
		args = append(args, limit)
	}

	rows, err := am.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}
	defer rows.Close()

	artifacts := []Artifact{}
	for rows.Next() {
		var artifact Artifact
		var metadataJSON string
		var sessionID sql.NullString
		var description sql.NullString

		err := rows.Scan(
			&artifact.ID,
			&artifact.ProjectID,
			&sessionID,
			&artifact.Type,
			&artifact.Name,
			&description,
			&artifact.LatestVersion,
			&artifact.CreatedAt,
			&artifact.UpdatedAt,
			&metadataJSON,
		)

		if err != nil {
			slog.Warn("failed to scan artifact row", "error", err)
			continue
		}

		if sessionID.Valid {
			artifact.SessionID = sessionID.String
		}

		if description.Valid {
			artifact.Description = description.String
		}

		if err := json.Unmarshal([]byte(metadataJSON), &artifact.Metadata); err != nil {
			slog.Warn("failed to unmarshal artifact metadata", "error", err)
			continue
		}

		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

func (am *ArtifactManager) ListVersions(ctx context.Context, artifactID string) ([]ArtifactVersion, error) {
	query := `SELECT id, artifact_id, version, content, diff, commit_message, created_at, created_by, metadata 
	          FROM artifact_versions WHERE artifact_id = ? ORDER BY version DESC`

	rows, err := am.db.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	versions := []ArtifactVersion{}
	for rows.Next() {
		var av ArtifactVersion
		var metadataJSON string
		var diff sql.NullString
		var commitMessage sql.NullString
		var createdBy sql.NullString

		err := rows.Scan(
			&av.ID,
			&av.ArtifactID,
			&av.Version,
			&av.Content,
			&diff,
			&commitMessage,
			&av.CreatedAt,
			&createdBy,
			&metadataJSON,
		)

		if err != nil {
			slog.Warn("failed to scan artifact version row", "error", err)
			continue
		}

		if diff.Valid {
			av.Diff = diff.String
		}

		if commitMessage.Valid {
			av.CommitMessage = commitMessage.String
		}

		if createdBy.Valid {
			av.CreatedBy = createdBy.String
		}

		if err := json.Unmarshal([]byte(metadataJSON), &av.Metadata); err != nil {
			slog.Warn("failed to unmarshal artifact version metadata", "error", err)
			continue
		}

		versions = append(versions, av)
	}

	return versions, nil
}

func (am *ArtifactManager) DeleteArtifact(ctx context.Context, id string) error {
	tx, err := am.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	deleteVersionsQuery := `DELETE FROM artifact_versions WHERE artifact_id = ?`
	_, err = tx.ExecContext(ctx, deleteVersionsQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete artifact versions: %w", err)
	}

	deleteArtifactQuery := `DELETE FROM artifacts WHERE id = ?`
	result, err := tx.ExecContext(ctx, deleteArtifactQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("artifact not found: %s", id)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (am *ArtifactManager) Close() error {
	return am.db.Close()
}
