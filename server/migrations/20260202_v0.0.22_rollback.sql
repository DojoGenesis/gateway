-- ============================================================================
-- Rollback Migration: v0.0.22 to v0.0.20
-- ============================================================================
-- Migration Date: February 2, 2026
-- Purpose: Rollback database from v0.0.22 (The Living Interface) to v0.0.20
-- Target: SQLite database at .dojo/memory.db
--
-- WARNING: This script will:
-- - DROP 3 tables (layout_transitions, user_preferences, command_history)
-- - DELETE all layout tracking, preferences, and command history data
--
-- BACKUP YOUR DATABASE BEFORE RUNNING THIS SCRIPT!
--
-- ⚠️  DATA LOSS WARNING:
-- This rollback will permanently delete:
-- - All layout transition history (analytics for layout mode changes)
-- - All user notification preferences (will revert to defaults on re-migration)
-- - All command usage history (analytics for command usage patterns)
--
-- ⚠️  PERFORMANCE WARNING - LARGE DATABASES:
-- Estimated rollback time:
-- - Small DB (<1,000 commands): <1 second
-- - Medium DB (1,000-10,000 commands): 1-2 seconds
-- - Large DB (10,000-100,000 commands): 2-5 seconds
--
-- For production databases with >10,000 command records:
-- 1. Schedule rollback during maintenance window
-- 2. Stop application to prevent write conflicts
-- 3. Test rollback on a backup copy first
-- ============================================================================

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Begin transaction to ensure atomic rollback
BEGIN TRANSACTION;

-- ============================================================================
-- Drop v0.0.22 Indexes (for cleaner rollback)
-- ============================================================================

-- Command History indexes
DROP INDEX IF EXISTS idx_command_history_project_type;
DROP INDEX IF EXISTS idx_command_history_success;
DROP INDEX IF EXISTS idx_command_history_type;
DROP INDEX IF EXISTS idx_command_history_created;
DROP INDEX IF EXISTS idx_command_history_project;
DROP INDEX IF EXISTS idx_command_history_session;

-- Layout Transitions indexes
DROP INDEX IF EXISTS idx_layout_transitions_modes;
DROP INDEX IF EXISTS idx_layout_transitions_trigger;
DROP INDEX IF EXISTS idx_layout_transitions_created;
DROP INDEX IF EXISTS idx_layout_transitions_project;
DROP INDEX IF EXISTS idx_layout_transitions_session;

-- ============================================================================
-- Drop v0.0.22 Tables (in reverse order of dependencies)
-- ============================================================================

-- Drop tables in reverse order
DROP TABLE IF EXISTS command_history;
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS layout_transitions;

-- ============================================================================
-- Remove Migration Version Record
-- ============================================================================

-- Delete the migration version record
DELETE FROM schema_migrations WHERE version = '20260202_v0.0.22';

-- ============================================================================
-- Rollback Complete
-- ============================================================================
-- Tables Dropped: 3
--   - layout_transitions
--   - user_preferences
--   - command_history
-- Indexes Dropped: 11
-- Data Preserved: None (all v0.0.22 data deleted)
-- Migration Record: Removed from schema_migrations
-- Database State: Reverted to v0.0.20
-- ============================================================================

-- Commit transaction
COMMIT;
