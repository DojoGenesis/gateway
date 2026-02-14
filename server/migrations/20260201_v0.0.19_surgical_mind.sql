-- ============================================================================
-- Migration: v0.0.19 (The Surgical Mind)
-- ============================================================================
-- Migration Date: February 1, 2026
-- Migration Version: 20260201_v0.0.19
-- 
-- **REQUIRES: v0.0.18 as base version**
-- Base Version: v0.0.18 (Migration: 20260131_v0.0.17_and_v0.0.18)
-- Target Database: SQLite at .dojo/memory.db
--
-- This migration adds:
-- - Vector embeddings for semantic search (memories, memory_seeds)
-- - Context type support for multi-user foundations (private/group/public)
-- - Memory files table for hierarchical memory tracking
--
-- IMPORTANT NOTES:
-- - This migration is atomic (transaction-wrapped) and can be safely retried
-- - All ALTER TABLE operations are safe and preserve existing data
-- - Embeddings are optional (NULL by default) and generated asynchronously
-- - Context type defaults to 'private' for backward compatibility
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
VALUES ('20260201_v0.0.19', datetime('now'), 'v0.0.19: The Surgical Mind - Vector embeddings, context types, hierarchical memory');

-- ============================================================================
-- Vector Embeddings for Semantic Search
-- ============================================================================

-- Add embedding column to memories table
-- Stores 768-dimensional float32 vector as binary BLOB
-- NULL by default, populated asynchronously
ALTER TABLE memories ADD COLUMN embedding BLOB;

-- ============================================================================
-- Context Type for Multi-User Foundations
-- ============================================================================

-- Add context_type to memories table
-- Values: 'private' (default), 'group', 'public'
-- Enables privacy rules: non-private contexts only get Tier 1 data
-- CHECK constraint ensures only valid values are inserted
ALTER TABLE memories ADD COLUMN context_type TEXT DEFAULT 'private' CHECK(context_type IN ('private', 'group', 'public'));

-- Add context_type to memory_seeds table
ALTER TABLE memory_seeds ADD COLUMN context_type TEXT DEFAULT 'private' CHECK(context_type IN ('private', 'group', 'public'));

-- ============================================================================
-- Indexes for Context Type Filtering
-- ============================================================================

-- Index for efficient filtering by context type
CREATE INDEX IF NOT EXISTS idx_memories_context_type ON memories(context_type);
CREATE INDEX IF NOT EXISTS idx_memory_seeds_context_type ON memory_seeds(context_type);

-- ============================================================================
-- Memory Files Table (Hierarchical Memory)
-- ============================================================================

-- Memory files: Tracks file-based memories for hierarchical system
-- Tier 1: Raw daily notes (memory/YYYY-MM-DD.md)
-- Tier 2: Curated wisdom (MEMORY.md)
-- Tier 3: Compressed archive (memory/archive/YYYY-MM.jsonl.gz)
CREATE TABLE IF NOT EXISTS memory_files (
    id TEXT PRIMARY KEY,
    file_path TEXT NOT NULL UNIQUE,
    tier INTEGER NOT NULL CHECK(tier IN (1, 2, 3)),  -- 1 (raw), 2 (curated), 3 (archive)
    content TEXT NOT NULL,          -- Full file content
    embedding BLOB,                 -- Vector embedding for semantic search
    themes TEXT CHECK(themes IS NULL OR json_valid(themes)),  -- JSON array of extracted themes
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    archived_at DATETIME            -- NULL for active files, set when archived
);

-- ============================================================================
-- Indexes for Memory Files
-- ============================================================================

-- Index for tier-based filtering (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_memory_files_tier ON memory_files(tier);

-- Index for archived files (maintenance cycle queries)
CREATE INDEX IF NOT EXISTS idx_memory_files_archived ON memory_files(archived_at);

-- Index for file path lookups (memory_get tool)
CREATE INDEX IF NOT EXISTS idx_memory_files_path ON memory_files(file_path);

-- Composite index for tier + archived queries (common in search)
CREATE INDEX IF NOT EXISTS idx_memory_files_tier_archived ON memory_files(tier, archived_at);

-- ============================================================================
-- Migration Complete
-- ============================================================================
-- Columns Added:
--   memories: embedding BLOB, context_type TEXT
--   memory_seeds: context_type TEXT
-- Tables Created:
--   memory_files (with 4 indexes)
-- Indexes Created: 6 total
-- Backward Compatibility: Full (all changes are additive)
-- ============================================================================

-- Commit transaction
COMMIT;
