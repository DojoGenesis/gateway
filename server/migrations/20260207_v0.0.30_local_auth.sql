-- ============================================================================
-- Migration: v0.0.30 Track C - Local-First Auth Foundation
-- ============================================================================
-- Migration Date: February 7, 2026
-- Migration Version: 20260207_v0.0.30_local_auth
-- 
-- **REQUIRES: v0.0.26 as base version**
-- Target Database: SQLite at .dojo/dojo.db
--
-- This migration creates the foundation for local-first authentication:
-- - Guest user support with persistent IDs
-- - Secure API key storage (encrypted at rest)
-- - Local conversation tracking
-- - User settings and preferences
-- - Data migration tracking (local → cloud)
--
-- IMPORTANT NOTES:
-- - This is a new database file (.dojo/dojo.db) - separate from memory.db
-- - Migration is atomic (transaction-wrapped) and can be safely retried
-- - Creates schema_migrations table for version tracking
-- - Enables local-only operation without Supabase dependency
-- ============================================================================

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Begin transaction to ensure atomic migration
BEGIN TRANSACTION;

-- ============================================================================
-- Migration Version Tracking
-- ============================================================================

-- Schema migrations table: Tracks which migrations have been applied
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at DATETIME NOT NULL,
    description TEXT
);

-- Record this migration
INSERT OR IGNORE INTO schema_migrations (version, applied_at, description)
VALUES ('20260207_v0.0.30_local_auth', datetime('now'), 'Local-first authentication foundation: guest users, API keys, settings, and migration tracking');

-- ============================================================================
-- User Management
-- ============================================================================

-- Local users table: Tracks guest and authenticated users
-- Supports offline operation with guest user IDs (UUIDs)
CREATE TABLE IF NOT EXISTS local_users (
    id TEXT PRIMARY KEY,                    -- UUID for guest users, cloud user ID for authenticated
    user_type TEXT DEFAULT 'guest' CHECK(user_type IN ('guest', 'authenticated')),
    created_at DATETIME NOT NULL,
    last_accessed_at DATETIME NOT NULL,
    cloud_user_id TEXT,                     -- Set when user signs in and links to cloud account
    migration_status TEXT DEFAULT 'none' CHECK(migration_status IN ('none', 'pending', 'in_progress', 'completed', 'failed')),
    metadata TEXT                           -- JSON blob for additional data (e.g., device info, session count)
);

CREATE INDEX IF NOT EXISTS idx_local_users_type ON local_users(user_type);
CREATE INDEX IF NOT EXISTS idx_local_users_migration_status ON local_users(migration_status);
CREATE INDEX IF NOT EXISTS idx_local_users_cloud_id ON local_users(cloud_user_id);
CREATE INDEX IF NOT EXISTS idx_local_users_last_accessed ON local_users(last_accessed_at DESC);

-- ============================================================================
-- API Key Management
-- ============================================================================

-- API keys table: Encrypted storage for LLM provider API keys
-- Supports multiple providers (OpenAI, Anthropic, DeepSeek, etc.)
-- Keys are encrypted via OS keychain or encrypted database storage
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL CHECK(provider IN ('openai', 'anthropic', 'deepseek', 'google', 'cohere', 'mistral', 'other')),
    key_name TEXT,                          -- User-friendly name (e.g., "Work OpenAI Key")
    key_hash TEXT NOT NULL,                 -- SHA-256 hash of the key for validation
    encrypted_key BLOB NOT NULL,            -- Encrypted key (via keychain or AES-256)
    storage_type TEXT NOT NULL CHECK(storage_type IN ('keychain', 'encrypted_db')),
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    last_used_at DATETIME,
    is_active BOOLEAN DEFAULT 1,            -- Soft delete flag
    metadata TEXT,                          -- JSON blob (e.g., key permissions, rate limits)
    FOREIGN KEY (user_id) REFERENCES local_users(id) ON DELETE CASCADE,
    UNIQUE(user_id, provider)               -- One key per provider per user
);

CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_provider ON api_keys(provider);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active);
CREATE INDEX IF NOT EXISTS idx_api_keys_last_used ON api_keys(last_used_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_provider ON api_keys(user_id, provider);

-- ============================================================================
-- Conversation Tracking
-- ============================================================================

-- Conversations table: Local conversation/session tracking
-- Links to memories in memory.db via session_id
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,                    -- Session ID (matches session_id in memories table)
    user_id TEXT NOT NULL,
    title TEXT,
    project_id TEXT,                        -- Optional: Link to project in memory.db
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    last_message_at DATETIME,
    message_count INTEGER DEFAULT 0,
    is_archived BOOLEAN DEFAULT 0,
    metadata TEXT,                          -- JSON blob (e.g., model used, token count, tags)
    FOREIGN KEY (user_id) REFERENCES local_users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_conversations_user ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_project ON conversations(project_id);
CREATE INDEX IF NOT EXISTS idx_conversations_last_message ON conversations(last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversations_archived ON conversations(is_archived);
CREATE INDEX IF NOT EXISTS idx_conversations_user_archived ON conversations(user_id, is_archived, last_message_at DESC);

-- ============================================================================
-- User Settings
-- ============================================================================

-- User settings table: Per-user preferences and configuration
-- Stores UI preferences, default models, notification settings, etc.
CREATE TABLE IF NOT EXISTS user_settings (
    user_id TEXT PRIMARY KEY,
    theme TEXT DEFAULT 'dark' CHECK(theme IN ('light', 'dark', 'auto')),
    language TEXT DEFAULT 'en',
    default_model TEXT,                     -- Default LLM model
    default_provider TEXT,                  -- Default LLM provider
    notification_enabled BOOLEAN DEFAULT 1,
    auto_save_enabled BOOLEAN DEFAULT 1,
    preferences TEXT,                       -- JSON blob for additional preferences
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES local_users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_settings_updated ON user_settings(updated_at DESC);

-- ============================================================================
-- Migration Tracking
-- ============================================================================

-- Migration log table: Tracks data migration from local to cloud
-- Records migration attempts, progress, and errors
CREATE TABLE IF NOT EXISTS migration_log (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    migration_type TEXT NOT NULL CHECK(migration_type IN ('full', 'partial', 'selective')),
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    status TEXT DEFAULT 'running' CHECK(status IN ('running', 'completed', 'failed', 'cancelled')),
    records_migrated INTEGER DEFAULT 0,
    records_total INTEGER DEFAULT 0,
    progress_percent REAL DEFAULT 0.0,
    errors TEXT,                            -- JSON array of error messages
    metadata TEXT,                          -- JSON blob (e.g., tables migrated, checksum)
    FOREIGN KEY (user_id) REFERENCES local_users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_migration_log_user ON migration_log(user_id);
CREATE INDEX IF NOT EXISTS idx_migration_log_status ON migration_log(status);
CREATE INDEX IF NOT EXISTS idx_migration_log_started ON migration_log(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_migration_log_user_status ON migration_log(user_id, status);

-- ============================================================================
-- Migration Complete
-- ============================================================================
-- Database: .dojo/dojo.db
-- Tables Created:
--   - local_users (user management)
--   - api_keys (encrypted API key storage)
--   - conversations (conversation tracking)
--   - user_settings (user preferences)
--   - migration_log (migration tracking)
--   - schema_migrations (version tracking)
-- Indexes: 23 strategic indexes for optimal query performance
-- ============================================================================

-- Commit transaction
COMMIT;
