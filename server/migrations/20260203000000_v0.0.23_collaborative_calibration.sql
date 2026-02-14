-- ============================================================================
-- Migration: v0.0.23 (The Collaborative Calibration)
-- ============================================================================
-- Migration Date: February 3, 2026
-- Migration Version: 20260203000000_v0.0.23
-- 
-- **REQUIRES: v0.0.22 as base version**
-- Base Version: v0.0.22 (Migration: 20260202_v0.0.22)
-- Target Database: SQLite at .dojo/memory.db
--
-- This migration adds:
-- - Feedback Engine: User feedback tracking
-- - Personalization Layer: Learned preferences and calibration events
-- - Collaborative Memory: User-editable memory seeds
-- - Judgment Layer Support: Decision tracking for learning
--
-- IMPORTANT NOTES:
-- - This migration is atomic (transaction-wrapped) and can be safely retried
-- - All operations preserve existing data
-- - New tables support feedback collection, preference learning, and memory collaboration
-- - Default behavior: Calibration disabled (opt-in via feature flag)
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
VALUES ('20260203000000_v0.0.23', datetime('now'), 'v0.0.23: The Collaborative Calibration - Feedback Engine, Personalization Layer, Collaborative Memory');

-- ============================================================================
-- Feedback Engine: User Feedback
-- ============================================================================

-- Stores explicit user feedback on AI actions (suggestions, messages, tool runs)
-- Rating types: helpful, not_helpful, intrusive, inaccurate
-- Processed flag indicates if calibration engine has incorporated this feedback
CREATE TABLE IF NOT EXISTS user_feedback (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    source_id TEXT NOT NULL,           -- ID of the suggestion, message, or tool run
    source_type TEXT NOT NULL CHECK(source_type IN ('suggestion', 'message', 'tool_run')),
    rating TEXT NOT NULL CHECK(rating IN ('helpful', 'not_helpful', 'intrusive', 'inaccurate')),
    comment TEXT CHECK(length(comment) <= 1000),  -- Optional user comment (max 1000 chars)
    processed BOOLEAN DEFAULT FALSE,   -- Has calibration engine processed this?
    created_at DATETIME NOT NULL,
    processed_at DATETIME,
    UNIQUE(user_id, source_id, source_type, created_at)  -- Prevent duplicate feedback
);

CREATE INDEX IF NOT EXISTS idx_user_feedback_source ON user_feedback(source_id, source_type);
CREATE INDEX IF NOT EXISTS idx_user_feedback_user ON user_feedback(user_id);
CREATE INDEX IF NOT EXISTS idx_user_feedback_processed ON user_feedback(processed, created_at);
CREATE INDEX IF NOT EXISTS idx_user_feedback_rating ON user_feedback(rating);
CREATE INDEX IF NOT EXISTS idx_user_feedback_created ON user_feedback(created_at DESC);

-- ============================================================================
-- Personalization Layer: Calibration Preferences
-- ============================================================================

-- Stores learned user preferences (separate from notification preferences in user_preferences)
-- Preference keys: suggestion.frequency, suggestion.intrusiveness, tone.formality, etc.
-- Confidence score increases as more feedback confirms the preference
CREATE TABLE IF NOT EXISTS calibration_preferences (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    preference_key TEXT NOT NULL,      -- e.g., 'suggestion.frequency', 'tone.formality'
    value TEXT NOT NULL,               -- Preference value (string for flexibility)
    value_type TEXT DEFAULT 'string' CHECK(value_type IN ('string', 'float', 'bool', 'json')),
    confidence REAL DEFAULT 0.5 CHECK(confidence >= 0.0 AND confidence <= 1.0),
    feedback_count INTEGER DEFAULT 0,  -- Number of feedback entries that informed this
    last_updated DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    UNIQUE(user_id, preference_key)
);

CREATE INDEX IF NOT EXISTS idx_calibration_preferences_user ON calibration_preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_calibration_preferences_key ON calibration_preferences(preference_key);
CREATE INDEX IF NOT EXISTS idx_calibration_preferences_confidence ON calibration_preferences(confidence DESC);
CREATE INDEX IF NOT EXISTS idx_calibration_preferences_updated ON calibration_preferences(last_updated DESC);

-- ============================================================================
-- Personalization Layer: Calibration Events
-- ============================================================================

-- Audit log of calibration adjustments for transparency and debugging
-- Event types: preference_created, preference_updated, preference_reset, feedback_processed
CREATE TABLE IF NOT EXISTS calibration_events (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    event_type TEXT NOT NULL CHECK(event_type IN ('preference_created', 'preference_updated', 'preference_reset', 'feedback_processed')),
    preference_key TEXT,               -- Which preference was affected (if applicable)
    old_value TEXT,                    -- Previous value (for updates)
    new_value TEXT,                    -- New value
    old_confidence REAL,               -- Previous confidence (for updates)
    new_confidence REAL,               -- New confidence
    feedback_ids TEXT,                 -- JSON array of feedback IDs that triggered this event
    metadata TEXT,                     -- JSON blob for additional context
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_calibration_events_user ON calibration_events(user_id);
CREATE INDEX IF NOT EXISTS idx_calibration_events_type ON calibration_events(event_type);
CREATE INDEX IF NOT EXISTS idx_calibration_events_key ON calibration_events(preference_key);
CREATE INDEX IF NOT EXISTS idx_calibration_events_created ON calibration_events(created_at DESC);

-- ============================================================================
-- Collaborative Memory: Memory Seeds
-- ============================================================================

-- Memory seeds for collaborative memory feature
-- Source types: system (AI-generated), user (user-created), calibrated (AI-adjusted from feedback)
-- User-editable seeds can be modified through the UI
CREATE TABLE IF NOT EXISTS memory_seeds (
    id TEXT PRIMARY KEY,
    project_id TEXT,                   -- NULL for global seeds
    content TEXT NOT NULL,
    seed_type TEXT NOT NULL,           -- Type of seed (e.g., 'pattern', 'preference', 'knowledge')
    source TEXT DEFAULT 'system' CHECK(source IN ('system', 'user', 'calibrated')),
    user_editable BOOLEAN DEFAULT FALSE,
    confidence REAL DEFAULT 1.0 CHECK(confidence >= 0.0 AND confidence <= 1.0),
    usage_count INTEGER DEFAULT 0,     -- How many times this seed has been used
    last_used_at DATETIME,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,               -- Soft delete support
    created_by TEXT,                   -- User ID who created (for user-created seeds)
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_memory_seeds_project ON memory_seeds(project_id);
CREATE INDEX IF NOT EXISTS idx_memory_seeds_type ON memory_seeds(seed_type);
CREATE INDEX IF NOT EXISTS idx_memory_seeds_source ON memory_seeds(source);
CREATE INDEX IF NOT EXISTS idx_memory_seeds_editable ON memory_seeds(user_editable);
CREATE INDEX IF NOT EXISTS idx_memory_seeds_confidence ON memory_seeds(confidence DESC);
CREATE INDEX IF NOT EXISTS idx_memory_seeds_usage ON memory_seeds(usage_count DESC);
CREATE INDEX IF NOT EXISTS idx_memory_seeds_created ON memory_seeds(created_at DESC);

-- ============================================================================
-- Judgment Layer: Decision Tracking
-- ============================================================================

-- Tracks judgment layer decisions for learning and calibration
-- Decision types: show_suggestion, suppress_suggestion, adjust_confidence
-- Allows calibration engine to learn from successful/unsuccessful decisions
CREATE TABLE IF NOT EXISTS judgment_decisions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    decision_type TEXT NOT NULL CHECK(decision_type IN ('show_suggestion', 'suppress_suggestion', 'adjust_confidence', 'auto_dismiss')),
    suggestion_id TEXT,                -- ID of suggestion this decision affected
    context TEXT,                      -- JSON blob: factors that influenced decision
    confidence_before REAL,            -- Confidence score before decision
    confidence_after REAL,             -- Confidence score after decision
    preference_keys_used TEXT,         -- JSON array: which preferences influenced this
    outcome TEXT,                      -- Result: 'accepted', 'dismissed', 'ignored', 'timeout'
    created_at DATETIME NOT NULL,
    outcome_recorded_at DATETIME,
    FOREIGN KEY (suggestion_id) REFERENCES project_suggestions(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_judgment_decisions_user ON judgment_decisions(user_id);
CREATE INDEX IF NOT EXISTS idx_judgment_decisions_type ON judgment_decisions(decision_type);
CREATE INDEX IF NOT EXISTS idx_judgment_decisions_suggestion ON judgment_decisions(suggestion_id);
CREATE INDEX IF NOT EXISTS idx_judgment_decisions_outcome ON judgment_decisions(outcome);
CREATE INDEX IF NOT EXISTS idx_judgment_decisions_created ON judgment_decisions(created_at DESC);

-- ============================================================================
-- Migration Complete
-- ============================================================================
-- Tables Created: 5 total
--   - user_feedback (5 indexes)
--   - calibration_preferences (4 indexes)
--   - calibration_events (4 indexes)
--   - memory_seeds (7 indexes)
--   - judgment_decisions (5 indexes)
-- Indexes Created: 25 total
-- Default Data: None (tables start empty)
-- Backward Compatibility: Full (all changes are additive)
-- Note: project_suggestions table already exists from v0.0.20
-- ============================================================================

-- Commit transaction
COMMIT;
