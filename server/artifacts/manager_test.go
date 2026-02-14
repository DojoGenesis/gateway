package artifacts

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS artifacts (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		session_id TEXT,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		latest_version INTEGER DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		metadata TEXT,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS artifact_versions (
		id TEXT PRIMARY KEY,
		artifact_id TEXT NOT NULL,
		version INTEGER NOT NULL,
		content TEXT NOT NULL,
		diff TEXT,
		commit_message TEXT,
		created_at DATETIME NOT NULL,
		created_by TEXT,
		metadata TEXT,
		FOREIGN KEY (artifact_id) REFERENCES artifacts(id) ON DELETE CASCADE,
		UNIQUE(artifact_id, version)
	);

	CREATE INDEX IF NOT EXISTS idx_artifacts_project ON artifacts(project_id);
	CREATE INDEX IF NOT EXISTS idx_artifacts_type ON artifacts(type);
	CREATE INDEX IF NOT EXISTS idx_artifacts_updated ON artifacts(updated_at DESC);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	return db
}

func TestNewArtifactManager(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	am, err := NewArtifactManager(db)
	assert.NoError(t, err)
	assert.NotNil(t, am)
}

func TestCreateArtifact(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	am, err := NewArtifactManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name         string
		projectID    string
		sessionID    string
		artifactType ArtifactType
		artifactName string
		content      string
		wantErr      bool
	}{
		{
			name:         "valid artifact creation",
			projectID:    "project-1",
			sessionID:    "session-1",
			artifactType: TypeDocument,
			artifactName: "Test Document",
			content:      "# Test Content",
			wantErr:      false,
		},
		{
			name:         "missing project_id",
			projectID:    "",
			sessionID:    "session-1",
			artifactType: TypeDocument,
			artifactName: "Test Document",
			content:      "# Test Content",
			wantErr:      true,
		},
		{
			name:         "missing name",
			projectID:    "project-1",
			sessionID:    "session-1",
			artifactType: TypeDocument,
			artifactName: "",
			content:      "# Test Content",
			wantErr:      true,
		},
		{
			name:         "missing type",
			projectID:    "project-1",
			sessionID:    "session-1",
			artifactType: "",
			artifactName: "Test Document",
			content:      "# Test Content",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact, err := am.CreateArtifact(ctx, tt.projectID, tt.sessionID, tt.artifactType, tt.artifactName, "", tt.content)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, artifact)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, artifact)
				assert.Equal(t, tt.projectID, artifact.ProjectID)
				assert.Equal(t, tt.sessionID, artifact.SessionID)
				assert.Equal(t, tt.artifactType, artifact.Type)
				assert.Equal(t, tt.artifactName, artifact.Name)
				assert.Equal(t, 1, artifact.LatestVersion)
				assert.NotEmpty(t, artifact.ID)
				assert.False(t, artifact.CreatedAt.IsZero())
				assert.False(t, artifact.UpdatedAt.IsZero())
			}
		})
	}
}

func TestUpdateArtifact(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	am, err := NewArtifactManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	artifact, err := am.CreateArtifact(ctx, "project-1", "session-1", TypeDocument, "Test Doc", "", "Version 1 content")
	require.NoError(t, err)
	require.Equal(t, 1, artifact.LatestVersion)

	time.Sleep(10 * time.Millisecond)

	updatedArtifact, err := am.UpdateArtifact(ctx, artifact.ID, "Version 2 content", "Updated to version 2")
	assert.NoError(t, err)
	assert.Equal(t, 2, updatedArtifact.LatestVersion)
	assert.True(t, updatedArtifact.UpdatedAt.After(artifact.UpdatedAt))

	version1, err := am.GetArtifactVersion(ctx, artifact.ID, 1)
	assert.NoError(t, err)
	assert.Equal(t, "Version 1 content", version1.Content)
	assert.Equal(t, 1, version1.Version)

	version2, err := am.GetArtifactVersion(ctx, artifact.ID, 2)
	assert.NoError(t, err)
	assert.Equal(t, "Version 2 content", version2.Content)
	assert.Equal(t, 2, version2.Version)
	assert.NotEmpty(t, version2.Diff)

	var diffResults []DiffResult
	err = json.Unmarshal([]byte(version2.Diff), &diffResults)
	assert.NoError(t, err)
	assert.NotEmpty(t, diffResults)
}

func TestGetArtifact(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	am, err := NewArtifactManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	created, err := am.CreateArtifact(ctx, "project-1", "session-1", TypeDiagram, "Test Diagram", "", "diagram content")
	require.NoError(t, err)

	retrieved, err := am.GetArtifact(ctx, created.ID)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.ProjectID, retrieved.ProjectID)
	assert.Equal(t, created.Type, retrieved.Type)
	assert.Equal(t, created.Name, retrieved.Name)

	_, err = am.GetArtifact(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "artifact not found")
}

func TestListArtifacts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	am, err := NewArtifactManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	_, err = am.CreateArtifact(ctx, "project-1", "session-1", TypeDocument, "Doc 1", "", "content 1")
	require.NoError(t, err)

	_, err = am.CreateArtifact(ctx, "project-1", "session-2", TypeDiagram, "Diagram 1", "", "diagram 1")
	require.NoError(t, err)

	_, err = am.CreateArtifact(ctx, "project-2", "session-3", TypeDocument, "Doc 2", "", "content 2")
	require.NoError(t, err)

	t.Run("list all artifacts", func(t *testing.T) {
		artifacts, err := am.ListArtifacts(ctx, "", "", 100)
		assert.NoError(t, err)
		assert.Len(t, artifacts, 3)
	})

	t.Run("list by project_id", func(t *testing.T) {
		artifacts, err := am.ListArtifacts(ctx, "project-1", "", 100)
		assert.NoError(t, err)
		assert.Len(t, artifacts, 2)
		for _, a := range artifacts {
			assert.Equal(t, "project-1", a.ProjectID)
		}
	})

	t.Run("list by type", func(t *testing.T) {
		artifacts, err := am.ListArtifacts(ctx, "", TypeDocument, 100)
		assert.NoError(t, err)
		assert.Len(t, artifacts, 2)
		for _, a := range artifacts {
			assert.Equal(t, TypeDocument, a.Type)
		}
	})

	t.Run("list by project_id and type", func(t *testing.T) {
		artifacts, err := am.ListArtifacts(ctx, "project-1", TypeDocument, 100)
		assert.NoError(t, err)
		assert.Len(t, artifacts, 1)
		assert.Equal(t, "project-1", artifacts[0].ProjectID)
		assert.Equal(t, TypeDocument, artifacts[0].Type)
	})

	t.Run("list with limit", func(t *testing.T) {
		artifacts, err := am.ListArtifacts(ctx, "", "", 2)
		assert.NoError(t, err)
		assert.Len(t, artifacts, 2)
	})
}

func TestListVersions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	am, err := NewArtifactManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	artifact, err := am.CreateArtifact(ctx, "project-1", "session-1", TypeDocument, "Test Doc", "", "v1")
	require.NoError(t, err)

	_, err = am.UpdateArtifact(ctx, artifact.ID, "v2", "Update to v2")
	require.NoError(t, err)

	_, err = am.UpdateArtifact(ctx, artifact.ID, "v3", "Update to v3")
	require.NoError(t, err)

	versions, err := am.ListVersions(ctx, artifact.ID)
	assert.NoError(t, err)
	assert.Len(t, versions, 3)

	assert.Equal(t, 3, versions[0].Version)
	assert.Equal(t, 2, versions[1].Version)
	assert.Equal(t, 1, versions[2].Version)

	assert.Equal(t, "v3", versions[0].Content)
	assert.Equal(t, "v2", versions[1].Content)
	assert.Equal(t, "v1", versions[2].Content)
}

func TestDeleteArtifact(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	am, err := NewArtifactManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	artifact, err := am.CreateArtifact(ctx, "project-1", "session-1", TypeDocument, "Test Doc", "", "content")
	require.NoError(t, err)

	_, err = am.UpdateArtifact(ctx, artifact.ID, "updated content", "Update")
	require.NoError(t, err)

	err = am.DeleteArtifact(ctx, artifact.ID)
	assert.NoError(t, err)

	_, err = am.GetArtifact(ctx, artifact.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "artifact not found")

	versions, err := am.ListVersions(ctx, artifact.ID)
	assert.NoError(t, err)
	assert.Empty(t, versions)

	err = am.DeleteArtifact(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "artifact not found")
}

func TestArtifactVersionIncrement(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	am, err := NewArtifactManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	artifact, err := am.CreateArtifact(ctx, "project-1", "session-1", TypeDocument, "Test", "", "content v1")
	require.NoError(t, err)
	assert.Equal(t, 1, artifact.LatestVersion)

	for i := 2; i <= 5; i++ {
		updated, err := am.UpdateArtifact(ctx, artifact.ID, "content v"+string(rune(i)), "version update")
		require.NoError(t, err)
		assert.Equal(t, i, updated.LatestVersion)
	}

	retrieved, err := am.GetArtifact(ctx, artifact.ID)
	assert.NoError(t, err)
	assert.Equal(t, 5, retrieved.LatestVersion)
}

func TestCalculateDiff(t *testing.T) {
	tests := []struct {
		name       string
		oldContent string
		newContent string
		wantErr    bool
	}{
		{
			name:       "simple text change",
			oldContent: "Hello World",
			newContent: "Hello Go",
			wantErr:    false,
		},
		{
			name:       "multiline change",
			oldContent: "Line 1\nLine 2\nLine 3",
			newContent: "Line 1\nLine 2 modified\nLine 3",
			wantErr:    false,
		},
		{
			name:       "empty to content",
			oldContent: "",
			newContent: "New content",
			wantErr:    false,
		},
		{
			name:       "content to empty",
			oldContent: "Old content",
			newContent: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := CalculateDiff(tt.oldContent, tt.newContent)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, diff)

				var results []DiffResult
				err = json.Unmarshal([]byte(diff), &results)
				assert.NoError(t, err)
			}
		})
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
