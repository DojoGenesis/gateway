-- ============================================================================
-- Rollback Migration: v0.0.19 to v0.0.18
-- ============================================================================
-- Migration Date: February 1, 2026
-- Purpose: Rollback database from v0.0.19 (The Surgical Mind) to v0.0.18
-- Target: SQLite database at .dojo/memory.db
--
-- WARNING: This script will:
-- - DROP the memory_files table and all its data
-- - REMOVE embedding and context_type columns from memories table
-- - REMOVE context_type column from memory_seeds table
--
-- IMPORTANT: SQLite does not support DROP COLUMN directly.
-- To remove columns, we must recreate the tables with the old schema.
--
-- BACKUP YOUR DATABASE BEFORE RUNNING THIS SCRIPT!
--
-- ⚠️  PERFORMANCE WARNING - LARGE DATABASES:
-- This rollback recreates the memories and memory_seeds tables to remove columns.
-- Performance impact:
-- - Database will be LOCKED during the entire rollback operation
-- - Data is copied twice (INSERT then DROP/RENAME) for each table
-- - Requires 2x disk space temporarily (old table + new table)
-- 
-- Estimated rollback time:
-- - Small DB (<1,000 memories): <1 second
-- - Medium DB (1,000-10,000 memories): 1-5 seconds
-- - Large DB (10,000-100,000 memories): 5-30 seconds
-- - Very Large DB (>100,000 memories): 30+ seconds
--
-- For production databases with >10,000 records:
-- 1. Schedule rollback during maintenance window
-- 2. Stop application to prevent write conflicts
-- 3. Monitor disk space (ensure at least 2x current DB size available)
-- 4. Test rollback on a backup copy first
-- ============================================================================

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Begin transaction to ensure atomic rollback
BEGIN TRANSACTION;

-- ============================================================================
-- Drop memory_files table
-- ============================================================================

-- Drop all indexes first
DROP INDEX IF EXISTS idx_memory_files_tier;
DROP INDEX IF EXISTS idx_memory_files_archived;
DROP INDEX IF EXISTS idx_memory_files_path;
DROP INDEX IF EXISTS idx_memory_files_tier_archived;

-- Drop the table
DROP TABLE IF EXISTS memory_files;

-- ============================================================================
-- Remove columns from memories table
-- ============================================================================
-- SQLite does not support DROP COLUMN, so we must:
-- 1. Create new table with old schema
-- 2. Copy data from old table (excluding new columns)
-- 3. Drop old table
-- 4. Rename new table

-- Create backup table with v0.0.18 schema
CREATE TABLE IF NOT EXISTS memories_v0_0_18 (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    content TEXT NOT NULL,
    metadata TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

-- Copy data from current memories table (excluding embedding and context_type)
INSERT INTO memories_v0_0_18 (id, type, content, metadata, created_at, updated_at)
SELECT id, type, content, metadata, created_at, updated_at FROM memories;

-- Drop current memories table
DROP TABLE memories;

-- Rename backup table to memories
ALTER TABLE memories_v0_0_18 RENAME TO memories;

-- Recreate indexes for memories table
CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
CREATE INDEX IF NOT EXISTS idx_memories_created_at ON memories(created_at);
CREATE INDEX IF NOT EXISTS idx_memories_updated_at ON memories(updated_at);
CREATE INDEX IF NOT EXISTS idx_memories_content_fts ON memories(content);

-- Drop v0.0.19 indexes that no longer exist
DROP INDEX IF EXISTS idx_memories_context_type;
DROP INDEX IF EXISTS idx_memory_seeds_context_type;

-- ============================================================================
-- Remove context_type from memory_seeds table
-- ============================================================================

-- Create backup table with v0.0.18 schema
CREATE TABLE IF NOT EXISTS memory_seeds_v0_0_18 (
    id TEXT PRIMARY KEY,
    project_id TEXT,
    type TEXT NOT NULL,
    content TEXT NOT NULL,
    embedding BLOB,
    source_memory_ids TEXT,
    confidence REAL DEFAULT 1.0,
    tier INTEGER DEFAULT 3,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    last_accessed_at DATETIME NOT NULL,
    access_count INTEGER DEFAULT 0,
    metadata TEXT
);

-- Copy data from current memory_seeds table (excluding context_type)
INSERT INTO memory_seeds_v0_0_18 (
    id, project_id, type, content, embedding, source_memory_ids,
    confidence, tier, created_at, updated_at, last_accessed_at,
    access_count, metadata
)
SELECT 
    id, project_id, type, content, embedding, source_memory_ids,
    confidence, tier, created_at, updated_at, last_accessed_at,
    access_count, metadata
FROM memory_seeds;

-- Drop current memory_seeds table
DROP TABLE memory_seeds;

-- Rename backup table to memory_seeds
ALTER TABLE memory_seeds_v0_0_18 RENAME TO memory_seeds;

-- Recreate indexes for memory_seeds table
CREATE INDEX IF NOT EXISTS idx_seeds_project ON memory_seeds(project_id);
CREATE INDEX IF NOT EXISTS idx_seeds_type ON memory_seeds(type);
CREATE INDEX IF NOT EXISTS idx_seeds_tier ON memory_seeds(tier);
CREATE INDEX IF NOT EXISTS idx_seeds_confidence ON memory_seeds(confidence DESC);
CREATE INDEX IF NOT EXISTS idx_seeds_last_accessed ON memory_seeds(last_accessed_at DESC);

-- Drop v0.0.19 indexes that no longer exist (already dropped above)
-- DROP INDEX IF EXISTS idx_memory_seeds_context_type;

-- ============================================================================
-- Remove Migration Version Record
-- ============================================================================

-- Delete the migration version record
DELETE FROM schema_migrations WHERE version = '20260201_v0.0.19';

-- ============================================================================
-- Rollback Complete
-- ============================================================================
-- Tables Dropped:
--   memory_files (with 4 indexes)
-- Columns Removed:
--   memories: embedding, context_type
--   memory_seeds: context_type
-- Data Preserved:
--   All data from memories and memory_seeds retained (columns excluded)
-- Migration Record: Removed from schema_migrations
-- ============================================================================

-- Commit transaction
COMMIT;
