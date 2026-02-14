-- ============================================================================
-- Rollback Migration: v0.0.20 to v0.0.19
-- ============================================================================
-- Migration Date: February 1, 2026
-- Purpose: Rollback database from v0.0.20 (The Compassionate Companion) to v0.0.19
-- Target: SQLite database at .dojo/memory.db
--
-- WARNING: This script will:
-- - DROP 8 tables (project_suggestions, action_patterns, project_activity_log,
--   goals, goal_steps, goal_artifact_links, judgment_decisions, quiet_hours)
-- - REMOVE 2 columns from projects table (suggestion_enabled, suggestion_sensitivity)
--
-- IMPORTANT: SQLite does not support DROP COLUMN directly.
-- To remove columns, we must recreate the projects table with the old schema.
--
-- BACKUP YOUR DATABASE BEFORE RUNNING THIS SCRIPT!
--
-- ⚠️  PERFORMANCE WARNING - LARGE DATABASES:
-- This rollback recreates the projects table to remove columns.
-- Performance impact:
-- - Database will be LOCKED during the entire rollback operation
-- - Data is copied twice (INSERT then DROP/RENAME)
-- - Requires 2x disk space temporarily (old table + new table)
-- 
-- Estimated rollback time:
-- - Small DB (<100 projects): <1 second
-- - Medium DB (100-1,000 projects): 1-3 seconds
-- - Large DB (1,000-10,000 projects): 3-10 seconds
--
-- For production databases with >1,000 records:
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
-- Drop v0.0.20 Tables (in reverse order of creation)
-- ============================================================================

-- Drop indexes first for cleaner rollback
DROP INDEX IF EXISTS idx_quiet_hours_day;
DROP INDEX IF EXISTS idx_decisions_target;
DROP INDEX IF EXISTS idx_decisions_feedback;
DROP INDEX IF EXISTS idx_decisions_created;
DROP INDEX IF EXISTS idx_goal_links_step;
DROP INDEX IF EXISTS idx_goal_links_artifact;
DROP INDEX IF EXISTS idx_goal_links_goal;
DROP INDEX IF EXISTS idx_steps_completed;
DROP INDEX IF EXISTS idx_steps_goal;
DROP INDEX IF EXISTS idx_goals_project_status;
DROP INDEX IF EXISTS idx_goals_updated;
DROP INDEX IF EXISTS idx_goals_deadline;
DROP INDEX IF EXISTS idx_goals_status;
DROP INDEX IF EXISTS idx_goals_project;
DROP INDEX IF EXISTS idx_activity_project_created;
DROP INDEX IF EXISTS idx_activity_type;
DROP INDEX IF EXISTS idx_activity_created;
DROP INDEX IF EXISTS idx_activity_project;
DROP INDEX IF EXISTS idx_patterns_unique;
DROP INDEX IF EXISTS idx_patterns_confidence;
DROP INDEX IF EXISTS idx_patterns_frequency;
DROP INDEX IF EXISTS idx_patterns_project;
DROP INDEX IF EXISTS idx_suggestions_created;
DROP INDEX IF EXISTS idx_suggestions_confidence;
DROP INDEX IF EXISTS idx_suggestions_status;
DROP INDEX IF EXISTS idx_suggestions_project;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS quiet_hours;
DROP TABLE IF EXISTS judgment_decisions;
DROP TABLE IF EXISTS goal_artifact_links;
DROP TABLE IF EXISTS goal_steps;
DROP TABLE IF EXISTS goals;
DROP TABLE IF EXISTS project_activity_log;
DROP TABLE IF EXISTS action_patterns;
DROP TABLE IF EXISTS project_suggestions;

-- ============================================================================
-- Remove Columns from Projects Table
-- ============================================================================
-- SQLite does not support DROP COLUMN, so we must:
-- 1. Create new table with v0.0.19 schema
-- 2. Copy data from old table (excluding new columns)
-- 3. Drop old table
-- 4. Rename new table
-- 5. Recreate all indexes and foreign keys

-- Create backup table with v0.0.19 schema
CREATE TABLE IF NOT EXISTS projects_v0_0_19 (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    last_accessed_at DATETIME,
    status TEXT DEFAULT 'active' CHECK(status IN ('active', 'archived', 'deleted'))
);

-- Copy data from current projects table (excluding suggestion_enabled and suggestion_sensitivity)
INSERT INTO projects_v0_0_19 (id, name, description, created_at, updated_at, last_accessed_at, status)
SELECT id, name, description, created_at, updated_at, last_accessed_at, status FROM projects;

-- Drop current projects table
DROP TABLE projects;

-- Rename backup table to projects
ALTER TABLE projects_v0_0_19 RENAME TO projects;

-- Recreate indexes for projects table (from v0.0.19)
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_updated ON projects(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_projects_last_accessed ON projects(last_accessed_at DESC);

-- ============================================================================
-- Remove Migration Version Record
-- ============================================================================

-- Delete the migration version record
DELETE FROM schema_migrations WHERE version = '20260201_v0.0.20';

-- ============================================================================
-- Rollback Complete
-- ============================================================================
-- Tables Dropped: 8
--   - project_suggestions
--   - action_patterns
--   - project_activity_log
--   - goals
--   - goal_steps
--   - goal_artifact_links
--   - judgment_decisions
--   - quiet_hours
-- Columns Removed:
--   - projects: suggestion_enabled, suggestion_sensitivity
-- Indexes Dropped: 25
-- Data Preserved:
--   - All data from projects table retained (columns excluded)
-- Migration Record: Removed from schema_migrations
-- ============================================================================

-- Commit transaction
COMMIT;
