-- ============================================================================
-- Rollback Migration: v0.0.26 (The Proactive Intelligence System)
-- ============================================================================
-- Migration Date: February 5, 2026
-- Migration Version: 20260205_v0.0.26_rollback
-- 
-- This rollback script removes all tables and indexes created by v0.0.26
-- migrations (voice_calls and monitoring).
--
-- **WARNING: This will permanently delete all data in the following tables:**
-- - voice_calls
-- - voice_call_actions
-- - monitored_emails
-- - monitored_events
-- - checkin_history
--
-- **BACKUP YOUR DATABASE BEFORE RUNNING THIS SCRIPT!**
--
-- Usage:
--   sqlite3 .dojo/memory.db < go_backend/migrations/20260205_v0.0.26_rollback.sql
-- ============================================================================

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Begin transaction to ensure atomic rollback
BEGIN TRANSACTION;

-- ============================================================================
-- Drop Tables (in reverse dependency order)
-- ============================================================================

-- Drop voice_call_actions first (has FK to voice_calls)
DROP TABLE IF EXISTS voice_call_actions;

-- Drop voice_calls
DROP TABLE IF EXISTS voice_calls;

-- Drop monitoring tables
DROP TABLE IF EXISTS monitored_emails;
DROP TABLE IF EXISTS monitored_events;
DROP TABLE IF EXISTS checkin_history;

-- ============================================================================
-- Remove Migration Records
-- ============================================================================

DELETE FROM schema_migrations WHERE version = '20260205_v0.0.26_voice_calls';
DELETE FROM schema_migrations WHERE version = '20260205_v0.0.26_monitoring';

-- ============================================================================
-- Rollback Complete
-- ============================================================================
-- Tables Dropped: 5 total
--   - voice_call_actions
--   - voice_calls
--   - monitored_emails
--   - monitored_events
--   - checkin_history
-- All associated indexes automatically dropped with tables
-- ============================================================================

-- Commit transaction
COMMIT;
