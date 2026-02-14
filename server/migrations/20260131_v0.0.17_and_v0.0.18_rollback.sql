-- ============================================================================
-- Rollback Migration: v0.0.17 + v0.0.18 to v0.0.16
-- ============================================================================
-- Migration Date: January 31, 2026
-- Purpose: Rollback database from v0.0.17/v0.0.18 schemas to v0.0.16 baseline
-- Target: SQLite database at .dojo/memory.db
--
-- WARNING: This script will DROP all tables created by the v0.0.17 and v0.0.18
-- migration. All data in these tables will be PERMANENTLY DELETED.
--
-- BACKUP YOUR DATABASE BEFORE RUNNING THIS SCRIPT!
-- ============================================================================

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Begin transaction to ensure atomic rollback
BEGIN TRANSACTION;

-- ============================================================================
-- Drop v0.0.17 Tables (in reverse dependency order)
-- ============================================================================

-- Drop traces table (no dependencies)
DROP TABLE IF EXISTS traces;

-- Drop memory_seeds table (no dependencies)
DROP TABLE IF EXISTS memory_seeds;

-- ============================================================================
-- Drop v0.0.18 Tables (in reverse dependency order)
-- ============================================================================

-- Drop project_files table (depends on projects)
DROP TABLE IF EXISTS project_files;

-- Drop artifact_versions table (depends on artifacts)
DROP TABLE IF EXISTS artifact_versions;

-- Drop artifacts table (depends on projects)
DROP TABLE IF EXISTS artifacts;

-- Drop project_templates table (no dependencies, but referenced by projects)
DROP TABLE IF EXISTS project_templates;

-- Drop projects table (base table)
DROP TABLE IF EXISTS projects;

-- ============================================================================
-- Remove Migration Version Record
-- ============================================================================

-- Delete the migration version record
DELETE FROM schema_migrations WHERE version = '20260131_v0.0.17_and_v0.0.18';

-- Optionally drop the schema_migrations table if it's now empty
-- Note: Comment this out if you have other migrations tracked
-- DROP TABLE IF EXISTS schema_migrations;

-- ============================================================================
-- Optional: Remove project_id from memories table
-- ============================================================================

-- WARNING: SQLite does not support DROP COLUMN directly.
-- To remove project_id from memories table, you would need to:
-- 1. Create a new table without project_id column
-- 2. Copy data from old table to new table
-- 3. Drop old table
-- 4. Rename new table
--
-- This is commented out for safety. Uncomment and modify if needed:

-- CREATE TABLE IF NOT EXISTS memories_backup (
--     id TEXT PRIMARY KEY,
--     type TEXT NOT NULL,
--     content TEXT NOT NULL,
--     metadata TEXT,
--     created_at DATETIME NOT NULL,
--     updated_at DATETIME NOT NULL
-- );
--
-- INSERT INTO memories_backup (id, type, content, metadata, created_at, updated_at)
-- SELECT id, type, content, metadata, created_at, updated_at FROM memories;
--
-- DROP TABLE memories;
--
-- ALTER TABLE memories_backup RENAME TO memories;
--
-- CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
-- CREATE INDEX IF NOT EXISTS idx_memories_created_at ON memories(created_at);
-- CREATE INDEX IF NOT EXISTS idx_memories_updated_at ON memories(updated_at);
-- CREATE INDEX IF NOT EXISTS idx_memories_content_fts ON memories(content);

-- ============================================================================
-- Rollback Complete
-- ============================================================================
-- Tables Dropped:
--   v0.0.18: projects, project_templates, artifacts, artifact_versions, project_files
--   v0.0.17: memory_seeds, traces
-- Migration Record: Removed from schema_migrations
-- ============================================================================

-- Commit transaction
COMMIT;
