package projects

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type DojoPacket struct {
	Version    string                 `json:"dojo_packet_version"`
	ExportedAt time.Time              `json:"exported_at"`
	Project    *ProjectExport         `json:"project"`
	Artifacts  []ArtifactExport       `json:"artifacts"`
	Memory     *MemoryExport          `json:"memory,omitempty"`
	Files      []FileExport           `json:"files"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type ProjectExport struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	TemplateID  string                 `json:"template_id,omitempty"`
	Settings    map[string]interface{} `json:"settings"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

type ArtifactExport struct {
	ID          string                  `json:"id"`
	Type        string                  `json:"type"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	Versions    []ArtifactVersionExport `json:"versions"`
	Metadata    map[string]interface{}  `json:"metadata"`
}

type ArtifactVersionExport struct {
	Version       int                    `json:"version"`
	Content       string                 `json:"content"`
	Diff          string                 `json:"diff,omitempty"`
	CommitMessage string                 `json:"commit_message,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	CreatedBy     string                 `json:"created_by"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type MemoryExport struct {
	Seeds []MemorySeedExport `json:"seeds"`
}

type MemorySeedExport struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Content   string                 `json:"content"`
	Tier      int                    `json:"tier"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type FileExport struct {
	Path     string `json:"path"`
	MimeType string `json:"mime_type,omitempty"`
	Data     string `json:"data"`
}

type Exporter struct {
	db             *sql.DB
	projectManager *ProjectManager
	projectBaseDir string
}

func NewExporter(db *sql.DB, pm *ProjectManager) *Exporter {
	return &Exporter{
		db:             db,
		projectManager: pm,
		projectBaseDir: pm.projectBaseDir,
	}
}

func (e *Exporter) ExportProject(ctx context.Context, projectID string) ([]byte, error) {
	project, err := e.projectManager.GetProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	artifacts, err := e.exportArtifacts(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to export artifacts: %w", err)
	}

	files, err := e.exportFiles(ctx, projectID, project.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to export files: %w", err)
	}

	memory, err := e.exportMemory(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to export memory: %w", err)
	}

	totalSize := int64(0)
	for _, f := range files {
		decoded, _ := base64.StdEncoding.DecodeString(f.Data)
		totalSize += int64(len(decoded))
	}

	packet := DojoPacket{
		Version:    "1.0",
		ExportedAt: time.Now(),
		Project: &ProjectExport{
			ID:          project.ID,
			Name:        project.Name,
			Description: project.Description,
			TemplateID:  project.TemplateID,
			Settings:    project.Settings,
			Metadata:    project.Metadata,
			CreatedAt:   project.CreatedAt,
		},
		Artifacts: artifacts,
		Memory:    memory,
		Files:     files,
		Metadata: map[string]interface{}{
			"total_artifacts":  len(artifacts),
			"total_files":      len(files),
			"total_size_bytes": totalSize,
		},
	}

	zipData, err := e.createZipArchive(&packet)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip archive: %w", err)
	}

	return zipData, nil
}

func (e *Exporter) exportArtifacts(ctx context.Context, projectID string) ([]ArtifactExport, error) {
	query := `SELECT id, type, name, description, metadata FROM artifacts WHERE project_id = ?`
	rows, err := e.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []ArtifactExport
	for rows.Next() {
		var artifact ArtifactExport
		var metadataJSON string
		var description sql.NullString

		err := rows.Scan(&artifact.ID, &artifact.Type, &artifact.Name, &description, &metadataJSON)
		if err != nil {
			slog.Warn("failed to scan artifact during export", "error", err)
			continue
		}

		if description.Valid {
			artifact.Description = description.String
		}

		if err := json.Unmarshal([]byte(metadataJSON), &artifact.Metadata); err != nil {
			artifact.Metadata = make(map[string]interface{})
		}

		versions, err := e.exportArtifactVersions(ctx, artifact.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to export versions for artifact %s: %w", artifact.ID, err)
		}
		artifact.Versions = versions

		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

func (e *Exporter) exportArtifactVersions(ctx context.Context, artifactID string) ([]ArtifactVersionExport, error) {
	query := `SELECT version, content, diff, commit_message, created_at, created_by, metadata 
	          FROM artifact_versions WHERE artifact_id = ? ORDER BY version ASC`
	rows, err := e.db.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to query versions: %w", err)
	}
	defer rows.Close()

	var versions []ArtifactVersionExport
	for rows.Next() {
		var version ArtifactVersionExport
		var diff, commitMessage, createdBy sql.NullString
		var metadataJSON string

		err := rows.Scan(&version.Version, &version.Content, &diff, &commitMessage,
			&version.CreatedAt, &createdBy, &metadataJSON)
		if err != nil {
			slog.Warn("failed to scan artifact version during export", "error", err)
			continue
		}

		if diff.Valid {
			version.Diff = diff.String
		}
		if commitMessage.Valid {
			version.CommitMessage = commitMessage.String
		}
		if createdBy.Valid {
			version.CreatedBy = createdBy.String
		}

		if err := json.Unmarshal([]byte(metadataJSON), &version.Metadata); err != nil {
			version.Metadata = make(map[string]interface{})
		}

		versions = append(versions, version)
	}

	return versions, nil
}

func (e *Exporter) exportFiles(ctx context.Context, projectID, projectName string) ([]FileExport, error) {
	projectDir := filepath.Join(e.projectBaseDir, projectName)
	var files []FileExport

	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return files, nil
	}

	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(projectDir, path)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		encodedData := base64.StdEncoding.EncodeToString(data)
		files = append(files, FileExport{
			Path: relPath,
			Data: encodedData,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk project directory: %w", err)
	}

	return files, nil
}

func (e *Exporter) exportMemory(ctx context.Context, projectID string) (*MemoryExport, error) {
	if !e.tableExists(ctx, "memory_seeds") {
		return nil, nil
	}

	query := `SELECT id, type, content, tier, created_at, metadata 
	          FROM memory_seeds WHERE project_id = ?`
	rows, err := e.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	var seeds []MemorySeedExport
	for rows.Next() {
		var seed MemorySeedExport
		var metadataJSON string

		err := rows.Scan(&seed.ID, &seed.Type, &seed.Content, &seed.Tier,
			&seed.CreatedAt, &metadataJSON)
		if err != nil {
			continue
		}

		if err := json.Unmarshal([]byte(metadataJSON), &seed.Metadata); err != nil {
			seed.Metadata = make(map[string]interface{})
		}

		seeds = append(seeds, seed)
	}

	if len(seeds) == 0 {
		return nil, nil
	}

	return &MemoryExport{Seeds: seeds}, nil
}

func (e *Exporter) tableExists(ctx context.Context, tableName string) bool {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?`
	var name string
	err := e.db.QueryRowContext(ctx, query, tableName).Scan(&name)
	return err == nil
}

func (e *Exporter) createZipArchive(packet *DojoPacket) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	manifestJSON, err := json.MarshalIndent(packet, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestWriter, err := zipWriter.Create("dojo_packet_manifest.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest in zip: %w", err)
	}

	if _, err := manifestWriter.Write(manifestJSON); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	for _, file := range packet.Files {
		fileData, err := base64.StdEncoding.DecodeString(file.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode file %s: %w", file.Path, err)
		}

		filePath := filepath.Join("files", file.Path)
		fileWriter, err := zipWriter.Create(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file %s in zip: %w", filePath, err)
		}

		if _, err := fileWriter.Write(fileData); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}

	return buf.Bytes(), nil
}

func (e *Exporter) ImportProject(ctx context.Context, zipData []byte) (*Project, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip archive: %w", err)
	}

	var manifest DojoPacket
	for _, file := range zipReader.File {
		if file.Name == "dojo_packet_manifest.json" {
			manifestFile, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open manifest: %w", err)
			}
			defer manifestFile.Close()

			manifestData, err := io.ReadAll(manifestFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read manifest: %w", err)
			}

			if err := json.Unmarshal(manifestData, &manifest); err != nil {
				return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
			}
			break
		}
	}

	if manifest.Version != "1.0" {
		return nil, fmt.Errorf("unsupported DojoPacket version: %s", manifest.Version)
	}

	if manifest.Project == nil {
		return nil, fmt.Errorf("invalid DojoPacket: missing project data")
	}

	if err := e.validateImport(ctx, &manifest); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	newProjectID := uuid.New().String()
	now := time.Now()

	project := &Project{
		ID:             newProjectID,
		Name:           manifest.Project.Name,
		Description:    manifest.Project.Description,
		TemplateID:     manifest.Project.TemplateID,
		Settings:       manifest.Project.Settings,
		Metadata:       manifest.Project.Metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
		LastAccessedAt: now,
		Status:         StatusActive,
	}

	projectDir := filepath.Join(e.projectBaseDir, project.Name)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		os.RemoveAll(projectDir)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
			os.RemoveAll(projectDir)
		}
	}()

	if err := e.importProjectData(ctx, tx, project); err != nil {
		return nil, fmt.Errorf("failed to import project: %w", err)
	}

	if err := e.importArtifacts(ctx, tx, newProjectID, manifest.Artifacts); err != nil {
		return nil, fmt.Errorf("failed to import artifacts: %w", err)
	}

	if manifest.Memory != nil {
		if err := e.importMemory(ctx, tx, newProjectID, manifest.Memory); err != nil {
			return nil, fmt.Errorf("failed to import memory: %w", err)
		}
	}

	if err := e.importFiles(ctx, zipReader, projectDir, manifest.Files); err != nil {
		return nil, fmt.Errorf("failed to import files: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	committed = true
	return project, nil
}

func (e *Exporter) validateImport(ctx context.Context, manifest *DojoPacket) error {
	query := `SELECT COUNT(*) FROM projects WHERE name = ?`
	var count int
	err := e.db.QueryRowContext(ctx, query, manifest.Project.Name).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for existing project: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("project with name '%s' already exists", manifest.Project.Name)
	}

	return nil
}

func (e *Exporter) importProjectData(ctx context.Context, tx *sql.Tx, project *Project) error {
	settingsJSON, err := json.Marshal(project.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	metadataJSON, err := json.Marshal(project.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO projects (id, name, description, template_id, created_at, updated_at, last_accessed_at, settings, metadata, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.ExecContext(ctx, query,
		project.ID,
		project.Name,
		project.Description,
		project.TemplateID,
		project.CreatedAt,
		project.UpdatedAt,
		project.LastAccessedAt,
		string(settingsJSON),
		string(metadataJSON),
		project.Status,
	)

	return err
}

func (e *Exporter) importArtifacts(ctx context.Context, tx *sql.Tx, projectID string, artifacts []ArtifactExport) error {
	for _, artifact := range artifacts {
		newArtifactID := uuid.New().String()

		metadataJSON, err := json.Marshal(artifact.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal artifact metadata: %w", err)
		}

		latestVersion := len(artifact.Versions)
		if latestVersion == 0 {
			return fmt.Errorf("artifact %q has no versions", artifact.Name)
		}

		now := time.Now()
		artifactQuery := `
			INSERT INTO artifacts (id, project_id, type, name, description, latest_version, created_at, updated_at, metadata)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		_, err = tx.ExecContext(ctx, artifactQuery,
			newArtifactID,
			projectID,
			artifact.Type,
			artifact.Name,
			artifact.Description,
			latestVersion,
			now,
			now,
			string(metadataJSON),
		)

		if err != nil {
			return fmt.Errorf("failed to insert artifact: %w", err)
		}

		for _, version := range artifact.Versions {
			versionMetadataJSON, err := json.Marshal(version.Metadata)
			if err != nil {
				return fmt.Errorf("failed to marshal version metadata: %w", err)
			}

			versionQuery := `
				INSERT INTO artifact_versions (id, artifact_id, version, content, diff, commit_message, created_at, created_by, metadata)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`

			_, err = tx.ExecContext(ctx, versionQuery,
				uuid.New().String(),
				newArtifactID,
				version.Version,
				version.Content,
				version.Diff,
				version.CommitMessage,
				version.CreatedAt,
				version.CreatedBy,
				string(versionMetadataJSON),
			)

			if err != nil {
				return fmt.Errorf("failed to insert artifact version: %w", err)
			}
		}
	}

	return nil
}

func (e *Exporter) importMemory(ctx context.Context, tx *sql.Tx, projectID string, memory *MemoryExport) error {
	if !e.tableExists(ctx, "memory_seeds") {
		return nil
	}

	for _, seed := range memory.Seeds {
		metadataJSON, err := json.Marshal(seed.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal seed metadata: %w", err)
		}

		query := `
			INSERT INTO memory_seeds (id, project_id, type, content, tier, created_at, updated_at, last_accessed_at, metadata)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		now := time.Now()
		_, err = tx.ExecContext(ctx, query,
			uuid.New().String(),
			projectID,
			seed.Type,
			seed.Content,
			seed.Tier,
			seed.CreatedAt,
			now,
			now,
			string(metadataJSON),
		)

		if err != nil {
			return fmt.Errorf("failed to insert memory seed: %w", err)
		}
	}

	return nil
}

func (e *Exporter) importFiles(ctx context.Context, zipReader *zip.Reader, projectDir string, fileManifest []FileExport) error {
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "dojo_packet_manifest.json" {
			continue
		}

		if len(zipFile.Name) < 6 || zipFile.Name[:6] != "files/" {
			continue
		}

		relPath := zipFile.Name[6:]

		cleanPath := filepath.Clean(relPath)
		if strings.Contains(cleanPath, "..") || filepath.IsAbs(cleanPath) {
			return fmt.Errorf("invalid file path in zip: %s", zipFile.Name)
		}

		targetPath := filepath.Join(projectDir, cleanPath)
		if !strings.HasPrefix(targetPath, projectDir) {
			return fmt.Errorf("path traversal attempt detected: %s", zipFile.Name)
		}

		fileReader, err := zipFile.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s in zip: %w", zipFile.Name, err)
		}

		fileData, err := io.ReadAll(fileReader)
		fileReader.Close()

		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", zipFile.Name, err)
		}

		targetDir := filepath.Dir(targetPath)

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
		}

		if err := os.WriteFile(targetPath, fileData, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}
	}

	return nil
}
