package memory

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"
)

func setupSeedBenchDB(b *testing.B) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}

	schema := `
		CREATE TABLE memory_seeds (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			content TEXT NOT NULL,
			seed_type TEXT NOT NULL,
			source TEXT NOT NULL,
			user_editable BOOLEAN DEFAULT FALSE,
			confidence REAL DEFAULT 1.0,
			usage_count INTEGER DEFAULT 0,
			last_used_at DATETIME,
			deleted_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			created_by TEXT,
			version INTEGER DEFAULT 1
		);

		CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL
		);

		INSERT INTO projects (id, name, created_at) VALUES 
			('project1', 'Test Project 1', datetime('now')),
			('project2', 'Test Project 2', datetime('now'));
	`
	if _, err := db.Exec(schema); err != nil {
		b.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func BenchmarkSeedManager_GetSeeds(b *testing.B) {
	db := setupSeedBenchDB(b)
	defer db.Close()

	sm, err := NewSeedManager(db)
	if err != nil {
		b.Fatalf("Failed to create SeedManager: %v", err)
	}

	ctx := context.Background()
	projectID := "project1"

	// Pre-populate with 100 seeds
	for i := 0; i < 100; i++ {
		content := fmt.Sprintf("Test seed content %d", i)
		_, err := sm.CreateUserSeed(ctx, &projectID, content, "test", "user1")
		if err != nil {
			b.Fatalf("Failed to create seed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sm.GetSeeds(ctx, &projectID, nil)
		if err != nil {
			b.Fatalf("Failed to get seeds: %v", err)
		}
	}
}

func BenchmarkSeedManager_GetSeedByID(b *testing.B) {
	db := setupSeedBenchDB(b)
	defer db.Close()

	sm, err := NewSeedManager(db)
	if err != nil {
		b.Fatalf("Failed to create SeedManager: %v", err)
	}

	ctx := context.Background()
	projectID := "project1"
	seed, err := sm.CreateUserSeed(ctx, &projectID, "Test seed content", "test", "user1")
	if err != nil {
		b.Fatalf("Failed to create seed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sm.GetSeedByID(ctx, seed.ID)
		if err != nil {
			b.Fatalf("Failed to get seed by ID: %v", err)
		}
	}
}

func BenchmarkSeedManager_CreateUserSeed(b *testing.B) {
	db := setupSeedBenchDB(b)
	defer db.Close()

	sm, err := NewSeedManager(db)
	if err != nil {
		b.Fatalf("Failed to create SeedManager: %v", err)
	}

	ctx := context.Background()
	projectID := "project1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		content := fmt.Sprintf("Test seed content %d", i)
		_, err := sm.CreateUserSeed(ctx, &projectID, content, "test", "user1")
		if err != nil {
			b.Fatalf("Failed to create seed: %v", err)
		}
	}
}

func BenchmarkSeedManager_UpdateSeed(b *testing.B) {
	db := setupSeedBenchDB(b)
	defer db.Close()

	sm, err := NewSeedManager(db)
	if err != nil {
		b.Fatalf("Failed to create SeedManager: %v", err)
	}

	ctx := context.Background()
	projectID := "project1"
	userID := "user1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Create a fresh seed for each iteration to avoid conflicts
		seed, err := sm.CreateUserSeed(ctx, &projectID, "Original content", "test", userID)
		if err != nil {
			b.Fatalf("Failed to create seed: %v", err)
		}
		b.StartTimer()

		newContent := fmt.Sprintf("Updated content %d", i)
		_, err = sm.UpdateSeed(ctx, seed.ID, newContent, &userID)
		if err != nil {
			b.Fatalf("Failed to update seed: %v", err)
		}
	}
}

func BenchmarkSeedManager_DeleteSeed(b *testing.B) {
	db := setupSeedBenchDB(b)
	defer db.Close()

	sm, err := NewSeedManager(db)
	if err != nil {
		b.Fatalf("Failed to create SeedManager: %v", err)
	}

	ctx := context.Background()
	projectID := "project1"
	userID := "user1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Create a fresh seed for each iteration
		seed, err := sm.CreateUserSeed(ctx, &projectID, "Content to delete", "test", userID)
		if err != nil {
			b.Fatalf("Failed to create seed: %v", err)
		}
		b.StartTimer()

		err = sm.DeleteSeed(ctx, seed.ID, &userID)
		if err != nil {
			b.Fatalf("Failed to delete seed: %v", err)
		}
	}
}

func BenchmarkSeedManager_IncrementUsage(b *testing.B) {
	db := setupSeedBenchDB(b)
	defer db.Close()

	sm, err := NewSeedManager(db)
	if err != nil {
		b.Fatalf("Failed to create SeedManager: %v", err)
	}

	ctx := context.Background()
	projectID := "project1"
	seed, err := sm.CreateUserSeed(ctx, &projectID, "Test seed content", "test", "user1")
	if err != nil {
		b.Fatalf("Failed to create seed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := sm.IncrementUsage(ctx, seed.ID)
		if err != nil {
			b.Fatalf("Failed to increment usage: %v", err)
		}
	}
}

func BenchmarkSeedManager_BulkCreate_100Seeds(b *testing.B) {
	db := setupSeedBenchDB(b)
	defer db.Close()

	sm, err := NewSeedManager(db)
	if err != nil {
		b.Fatalf("Failed to create SeedManager: %v", err)
	}

	ctx := context.Background()
	projectID := "project1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			content := fmt.Sprintf("Bulk seed content %d-%d", i, j)
			_, err := sm.CreateUserSeed(ctx, &projectID, content, "test", "user1")
			if err != nil {
				b.Fatalf("Failed to create seed: %v", err)
			}
		}
	}
}

func BenchmarkSeedManager_GetSeedsWithFilters(b *testing.B) {
	db := setupSeedBenchDB(b)
	defer db.Close()

	sm, err := NewSeedManager(db)
	if err != nil {
		b.Fatalf("Failed to create SeedManager: %v", err)
	}

	ctx := context.Background()
	projectID := "project1"

	// Pre-populate with seeds of different types
	for i := 0; i < 50; i++ {
		seedType := "type1"
		if i%2 == 0 {
			seedType = "type2"
		}
		content := fmt.Sprintf("Test seed content %d", i)
		_, err := sm.CreateUserSeed(ctx, &projectID, content, seedType, "user1")
		if err != nil {
			b.Fatalf("Failed to create seed: %v", err)
		}
	}

	filters := map[string]interface{}{
		"seed_type": "type1",
		"source":    SourceUser,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sm.GetSeeds(ctx, &projectID, filters)
		if err != nil {
			b.Fatalf("Failed to get filtered seeds: %v", err)
		}
	}
}
