package projects

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type ProjectManager struct {
	db             *sql.DB
	projectBaseDir string
	exporter       *Exporter
}

// NewProjectManager creates a new ProjectManager with the given database connection.
// Initialization includes schema creation and default template loading.
func NewProjectManager(db *sql.DB) (*ProjectManager, error) {
	return NewProjectManagerWithContext(context.Background(), db)
}

// NewProjectManagerWithContext creates a new ProjectManager with the given database connection and context.
// The context controls the initialization phase (schema and template creation).
func NewProjectManagerWithContext(ctx context.Context, db *sql.DB) (*ProjectManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	projectBaseDir := filepath.Join(homeDir, "DojoProjects")
	if err := os.MkdirAll(projectBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base project directory: %w", err)
	}

	pm := &ProjectManager{
		db:             db,
		projectBaseDir: projectBaseDir,
	}

	pm.exporter = NewExporter(db, pm)

	if err := pm.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	if err := pm.initDefaultTemplatesWithContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize default templates: %w", err)
	}

	return pm, nil
}

func (pm *ProjectManager) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		template_id TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		last_accessed_at DATETIME NOT NULL,
		settings TEXT,
		metadata TEXT,
		status TEXT DEFAULT 'active'
	);

	CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
	CREATE INDEX IF NOT EXISTS idx_projects_last_accessed ON projects(last_accessed_at DESC);
	CREATE INDEX IF NOT EXISTS idx_projects_updated ON projects(updated_at DESC);

	CREATE TABLE IF NOT EXISTS project_templates (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		category TEXT,
		structure TEXT NOT NULL,
		default_settings TEXT,
		is_system BOOLEAN DEFAULT FALSE,
		created_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_templates_category ON project_templates(category);

	CREATE TABLE IF NOT EXISTS project_files (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		filename TEXT NOT NULL,
		file_path TEXT NOT NULL,
		file_size INTEGER,
		mime_type TEXT,
		uploaded_at DATETIME NOT NULL,
		metadata TEXT,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_project_files_project ON project_files(project_id);
	CREATE INDEX IF NOT EXISTS idx_project_files_uploaded ON project_files(uploaded_at DESC);
	`

	_, err := pm.db.Exec(schema)
	return err
}

func (pm *ProjectManager) initDefaultTemplates() error {
	return pm.initDefaultTemplatesWithContext(context.Background())
}

func (pm *ProjectManager) initDefaultTemplatesWithContext(ctx context.Context) error {
	templates := []ProjectTemplate{
		{
			ID:          "research-report",
			Name:        "Research Report",
			Description: "Structured project for research and analysis with documentation",
			Category:    CategoryResearch,
			Structure: map[string]interface{}{
				"directories": []string{"research", "notes", "artifacts", "references"},
			},
			DefaultSettings: map[string]interface{}{
				"auto_save": true,
			},
			IsSystem:  true,
			CreatedAt: time.Now(),
		},
		{
			ID:          "software-design",
			Name:        "Software Design",
			Description: "Project template for software architecture and design work",
			Category:    CategoryDevelopment,
			Structure: map[string]interface{}{
				"directories": []string{"diagrams", "specs", "code", "docs"},
			},
			DefaultSettings: map[string]interface{}{
				"auto_save": true,
			},
			IsSystem:  true,
			CreatedAt: time.Now(),
		},
		{
			ID:          "data-analysis",
			Name:        "Data Analysis",
			Description: "Project for data analysis and visualization tasks",
			Category:    CategoryAnalysis,
			Structure: map[string]interface{}{
				"directories": []string{"data", "analysis", "visualizations", "reports"},
			},
			DefaultSettings: map[string]interface{}{
				"auto_save": true,
			},
			IsSystem:  true,
			CreatedAt: time.Now(),
		},
		{
			ID:          "creative-studio",
			Name:        "Creative Studio",
			Description: "General purpose creative workspace for ideation and prototyping",
			Category:    CategoryDesign,
			Structure: map[string]interface{}{
				"directories": []string{"drafts", "assets", "exports"},
			},
			DefaultSettings: map[string]interface{}{
				"auto_save": true,
			},
			IsSystem:  true,
			CreatedAt: time.Now(),
		},
	}

	for _, template := range templates {
		existing, err := pm.GetTemplate(ctx, template.ID)
		if err == nil && existing != nil {
			continue
		}

		if err := pm.createTemplate(ctx, &template); err != nil {
			return fmt.Errorf("failed to create template %s: %w", template.ID, err)
		}
	}

	return nil
}

func (pm *ProjectManager) createTemplate(ctx context.Context, template *ProjectTemplate) error {
	structureJSON, err := json.Marshal(template.Structure)
	if err != nil {
		return fmt.Errorf("failed to marshal structure: %w", err)
	}

	settingsJSON, err := json.Marshal(template.DefaultSettings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `
		INSERT INTO project_templates (id, name, description, category, structure, default_settings, is_system, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`

	_, err = pm.db.ExecContext(ctx, query,
		template.ID,
		template.Name,
		template.Description,
		template.Category,
		string(structureJSON),
		string(settingsJSON),
		template.IsSystem,
		template.CreatedAt,
	)

	return err
}

// GetTemplate retrieves a project template by ID.
func (pm *ProjectManager) GetTemplate(ctx context.Context, id string) (*ProjectTemplate, error) {
	query := `SELECT id, name, description, category, structure, default_settings, is_system, created_at 
	          FROM project_templates WHERE id = ?`

	var template ProjectTemplate
	var structureJSON, settingsJSON string

	err := pm.db.QueryRowContext(ctx, query, id).Scan(
		&template.ID,
		&template.Name,
		&template.Description,
		&template.Category,
		&structureJSON,
		&settingsJSON,
		&template.IsSystem,
		&template.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve template: %w", err)
	}

	if err := json.Unmarshal([]byte(structureJSON), &template.Structure); err != nil {
		return nil, fmt.Errorf("failed to unmarshal structure: %w", err)
	}

	if err := json.Unmarshal([]byte(settingsJSON), &template.DefaultSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	return &template, nil
}

// ListTemplates returns all project templates ordered by category and name.
func (pm *ProjectManager) ListTemplates(ctx context.Context) ([]ProjectTemplate, error) {
	query := `SELECT id, name, description, category, structure, default_settings, is_system, created_at 
	          FROM project_templates ORDER BY category, name`

	rows, err := pm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	templates := []ProjectTemplate{}
	for rows.Next() {
		var template ProjectTemplate
		var structureJSON, settingsJSON string

		err := rows.Scan(
			&template.ID,
			&template.Name,
			&template.Description,
			&template.Category,
			&structureJSON,
			&settingsJSON,
			&template.IsSystem,
			&template.CreatedAt,
		)

		if err != nil {
			log.Printf("warning: failed to scan template row: %v", err)
			continue
		}

		if err := json.Unmarshal([]byte(structureJSON), &template.Structure); err != nil {
			log.Printf("warning: failed to unmarshal template structure for %s: %v", template.ID, err)
			continue
		}

		if err := json.Unmarshal([]byte(settingsJSON), &template.DefaultSettings); err != nil {
			log.Printf("warning: failed to unmarshal template settings for %s: %v", template.ID, err)
			continue
		}

		templates = append(templates, template)
	}

	return templates, nil
}

func validateProjectName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	if len(name) > 255 {
		return fmt.Errorf("project name too long (maximum 255 characters)")
	}
	if strings.ContainsAny(name, `/\:*?"<>|`) {
		return fmt.Errorf("project name contains invalid characters")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("project name cannot contain '..'")
	}
	return nil
}

// CreateProject creates a new project with the given name, description, and optional template.
// The project name is validated and must not contain path traversal characters.
// A directory is created at ~/DojoProjects/{name}/.
func (pm *ProjectManager) CreateProject(ctx context.Context, name, description, templateID string) (*Project, error) {
	name = strings.TrimSpace(name)
	if err := validateProjectName(name); err != nil {
		return nil, err
	}

	var template *ProjectTemplate
	var err error
	if templateID != "" {
		template, err = pm.GetTemplate(ctx, templateID)
		if err != nil {
			return nil, fmt.Errorf("failed to get template: %w", err)
		}
		if template == nil {
			return nil, fmt.Errorf("template not found: %s", templateID)
		}
	}

	now := time.Now()
	project := &Project{
		ID:             uuid.New().String(),
		Name:           name,
		Description:    description,
		TemplateID:     templateID,
		CreatedAt:      now,
		UpdatedAt:      now,
		LastAccessedAt: now,
		Settings:       make(map[string]interface{}),
		Metadata:       make(map[string]interface{}),
		Status:         StatusActive,
	}

	if template != nil && template.DefaultSettings != nil {
		project.Settings = template.DefaultSettings
	}

	projectDir := filepath.Join(pm.projectBaseDir, name)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}

	if template != nil {
		if dirs, ok := template.Structure["directories"].([]interface{}); ok {
			for _, dir := range dirs {
				if dirName, ok := dir.(string); ok {
					dirPath := filepath.Join(projectDir, dirName)
					if err := os.MkdirAll(dirPath, 0755); err != nil {
						return nil, fmt.Errorf("failed to create directory %s: %w", dirName, err)
					}
				}
			}
		}
	}

	settingsJSON, err := json.Marshal(project.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	metadataJSON, err := json.Marshal(project.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO projects (id, name, description, template_id, created_at, updated_at, last_accessed_at, settings, metadata, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = pm.db.ExecContext(ctx, query,
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

	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

// GetProject retrieves a project by ID.
// Note: This method updates the project's last_accessed_at timestamp as a side effect.
func (pm *ProjectManager) GetProject(ctx context.Context, id string) (*Project, error) {
	query := `SELECT id, name, description, template_id, created_at, updated_at, last_accessed_at, settings, metadata, status 
	          FROM projects WHERE id = ?`

	var project Project
	var settingsJSON, metadataJSON string
	var templateID sql.NullString

	err := pm.db.QueryRowContext(ctx, query, id).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&templateID,
		&project.CreatedAt,
		&project.UpdatedAt,
		&project.LastAccessedAt,
		&settingsJSON,
		&metadataJSON,
		&project.Status,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found: %s", id)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve project: %w", err)
	}

	if templateID.Valid {
		project.TemplateID = templateID.String
	}

	if err := json.Unmarshal([]byte(settingsJSON), &project.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	if err := json.Unmarshal([]byte(metadataJSON), &project.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	updateQuery := `UPDATE projects SET last_accessed_at = ? WHERE id = ?`
	if _, err := pm.db.ExecContext(ctx, updateQuery, time.Now(), id); err != nil {
		log.Printf("warning: failed to update last_accessed_at for project %s: %v", id, err)
	}

	return &project, nil
}

// ListProjects returns all projects, optionally filtered by status.
// Projects are ordered by last accessed time (most recent first).
func (pm *ProjectManager) ListProjects(ctx context.Context, status string) ([]Project, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `SELECT id, name, description, template_id, created_at, updated_at, last_accessed_at, settings, metadata, status 
		         FROM projects WHERE status = ? ORDER BY last_accessed_at DESC`
		args = append(args, status)
	} else {
		query = `SELECT id, name, description, template_id, created_at, updated_at, last_accessed_at, settings, metadata, status 
		         FROM projects ORDER BY last_accessed_at DESC`
	}

	rows, err := pm.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	projects := []Project{}
	for rows.Next() {
		var project Project
		var settingsJSON, metadataJSON string
		var templateID sql.NullString

		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Description,
			&templateID,
			&project.CreatedAt,
			&project.UpdatedAt,
			&project.LastAccessedAt,
			&settingsJSON,
			&metadataJSON,
			&project.Status,
		)

		if err != nil {
			log.Printf("warning: failed to scan project row: %v", err)
			continue
		}

		if templateID.Valid {
			project.TemplateID = templateID.String
		}

		if err := json.Unmarshal([]byte(settingsJSON), &project.Settings); err != nil {
			log.Printf("warning: failed to unmarshal project settings for %s: %v", project.ID, err)
			continue
		}

		if err := json.Unmarshal([]byte(metadataJSON), &project.Metadata); err != nil {
			log.Printf("warning: failed to unmarshal project metadata for %s: %v", project.ID, err)
			continue
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// UpdateProject updates an existing project's properties.
// Note: Project names cannot be changed after creation to maintain file system consistency.
func (pm *ProjectManager) UpdateProject(ctx context.Context, id string, name, description, status string, settings, metadata map[string]interface{}) (*Project, error) {
	existing, err := pm.GetProject(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != "" && name != existing.Name {
		return nil, fmt.Errorf("project name cannot be changed after creation")
	}

	if description != "" {
		existing.Description = description
	}

	if status != "" {
		existing.Status = status
	}

	if settings != nil {
		existing.Settings = settings
	}

	if metadata != nil {
		existing.Metadata = metadata
	}

	existing.UpdatedAt = time.Now()

	settingsJSON, err := json.Marshal(existing.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	metadataJSON, err := json.Marshal(existing.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `UPDATE projects SET name = ?, description = ?, status = ?, settings = ?, metadata = ?, updated_at = ? WHERE id = ?`
	_, err = pm.db.ExecContext(ctx, query, existing.Name, existing.Description, existing.Status, string(settingsJSON), string(metadataJSON), existing.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return existing, nil
}

// DeleteProject deletes a project from the database and removes its file system directory.
func (pm *ProjectManager) DeleteProject(ctx context.Context, id string) error {
	project, err := pm.GetProject(ctx, id)
	if err != nil {
		return err
	}

	query := `DELETE FROM projects WHERE id = ?`
	result, err := pm.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project not found: %s", id)
	}

	projectDir := filepath.Join(pm.projectBaseDir, project.Name)
	if err := os.RemoveAll(projectDir); err != nil {
		return fmt.Errorf("failed to delete project directory: %w", err)
	}

	return nil
}

// GetProjectDirectory returns the file system path for a project's directory.
func (pm *ProjectManager) GetProjectDirectory(projectName string) string {
	return filepath.Join(pm.projectBaseDir, projectName)
}

// Close is a no-op method that exists for interface compliance.
// ProjectManager does not own the database connection and does not require cleanup.
func (pm *ProjectManager) Close() error {
	return nil
}

// ExportProject exports a project and its artifacts to a DojoPacket ZIP file.
func (pm *ProjectManager) ExportProject(ctx context.Context, projectID string) ([]byte, error) {
	return pm.exporter.ExportProject(ctx, projectID)
}

// ImportProject imports a project from a DojoPacket ZIP file.
func (pm *ProjectManager) ImportProject(ctx context.Context, zipData []byte) (*Project, error) {
	return pm.exporter.ImportProject(ctx, zipData)
}
