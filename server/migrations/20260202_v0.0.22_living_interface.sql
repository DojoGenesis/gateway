-- ============================================================================
-- Migration: v0.0.22 (The Living Interface)
-- ============================================================================
-- Migration Date: February 2, 2026
-- Migration Version: 20260202_v0.0.22
-- 
-- **REQUIRES: v0.0.20 as base version**
-- Base Version: v0.0.20 (Migration: 20260201_v0.0.20)
-- Target Database: SQLite at .dojo/memory.db
--
-- This migration adds:
-- - Adaptive Layout Engine: Layout transitions tracking
-- - Ambient Intelligence: User preferences for notifications
-- - Conversational Workspace: Command history tracking
--
-- IMPORTANT NOTES:
-- - This migration is atomic (transaction-wrapped) and can be safely retried
-- - All operations preserve existing data
-- - New tables support adaptive UI, notifications, and command control
-- - Default behavior: All notifications enabled (opt-out via preferences)
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
VALUES ('20260202_v0.0.22', datetime('now'), 'v0.0.22: The Living Interface - Adaptive Layout, Ambient Intelligence, Conversational Workspace');

-- ============================================================================
-- Adaptive Layout Engine: Layout Transitions
-- ============================================================================

-- Tracks layout mode changes and their triggers for learning user preferences
-- Modes: Focus, Create, Review, Plan, Debug, Explore
-- Trigger types: automatic (context-based), manual (user-initiated), command (from chat)
CREATE TABLE IF NOT EXISTS layout_transitions (
    id TEXT PRIMARY KEY,
    from_mode TEXT NOT NULL CHECK(from_mode IN ('Focus', 'Create', 'Review', 'Plan', 'Debug', 'Explore')),
    to_mode TEXT NOT NULL CHECK(to_mode IN ('Focus', 'Create', 'Review', 'Plan', 'Debug', 'Explore')),
    trigger_type TEXT NOT NULL CHECK(trigger_type IN ('automatic', 'manual', 'command')),
    trigger_context TEXT,          -- JSON blob: event details, user activity, etc.
    session_id TEXT,               -- Link to conversation session
    project_id TEXT,               -- Link to active project
    created_at DATETIME NOT NULL,
    duration_ms INTEGER,           -- Time spent in from_mode before transition
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_layout_transitions_session ON layout_transitions(session_id);
CREATE INDEX IF NOT EXISTS idx_layout_transitions_project ON layout_transitions(project_id);
CREATE INDEX IF NOT EXISTS idx_layout_transitions_created ON layout_transitions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_layout_transitions_trigger ON layout_transitions(trigger_type);
CREATE INDEX IF NOT EXISTS idx_layout_transitions_modes ON layout_transitions(from_mode, to_mode);

-- ============================================================================
-- Ambient Intelligence: User Preferences
-- ============================================================================

-- Stores user preferences for notifications and quiet hours
-- Single-row table (id = 'default') for global settings
-- Notification types: memory_compression, seeds_extracted, goal_milestone, 
--                     project_dormancy, common_patterns
CREATE TABLE IF NOT EXISTS user_preferences (
    id TEXT PRIMARY KEY DEFAULT 'default',
    -- Notification Settings (TRUE = enabled, FALSE = disabled)
    notify_memory_compression BOOLEAN DEFAULT TRUE,
    notify_seeds_extracted BOOLEAN DEFAULT TRUE,
    notify_goal_milestone BOOLEAN DEFAULT TRUE,
    notify_project_dormancy BOOLEAN DEFAULT TRUE,
    notify_common_patterns BOOLEAN DEFAULT TRUE,
    
    -- Quiet Hours Configuration (JSON array of time ranges)
    -- Format: [{"day": 0, "start_hour": 22, "end_hour": 8}, ...]
    -- day: 0 (Sunday) through 6 (Saturday)
    -- start_hour, end_hour: 0-23 (24-hour format)
    quiet_hours TEXT CHECK(quiet_hours IS NULL OR json_valid(quiet_hours)),
    
    -- Rate Limiting (minutes between notifications of same type)
    notification_rate_limit INTEGER DEFAULT 5 CHECK(notification_rate_limit >= 1),
    
    -- UI Preferences
    layout_auto_switch BOOLEAN DEFAULT TRUE,  -- Enable automatic layout switching
    layout_animation BOOLEAN DEFAULT TRUE,    -- Enable layout transition animations
    
    -- Metadata
    updated_at DATETIME NOT NULL,
    CHECK(id = 'default')  -- Enforce single-row constraint
);

-- Insert default preferences row
INSERT OR IGNORE INTO user_preferences (id, updated_at)
VALUES ('default', datetime('now'));

-- ============================================================================
-- Conversational Workspace: Command History
-- ============================================================================

-- Tracks command usage for analytics and suggestions
-- Command types match intent names from IntentParser:
-- show_goals, open_artifact, set_layout, search_memory, show_trace, etc.
CREATE TABLE IF NOT EXISTS command_history (
    id TEXT PRIMARY KEY,
    command_type TEXT NOT NULL,    -- Intent name (e.g., 'show_goals', 'open_artifact')
    command_text TEXT NOT NULL,    -- Original user input
    parameters TEXT,               -- JSON blob: extracted parameters
    session_id TEXT,               -- Link to conversation session
    project_id TEXT,               -- Link to active project
    confidence REAL CHECK(confidence >= 0.0 AND confidence <= 1.0),
    success BOOLEAN,               -- Did command execute successfully?
    error_message TEXT,            -- Error details if success = FALSE
    execution_time_ms INTEGER,     -- Command execution time
    created_at DATETIME NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_command_history_session ON command_history(session_id);
CREATE INDEX IF NOT EXISTS idx_command_history_project ON command_history(project_id);
CREATE INDEX IF NOT EXISTS idx_command_history_created ON command_history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_command_history_type ON command_history(command_type);
CREATE INDEX IF NOT EXISTS idx_command_history_success ON command_history(success);
CREATE INDEX IF NOT EXISTS idx_command_history_project_type ON command_history(project_id, command_type);

-- ============================================================================
-- Migration Complete
-- ============================================================================
-- Tables Created: 3 total
--   - layout_transitions (5 indexes)
--   - user_preferences (0 indexes, single-row)
--   - command_history (6 indexes)
-- Indexes Created: 11 total
-- Default Data: user_preferences row with all notifications enabled
-- Backward Compatibility: Full (all changes are additive)
-- ============================================================================

-- Commit transaction
COMMIT;
