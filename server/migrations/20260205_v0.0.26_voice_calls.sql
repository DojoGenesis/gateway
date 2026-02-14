-- ============================================================================
-- Migration: v0.0.26 (The Proactive Intelligence System - Voice Calls)
-- ============================================================================
-- Migration Date: February 5, 2026
-- Migration Version: 20260205_v0.0.26_voice_calls
-- 
-- **REQUIRES: v0.0.24 as base version**
-- Base Version: v0.0.24 (Migration: 20260204_v0.0.24)
-- Target Database: SQLite at .dojo/memory.db
--
-- This migration adds:
-- - Voice Calls: Track bidirectional voice calls with ElevenLabs integration
-- - Voice Call Actions: Extract and track actionable items from call transcripts
--
-- IMPORTANT NOTES:
-- - This migration is atomic (transaction-wrapped) and can be safely retried
-- - All operations preserve existing data
-- - New tables support voice context injection and post-call processing
-- - Integrates with ElevenLabs Conversational AI and Twilio
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
VALUES ('20260205_v0.0.26_voice_calls', datetime('now'), 'v0.0.26: The Proactive Intelligence System - Voice Calls, Voice Call Actions');

-- ============================================================================
-- Voice Calls: Call Tracking
-- ============================================================================

-- Tracks all voice calls (inbound and outbound) with context injection
-- Direction: 'outbound' (user-initiated) or 'inbound' (external call)
-- Status: 'initiated', 'connected', 'completed', 'failed'
CREATE TABLE IF NOT EXISTS voice_calls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL,
    phone_number TEXT NOT NULL,
    direction TEXT NOT NULL CHECK(direction IN ('outbound', 'inbound')),
    status TEXT NOT NULL CHECK(status IN ('initiated', 'connected', 'completed', 'failed')),
    duration_seconds INTEGER,
    transcript TEXT,
    context_injected TEXT,              -- JSON blob: context provided to ElevenLabs
    actions_extracted TEXT,             -- JSON blob: actions extracted from transcript
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    completed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_voice_calls_user_id ON voice_calls(user_id);
CREATE INDEX IF NOT EXISTS idx_voice_calls_session_id ON voice_calls(session_id);
CREATE INDEX IF NOT EXISTS idx_voice_calls_created_at ON voice_calls(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_voice_calls_status ON voice_calls(status);
CREATE INDEX IF NOT EXISTS idx_voice_calls_direction ON voice_calls(direction);

-- ============================================================================
-- Voice Call Actions: Extracted Actions
-- ============================================================================

-- Stores actionable items extracted from voice call transcripts
-- Action types: 'create_goal', 'send_message', 'schedule_reminder', 'update_project'
-- Status: 'pending', 'completed', 'failed'
CREATE TABLE IF NOT EXISTS voice_call_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    call_session_id TEXT NOT NULL,
    action_type TEXT NOT NULL CHECK(action_type IN ('create_goal', 'send_message', 'schedule_reminder', 'update_project', 'other')),
    action_data TEXT NOT NULL,          -- JSON blob: action-specific data
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'completed', 'failed')),
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    completed_at DATETIME,
    FOREIGN KEY (call_session_id) REFERENCES voice_calls(session_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_voice_call_actions_session_id ON voice_call_actions(call_session_id);
CREATE INDEX IF NOT EXISTS idx_voice_call_actions_status ON voice_call_actions(status);
CREATE INDEX IF NOT EXISTS idx_voice_call_actions_type ON voice_call_actions(action_type);
CREATE INDEX IF NOT EXISTS idx_voice_call_actions_created ON voice_call_actions(created_at DESC);

-- ============================================================================
-- Migration Complete
-- ============================================================================
-- Tables Created: 2 total
--   - voice_calls (5 indexes)
--   - voice_call_actions (4 indexes)
-- Indexes Created: 9 total
-- Default Data: None (tables start empty)
-- Backward Compatibility: Full (all changes are additive)
-- ============================================================================

-- Commit transaction
COMMIT;
