-- ============================================================================
-- Rollback Migration: v0.0.30 Track C - Local-First Auth Foundation
-- ============================================================================
-- Migration Date: February 7, 2026
-- Migration Version: 20260207_v0.0.30_rollback
-- 
-- **WARNING: This rollback will permanently delete all local user data**
-- Target Database: SQLite at .dojo/dojo.db
--
-- This rollback script removes all tables created by the v0.0.30 migration.
-- Data loss is permanent and cannot be recovered after rollback.
--
-- USAGE:
--   sqlite3 .dojo/dojo.db < go_backend/migrations/20260207_v0.0.30_rollback.sql
--
-- IMPORTANT NOTES:
-- - Always backup your database before running this rollback
-- - All user data, API keys, conversations, and settings will be deleted
-- - This rollback is atomic (transaction-wrapped)
-- ============================================================================

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Begin transaction to ensure atomic rollback
BEGIN TRANSACTION;

-- ============================================================================
-- Drop Tables (in reverse dependency order)
-- ============================================================================

-- Drop migration_log (depends on local_users)
DROP TABLE IF EXISTS migration_log;

-- Drop user_settings (depends on local_users)
DROP TABLE IF EXISTS user_settings;

-- Drop conversations (depends on local_users)
DROP TABLE IF EXISTS conversations;

-- Drop api_keys (depends on local_users)
DROP TABLE IF EXISTS api_keys;

-- Drop local_users (base table)
DROP TABLE IF EXISTS local_users;

-- ============================================================================
-- Remove Migration Record
-- ============================================================================

DELETE FROM schema_migrations 
WHERE version = '20260207_v0.0.30_local_auth';

-- ============================================================================
-- Rollback Complete
-- ============================================================================
-- Tables Dropped:
--   - local_users
--   - api_keys
--   - conversations
--   - user_settings
--   - migration_log
-- Migration Record: Removed from schema_migrations
-- ============================================================================

-- Commit transaction
COMMIT;
