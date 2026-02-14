-- ============================================================================
-- Migration: v0.0.26 (The Proactive Intelligence System - Monitoring)
-- ============================================================================
-- Migration Date: February 5, 2026
-- Migration Version: 20260205_v0.0.26_monitoring
-- 
-- **REQUIRES: v0.0.24 as base version**
-- Base Version: v0.0.24 (Migration: 20260204_v0.0.24)
-- Target Database: SQLite at .dojo/memory.db
--
-- This migration adds:
-- - Monitored Emails: Track emails from Gmail for check-in generation
-- - Monitored Events: Track calendar events for check-in generation
-- - Check-In History: Track all check-ins sent with perspectives and decisions
--
-- IMPORTANT NOTES:
-- - This migration is atomic (transaction-wrapped) and can be safely retried
-- - All operations preserve existing data
-- - New tables support multi-channel monitoring and smart gating
-- - Integrates with Gmail API and Google Calendar API
-- ============================================================================

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Begin transaction to ensure atomic migration
BEGIN TRANSACTION;

-- ============================================================================
-- Migration Version Tracking
-- ============================================================================

-- Record this migration
INSERT INTO schema_migrations (version, applied_at, description)
VALUES ('20260205_v0.0.26_monitoring', datetime('now'), 'v0.0.26: The Proactive Intelligence System - Monitoring, Check-In History');

-- ============================================================================
-- Monitored Emails: Gmail Integration
-- ============================================================================

-- Tracks emails monitored from Gmail for proactive check-ins
-- Stores email metadata and generated perspectives
CREATE TABLE IF NOT EXISTS monitored_emails (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    email_id TEXT NOT NULL UNIQUE,      -- Gmail message ID
    sender TEXT NOT NULL,
    subject TEXT NOT NULL,
    snippet TEXT,                        -- Email preview text
    received_at DATETIME NOT NULL,
    processed_at DATETIME,
    perspectives_generated TEXT,         -- JSON array: perspectives generated for this email
    notification_sent BOOLEAN DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_monitored_emails_user_id ON monitored_emails(user_id);
CREATE INDEX IF NOT EXISTS idx_monitored_emails_email_id ON monitored_emails(email_id);
CREATE INDEX IF NOT EXISTS idx_monitored_emails_processed_at ON monitored_emails(processed_at);
CREATE INDEX IF NOT EXISTS idx_monitored_emails_received ON monitored_emails(received_at DESC);
CREATE INDEX IF NOT EXISTS idx_monitored_emails_notification ON monitored_emails(notification_sent);

-- ============================================================================
-- Monitored Events: Calendar Integration
-- ============================================================================

-- Tracks calendar events monitored from Google Calendar for proactive check-ins
-- Stores event metadata and generated perspectives
CREATE TABLE IF NOT EXISTS monitored_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    event_id TEXT NOT NULL UNIQUE,      -- Google Calendar event ID
    title TEXT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    location TEXT,
    attendees TEXT,                      -- JSON array: event attendees
    processed_at DATETIME,
    perspectives_generated TEXT,         -- JSON array: perspectives generated for this event
    notification_sent BOOLEAN DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_monitored_events_user_id ON monitored_events(user_id);
CREATE INDEX IF NOT EXISTS idx_monitored_events_event_id ON monitored_events(event_id);
CREATE INDEX IF NOT EXISTS idx_monitored_events_start_time ON monitored_events(start_time);
CREATE INDEX IF NOT EXISTS idx_monitored_events_processed ON monitored_events(processed_at);
CREATE INDEX IF NOT EXISTS idx_monitored_events_notification ON monitored_events(notification_sent);

-- ============================================================================
-- Check-In History: Decision Tracking
-- ============================================================================

-- Tracks all check-in notifications sent to users
-- Stores perspectives, gating decisions, and user responses
-- Source types: 'email', 'calendar', 'task', 'goal'
-- Decisions: 'speak' (notification sent), 'silent' (suppressed), 'ask' (pending approval)
-- User responses: 'dig_deeper' (user wants more info), 'noted' (acknowledged), 'silence' (suppress similar)
CREATE TABLE IF NOT EXISTS checkin_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    check_cycle_id TEXT NOT NULL,       -- Unique ID for this check cycle
    source_type TEXT NOT NULL CHECK(source_type IN ('email', 'calendar', 'task', 'goal', 'other')),
    source_id TEXT NOT NULL,             -- ID of the email, event, task, or goal
    perspectives TEXT NOT NULL,          -- JSON array: perspectives generated
    decision TEXT NOT NULL CHECK(decision IN ('speak', 'silent', 'ask')),
    decision_reason TEXT,                -- Why this decision was made (for learning)
    notification_sent BOOLEAN DEFAULT FALSE,
    user_response TEXT CHECK(user_response IN ('dig_deeper', 'noted', 'silence', NULL)),
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_checkin_history_user_id ON checkin_history(user_id);
CREATE INDEX IF NOT EXISTS idx_checkin_history_cycle_id ON checkin_history(check_cycle_id);
CREATE INDEX IF NOT EXISTS idx_checkin_history_created_at ON checkin_history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_checkin_history_source_type ON checkin_history(source_type);
CREATE INDEX IF NOT EXISTS idx_checkin_history_decision ON checkin_history(decision);
CREATE INDEX IF NOT EXISTS idx_checkin_history_response ON checkin_history(user_response);

-- ============================================================================
-- OAuth Tokens: External Service Authentication
-- ============================================================================

-- Stores OAuth tokens for external services (Gmail, Calendar)
-- Supports automatic token refresh
CREATE TABLE IF NOT EXISTS oauth_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    service TEXT NOT NULL,              -- 'gmail', 'calendar', etc.
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type TEXT NOT NULL,
    expiry DATETIME NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, service)
);

CREATE INDEX IF NOT EXISTS idx_oauth_tokens_user_service ON oauth_tokens(user_id, service);

-- ============================================================================
-- Monitoring Preferences: User Preferences for Proactive Monitoring
-- ============================================================================

-- Stores user preferences for proactive monitoring
-- Controls which sources are monitored and notification frequency
CREATE TABLE IF NOT EXISTS monitoring_preferences (
    user_id TEXT PRIMARY KEY,
    max_daily_checkins INTEGER NOT NULL DEFAULT 5,
    enabled_sources TEXT,               -- JSON array: enabled source types (email, calendar, etc.)
    check_interval_minutes INTEGER NOT NULL DEFAULT 30,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_monitoring_preferences_enabled ON monitoring_preferences(enabled);

-- ============================================================================
-- Check Cycle Results: Monitoring Service Execution Logs
-- ============================================================================

-- Logs results of each monitoring check cycle
-- Tracks data collected, perspectives generated, and errors
CREATE TABLE IF NOT EXISTS check_cycle_results (
    check_cycle_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    data_collected TEXT,                -- JSON: emails and events collected
    perspectives TEXT,                  -- JSON array: all perspectives generated
    checkins_created INTEGER NOT NULL DEFAULT 0,
    notifications_sent INTEGER NOT NULL DEFAULT 0,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    duration_ms INTEGER NOT NULL,
    errors TEXT,                        -- JSON array: errors encountered
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_check_cycle_results_user_id ON check_cycle_results(user_id);
CREATE INDEX IF NOT EXISTS idx_check_cycle_results_start_time ON check_cycle_results(start_time DESC);

-- ============================================================================
-- Migration Complete
-- ============================================================================
-- Tables Created: 6 total
--   - monitored_emails (5 indexes)
--   - monitored_events (5 indexes)
--   - checkin_history (6 indexes)
--   - oauth_tokens (1 index)
--   - monitoring_preferences (1 index)
--   - check_cycle_results (2 indexes)
-- Indexes Created: 20 total
-- Default Data: None (tables start empty)
-- Backward Compatibility: Full (all changes are additive)
-- ============================================================================

-- Commit transaction
COMMIT;
