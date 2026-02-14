package memory

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSeedTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", "file::memory:?mode=memory&cache=shared")
	require.NoError(t, err)

	schema := `
	PRAGMA foreign_keys = ON;

	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS memory_seeds (
		id TEXT PRIMARY KEY,
		project_id TEXT,
		content TEXT NOT NULL,
		seed_type TEXT NOT NULL,
		source TEXT DEFAULT 'system' CHECK(source IN ('system', 'user', 'calibrated')),
		user_editable BOOLEAN DEFAULT FALSE,
		confidence REAL DEFAULT 1.0 CHECK(confidence >= 0.0 AND confidence <= 1.0),
		usage_count INTEGER DEFAULT 0,
		last_used_at DATETIME,
		deleted_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		created_by TEXT,
		version INTEGER DEFAULT 1,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_memory_seeds_project ON memory_seeds(project_id);
	CREATE INDEX IF NOT EXISTS idx_memory_seeds_type ON memory_seeds(seed_type);
	CREATE INDEX IF NOT EXISTS idx_memory_seeds_source ON memory_seeds(source);
	CREATE INDEX IF NOT EXISTS idx_memory_seeds_editable ON memory_seeds(user_editable);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO projects (id, name) VALUES ('project-1', 'Test Project')")
	require.NoError(t, err)

	return db
}

func TestNewSeedManager(t *testing.T) {
	t.Run("with valid db", func(t *testing.T) {
		db := setupSeedTestDB(t)
		defer db.Close()

		sm, err := NewSeedManager(db)
		assert.NoError(t, err)
		assert.NotNil(t, sm)
	})

	t.Run("with nil db", func(t *testing.T) {
		sm, err := NewSeedManager(nil)
		assert.Error(t, err)
		assert.Nil(t, sm)
		assert.Contains(t, err.Error(), "database connection is required")
	})
}

func TestCreateUserSeed(t *testing.T) {
	db := setupSeedTestDB(t)
	defer db.Close()

	sm, err := NewSeedManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name      string
		projectID *string
		content   string
		seedType  string
		userID    string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid global seed",
			projectID: nil,
			content:   "User preference: dark mode",
			seedType:  "preference",
			userID:    "user-1",
			wantErr:   false,
		},
		{
			name:      "valid project seed",
			projectID: stringPtr("project-1"),
			content:   "Project-specific pattern",
			seedType:  "pattern",
			userID:    "user-1",
			wantErr:   false,
		},
		{
			name:      "missing content",
			projectID: nil,
			content:   "",
			seedType:  "preference",
			userID:    "user-1",
			wantErr:   true,
			errMsg:    "invalid content",
		},
		{
			name:      "missing seed_type",
			projectID: nil,
			content:   "Some content",
			seedType:  "",
			userID:    "user-1",
			wantErr:   true,
			errMsg:    "seed_type is required",
		},
		{
			name:      "missing user_id",
			projectID: nil,
			content:   "Some content",
			seedType:  "preference",
			userID:    "",
			wantErr:   true,
			errMsg:    "user_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seed, err := sm.CreateUserSeed(ctx, tt.projectID, tt.content, tt.seedType, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, seed)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, seed)
				assert.NotEmpty(t, seed.ID)
				assert.Equal(t, tt.content, seed.Content)
				assert.Equal(t, tt.seedType, seed.SeedType)
				assert.Equal(t, SourceUser, seed.Source)
				assert.True(t, seed.UserEditable)
				assert.Equal(t, 1.0, seed.Confidence)
				assert.Equal(t, 0, seed.UsageCount)
				assert.NotNil(t, seed.CreatedBy)
				assert.Equal(t, tt.userID, *seed.CreatedBy)

				if tt.projectID != nil {
					assert.NotNil(t, seed.ProjectID)
					assert.Equal(t, *tt.projectID, *seed.ProjectID)
				} else {
					assert.Nil(t, seed.ProjectID)
				}
			}
		})
	}
}

func TestGetSeedByID(t *testing.T) {
	db := setupSeedTestDB(t)
	defer db.Close()

	sm, err := NewSeedManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	seed, err := sm.CreateUserSeed(ctx, nil, "Test content", "preference", "user-1")
	require.NoError(t, err)

	t.Run("existing seed", func(t *testing.T) {
		retrieved, err := sm.GetSeedByID(ctx, seed.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, seed.ID, retrieved.ID)
		assert.Equal(t, seed.Content, retrieved.Content)
		assert.Equal(t, seed.SeedType, retrieved.SeedType)
	})

	t.Run("non-existent seed", func(t *testing.T) {
		retrieved, err := sm.GetSeedByID(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Contains(t, err.Error(), "seed not found")
	})

	t.Run("empty id", func(t *testing.T) {
		retrieved, err := sm.GetSeedByID(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Contains(t, err.Error(), "id is required")
	})
}

func TestGetSeeds(t *testing.T) {
	db := setupSeedTestDB(t)
	defer db.Close()

	sm, err := NewSeedManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	_, err = sm.CreateUserSeed(ctx, nil, "Global seed 1", "preference", "user-1")
	require.NoError(t, err)

	projectID := "project-1"
	_, err = sm.CreateUserSeed(ctx, &projectID, "Project seed 1", "pattern", "user-1")
	require.NoError(t, err)

	_, err = sm.CreateUserSeed(ctx, &projectID, "Project seed 2", "knowledge", "user-2")
	require.NoError(t, err)

	t.Run("all seeds", func(t *testing.T) {
		seeds, err := sm.GetSeeds(ctx, nil, nil)
		assert.NoError(t, err)
		assert.Len(t, seeds, 3)
	})

	t.Run("global seeds only", func(t *testing.T) {
		filters := map[string]interface{}{"global_only": true}
		seeds, err := sm.GetSeeds(ctx, nil, filters)
		assert.NoError(t, err)
		assert.Len(t, seeds, 1)
		assert.Nil(t, seeds[0].ProjectID)
	})

	t.Run("project seeds", func(t *testing.T) {
		seeds, err := sm.GetSeeds(ctx, &projectID, nil)
		assert.NoError(t, err)
		assert.Len(t, seeds, 2)
		for _, seed := range seeds {
			assert.NotNil(t, seed.ProjectID)
			assert.Equal(t, projectID, *seed.ProjectID)
		}
	})

	t.Run("filter by seed_type", func(t *testing.T) {
		filters := map[string]interface{}{"seed_type": "pattern"}
		seeds, err := sm.GetSeeds(ctx, nil, filters)
		assert.NoError(t, err)
		assert.Len(t, seeds, 1)
		assert.Equal(t, "pattern", seeds[0].SeedType)
	})

	t.Run("filter by source", func(t *testing.T) {
		filters := map[string]interface{}{"source": "user"}
		seeds, err := sm.GetSeeds(ctx, nil, filters)
		assert.NoError(t, err)
		assert.Len(t, seeds, 3)
		for _, seed := range seeds {
			assert.Equal(t, SourceUser, seed.Source)
		}
	})

	t.Run("filter by user_editable", func(t *testing.T) {
		filters := map[string]interface{}{"user_editable": true}
		seeds, err := sm.GetSeeds(ctx, nil, filters)
		assert.NoError(t, err)
		assert.Len(t, seeds, 3)
		for _, seed := range seeds {
			assert.True(t, seed.UserEditable)
		}
	})
}

func TestUpdateSeed(t *testing.T) {
	db := setupSeedTestDB(t)
	defer db.Close()

	sm, err := NewSeedManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	editableSeed, err := sm.CreateUserSeed(ctx, nil, "Original content", "preference", "user-1")
	require.NoError(t, err)

	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO memory_seeds (id, content, seed_type, source, user_editable, confidence, usage_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "non-editable-seed", "System seed", "system", "system", false, 1.0, 0, now, now)
	require.NoError(t, err)

	t.Run("update editable seed", func(t *testing.T) {
		newContent := "Updated content"
		userID := "user-1"
		updated, err := sm.UpdateSeed(ctx, editableSeed.ID, newContent, &userID)
		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, newContent, updated.Content)
		assert.Equal(t, SourceUser, updated.Source)
		assert.True(t, updated.UpdatedAt.After(editableSeed.UpdatedAt))
	})

	t.Run("update non-editable seed", func(t *testing.T) {
		userID := "user-1"
		updated, err := sm.UpdateSeed(ctx, "non-editable-seed", "New content", &userID)
		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "seed is not editable")
	})

	t.Run("update non-existent seed", func(t *testing.T) {
		userID := "user-1"
		updated, err := sm.UpdateSeed(ctx, "non-existent-id", "New content", &userID)
		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "seed not found")
	})

	t.Run("update with empty content", func(t *testing.T) {
		userID := "user-1"
		updated, err := sm.UpdateSeed(ctx, editableSeed.ID, "", &userID)
		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "invalid content")
	})

	t.Run("update with empty id", func(t *testing.T) {
		userID := "user-1"
		updated, err := sm.UpdateSeed(ctx, "", "New content", &userID)
		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "id is required")
	})
}

func TestDeleteSeed(t *testing.T) {
	db := setupSeedTestDB(t)
	defer db.Close()

	sm, err := NewSeedManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	user1ID := "user-1"
	user2ID := "user-2"

	seed1, err := sm.CreateUserSeed(ctx, nil, "User 1 seed", "preference", user1ID)
	require.NoError(t, err)

	seed2, err := sm.CreateUserSeed(ctx, nil, "User 2 seed", "preference", user2ID)
	require.NoError(t, err)

	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO memory_seeds (id, content, seed_type, source, user_editable, confidence, usage_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "system-seed", "System seed", "system", "system", false, 1.0, 0, now, now)
	require.NoError(t, err)

	t.Run("delete own user seed", func(t *testing.T) {
		err := sm.DeleteSeed(ctx, seed1.ID, &user1ID)
		assert.NoError(t, err)

		_, err = sm.GetSeedByID(ctx, seed1.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "seed not found")
	})

	t.Run("delete other user's seed", func(t *testing.T) {
		err := sm.DeleteSeed(ctx, seed2.ID, &user1ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
	})

	t.Run("delete system seed", func(t *testing.T) {
		err := sm.DeleteSeed(ctx, "system-seed", &user1ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete system seed")
	})

	t.Run("delete non-existent seed", func(t *testing.T) {
		err := sm.DeleteSeed(ctx, "non-existent-id", &user1ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "seed not found")
	})

	t.Run("delete with empty id", func(t *testing.T) {
		err := sm.DeleteSeed(ctx, "", &user1ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "id is required")
	})

	t.Run("delete without user check", func(t *testing.T) {
		seed3, err := sm.CreateUserSeed(ctx, nil, "User 2 another seed", "preference", user2ID)
		require.NoError(t, err)

		err = sm.DeleteSeed(ctx, seed3.ID, nil)
		assert.NoError(t, err)
	})
}

func TestIncrementUsage(t *testing.T) {
	db := setupSeedTestDB(t)
	defer db.Close()

	sm, err := NewSeedManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	seed, err := sm.CreateUserSeed(ctx, nil, "Test seed", "preference", "user-1")
	require.NoError(t, err)

	t.Run("increment usage", func(t *testing.T) {
		err := sm.IncrementUsage(ctx, seed.ID)
		assert.NoError(t, err)

		updated, err := sm.GetSeedByID(ctx, seed.ID)
		assert.NoError(t, err)
		assert.Equal(t, 1, updated.UsageCount)
		assert.NotNil(t, updated.LastUsedAt)
		assert.True(t, updated.LastUsedAt.After(seed.CreatedAt))
	})

	t.Run("increment usage multiple times", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			err := sm.IncrementUsage(ctx, seed.ID)
			assert.NoError(t, err)
		}

		updated, err := sm.GetSeedByID(ctx, seed.ID)
		assert.NoError(t, err)
		assert.Equal(t, 6, updated.UsageCount)
	})

	t.Run("increment non-existent seed", func(t *testing.T) {
		err := sm.IncrementUsage(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "seed not found")
	})

	t.Run("increment with empty id", func(t *testing.T) {
		err := sm.IncrementUsage(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "id is required")
	})
}

func TestConcurrentAccess(t *testing.T) {
	db := setupSeedTestDB(t)
	defer db.Close()

	sm, err := NewSeedManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	seed, err := sm.CreateUserSeed(ctx, nil, "Concurrent test seed", "preference", "user-1")
	require.NoError(t, err)

	t.Run("concurrent usage increments", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				err := sm.IncrementUsage(ctx, seed.ID)
				assert.NoError(t, err)
				done <- true
			}()
		}

		for i := 0; i < concurrency; i++ {
			<-done
		}

		updated, err := sm.GetSeedByID(ctx, seed.ID)
		assert.NoError(t, err)
		assert.Equal(t, concurrency, updated.UsageCount)
	})

	t.Run("concurrent reads", func(t *testing.T) {
		concurrency := 20
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				_, err := sm.GetSeedByID(ctx, seed.ID)
				assert.NoError(t, err)
				done <- true
			}()
		}

		for i := 0; i < concurrency; i++ {
			<-done
		}
	})
}

func TestEdgeCase_LongContent(t *testing.T) {
	db := setupSeedTestDB(t)
	defer db.Close()

	sm, err := NewSeedManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	longContent := make([]byte, 10000)
	for i := range longContent {
		longContent[i] = 'a'
	}

	seed, err := sm.CreateUserSeed(ctx, nil, string(longContent), "preference", "user-1")
	assert.NoError(t, err)
	assert.NotNil(t, seed)

	retrieved, err := sm.GetSeedByID(ctx, seed.ID)
	assert.NoError(t, err)
	assert.Equal(t, string(longContent), retrieved.Content)
}

func TestEdgeCase_SpecialCharacters(t *testing.T) {
	db := setupSeedTestDB(t)
	defer db.Close()

	sm, err := NewSeedManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	specialContent := "Test with special chars: 日本語, émojis 🎉, SQL: '; DROP TABLE;"
	seed, err := sm.CreateUserSeed(ctx, nil, specialContent, "preference", "user-1")
	assert.NoError(t, err)
	assert.NotNil(t, seed)

	retrieved, err := sm.GetSeedByID(ctx, seed.ID)
	assert.NoError(t, err)
	assert.Equal(t, specialContent, retrieved.Content)
}

func TestSearchSeeds(t *testing.T) {
	db := setupSeedTestDB(t)
	defer db.Close()

	sm, err := NewSeedManager(db)
	require.NoError(t, err)

	ctx := context.Background()

	projectID := "test-project"
	_, err = db.Exec(`INSERT INTO projects (id, name) VALUES (?, ?)`, projectID, "Test Project")
	require.NoError(t, err)

	_, err = sm.CreateUserSeed(ctx, &projectID, "This is a test with apple content", "preference", "user-1")
	require.NoError(t, err)
	_, err = sm.CreateUserSeed(ctx, &projectID, "This contains banana information", "preference", "user-1")
	require.NoError(t, err)
	_, err = sm.CreateUserSeed(ctx, &projectID, "Orange juice recipe with apple", "recipe", "user-1")
	require.NoError(t, err)
	_, err = sm.CreateUserSeed(ctx, nil, "Global seed with banana", "global", "user-1")
	require.NoError(t, err)

	t.Run("search with results", func(t *testing.T) {
		seeds, err := sm.SearchSeeds(ctx, &projectID, "apple", 10)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(seeds))
	})

	t.Run("search with no results", func(t *testing.T) {
		seeds, err := sm.SearchSeeds(ctx, &projectID, "nonexistent", 10)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(seeds))
	})

	t.Run("search with limit", func(t *testing.T) {
		seeds, err := sm.SearchSeeds(ctx, &projectID, "apple", 1)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(seeds), 1)
	})

	t.Run("search case insensitive", func(t *testing.T) {
		seeds, err := sm.SearchSeeds(ctx, &projectID, "APPLE", 10)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(seeds))
	})

	t.Run("search without project filter", func(t *testing.T) {
		seeds, err := sm.SearchSeeds(ctx, nil, "banana", 10)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(seeds), 2)
	})

	t.Run("search with empty query", func(t *testing.T) {
		seeds, err := sm.SearchSeeds(ctx, &projectID, "", 10)
		assert.Error(t, err)
		assert.Nil(t, seeds)
		assert.Contains(t, err.Error(), "search query is required")
	})

	t.Run("search with query too short", func(t *testing.T) {
		seeds, err := sm.SearchSeeds(ctx, &projectID, "a", 10)
		assert.Error(t, err)
		assert.Nil(t, seeds)
		assert.Contains(t, err.Error(), "at least 2 characters")
	})

	t.Run("search with default limit", func(t *testing.T) {
		seeds, err := sm.SearchSeeds(ctx, &projectID, "test", 0)
		assert.NoError(t, err)
		assert.NotNil(t, seeds)
	})

	t.Run("search with limit exceeding max", func(t *testing.T) {
		seeds, err := sm.SearchSeeds(ctx, &projectID, "test", 200)
		assert.NoError(t, err)
		assert.NotNil(t, seeds)
		assert.LessOrEqual(t, len(seeds), 100)
	})
}

func stringPtr(s string) *string {
	return &s
}
