-- ============================================================================
-- Rollback Migration: v0.0.23 to v0.0.22
-- ============================================================================
-- Migration Date: February 3, 2026
-- Purpose: Rollback database from v0.0.23 (The Collaborative Calibration) to v0.0.22
-- Target: SQLite database at .dojo/memory.db
--
-- WARNING: This script will:
-- - DROP 5 tables (user_feedback, calibration_preferences, calibration_events, 
--                  memory_seeds, judgment_decisions)
-- - DELETE all feedback, preferences, calibration events, memory seeds, and judgment data
--
-- BACKUP YOUR DATABASE BEFORE RUNNING THIS SCRIPT!
--
-- ⚠️  DATA LOSS WARNING:
-- This rollback will permanently delete:
-- - All user feedback on AI suggestions and actions
-- - All learned user preferences and calibration adjustments
-- - All calibration event history (audit trail)
-- - All memory seeds (both system and user-created)
-- - All judgment layer decision history
--
-- ⚠️  PERFORMANCE WARNING - LARGE DATABASES:
-- Estimated rollback time:
-- - Small DB (<1,000 feedback entries): <1 second
-- - Medium DB (1,000-10,000 feedback entries): 1-3 seconds
-- - Large DB (10,000-100,000 feedback entries): 3-10 seconds
--
-- For production databases with >10,000 feedback records:
-- 1. Schedule rollback during maintenance window
-- 2. Stop application to prevent write conflicts
-- 3. Test rollback on a backup copy first
-- ============================================================================

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Begin transaction to ensure atomic rollback
BEGIN TRANSACTION;

-- ============================================================================
-- Drop v0.0.23 Indexes (for cleaner rollback)
-- ============================================================================

-- Judgment Decisions indexes
DROP INDEX IF EXISTS idx_judgment_decisions_created;
DROP INDEX IF EXISTS idx_judgment_decisions_outcome;
DROP INDEX IF EXISTS idx_judgment_decisions_suggestion;
DROP INDEX IF EXISTS idx_judgment_decisions_type;
DROP INDEX IF EXISTS idx_judgment_decisions_user;

-- Memory Seeds indexes
DROP INDEX IF EXISTS idx_memory_seeds_created;
DROP INDEX IF EXISTS idx_memory_seeds_usage;
DROP INDEX IF EXISTS idx_memory_seeds_confidence;
DROP INDEX IF EXISTS idx_memory_seeds_editable;
DROP INDEX IF EXISTS idx_memory_seeds_source;
DROP INDEX IF EXISTS idx_memory_seeds_type;
DROP INDEX IF EXISTS idx_memory_seeds_project;

-- Calibration Events indexes
DROP INDEX IF EXISTS idx_calibration_events_created;
DROP INDEX IF EXISTS idx_calibration_events_key;
DROP INDEX IF EXISTS idx_calibration_events_type;
DROP INDEX IF EXISTS idx_calibration_events_user;

-- Calibration Preferences indexes
DROP INDEX IF EXISTS idx_calibration_preferences_updated;
DROP INDEX IF EXISTS idx_calibration_preferences_confidence;
DROP INDEX IF EXISTS idx_calibration_preferences_key;
DROP INDEX IF EXISTS idx_calibration_preferences_user;

-- User Feedback indexes
DROP INDEX IF EXISTS idx_user_feedback_created;
DROP INDEX IF EXISTS idx_user_feedback_rating;
DROP INDEX IF EXISTS idx_user_feedback_processed;
DROP INDEX IF EXISTS idx_user_feedback_user;
DROP INDEX IF EXISTS idx_user_feedback_source;

-- ============================================================================
-- Drop v0.0.23 Tables (in reverse order of dependencies)
-- ============================================================================

-- Drop tables with foreign key dependencies first
DROP TABLE IF EXISTS judgment_decisions;

-- Drop remaining tables
DROP TABLE IF EXISTS memory_seeds;
DROP TABLE IF EXISTS calibration_events;
DROP TABLE IF EXISTS calibration_preferences;
DROP TABLE IF EXISTS user_feedback;

-- ============================================================================
-- Remove Migration Version Record
-- ============================================================================

-- Delete the migration version record
DELETE FROM schema_migrations WHERE version = '20260203000000_v0.0.23';

-- ============================================================================
-- Rollback Complete
-- ============================================================================
-- Tables Dropped: 5
--   - user_feedback
--   - calibration_preferences
--   - calibration_events
--   - memory_seeds
--   - judgment_decisions
-- Indexes Dropped: 25
-- Data Preserved: None (all v0.0.23 data deleted)
-- Migration Record: Removed from schema_migrations
-- Database State: Reverted to v0.0.22
-- ============================================================================

-- Commit transaction
COMMIT;
