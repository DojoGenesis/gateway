-- ============================================================================
-- Unified Migration: v0.0.17 (The Thoughtful System) + v0.0.18 (The Creative Studio)
-- ============================================================================
-- Migration Date: January 31, 2026
-- Migration Version: 20260131_v0.0.17_and_v0.0.18
-- 
-- **REQUIRES: v0.0.16 as base version**
-- Base Version: v0.0.16 (Tag: v0.0.16, Commit: c1824bd)
-- Target Database: SQLite at .dojo/memory.db
--
-- This migration adds database schemas for both parallel development tracks:
-- - v0.0.17: Memory Garden & Trace Viewer
-- - v0.0.18: Project Workspace, Artifact Engine & Visual Canvas
--
-- IMPORTANT NOTES:
-- - This is a unified migration to avoid conflicts between parallel tracks
-- - Requires v0.0.16 baseline with 'memories' table already present
-- - Migration is atomic (transaction-wrapped) and can be safely retried
-- - Creates schema_migrations table for version tracking
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
INSERT INTO schema_migrations (version, applied_at, description)
VALUES ('20260131_v0.0.17_and_v0.0.18', datetime('now'), 'Unified migration for v0.0.17 (Memory Garden, Trace Viewer) and v0.0.18 (Project Workspace, Artifact Engine)');

-- ============================================================================
-- v0.0.18: Project Workspace
-- ============================================================================

-- Projects table: Organizational units for grouping work
-- Each project is a scoped workspace with its own artifacts, memory, and context
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    template_id TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    last_accessed_at DATETIME NOT NULL,
    settings TEXT,              -- JSON blob for project-specific settings
    metadata TEXT,              -- JSON blob for additional metadata
    status TEXT DEFAULT 'active' -- active, archived, deleted
);

CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_last_accessed ON projects(last_accessed_at DESC);
CREATE INDEX IF NOT EXISTS idx_projects_updated ON projects(updated_at DESC);

-- Project templates: Pre-defined structures for common workflows
-- System templates include: Research Report, Software Design, Data Analysis, etc.
CREATE TABLE IF NOT EXISTS project_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    category TEXT,              -- e.g., "research", "development", "design"
    structure TEXT NOT NULL,    -- JSON blob defining directory structure
    default_settings TEXT,      -- JSON blob for default project settings
    is_system BOOLEAN DEFAULT FALSE,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_templates_category ON project_templates(category);

-- ============================================================================
-- v0.0.18: Artifact Engine
-- ============================================================================

-- Artifacts table: Persistent, version-controlled outputs
-- Five types: document, diagram, code_project, data_viz, image
CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    session_id TEXT,            -- Link to conversation that created it
    type TEXT NOT NULL,         -- document, diagram, code_project, data_viz, image
    name TEXT NOT NULL,
    description TEXT,
    latest_version INTEGER DEFAULT 1,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    metadata TEXT,              -- JSON blob: file_size, dimensions, etc.
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_artifacts_project ON artifacts(project_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_type ON artifacts(type);
CREATE INDEX IF NOT EXISTS idx_artifacts_updated ON artifacts(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_artifacts_project_type ON artifacts(project_id, type);

-- Artifact versions table: Version history with diffs
-- Provides Git-like versioning for all artifacts
CREATE TABLE IF NOT EXISTS artifact_versions (
    id TEXT PRIMARY KEY,
    artifact_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    content TEXT NOT NULL,      -- Full content for this version
    diff TEXT,                  -- JSON diff from previous version
    commit_message TEXT,
    created_at DATETIME NOT NULL,
    created_by TEXT,            -- User ID or "agent"
    metadata TEXT,              -- JSON blob: tokens_used, generation_time, etc.
    FOREIGN KEY (artifact_id) REFERENCES artifacts(id) ON DELETE CASCADE,
    UNIQUE(artifact_id, version)
);

CREATE INDEX IF NOT EXISTS idx_versions_artifact ON artifact_versions(artifact_id, version DESC);
CREATE INDEX IF NOT EXISTS idx_versions_created ON artifact_versions(created_at DESC);

-- Project files table: Files uploaded or generated for a project
-- Tracks all files in the ~/DojoProjects/{project_name}/ directory
CREATE TABLE IF NOT EXISTS project_files (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    file_path TEXT NOT NULL,    -- Path within project directory
    file_size INTEGER,
    mime_type TEXT,
    uploaded_at DATETIME NOT NULL,
    metadata TEXT,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_project_files_project ON project_files(project_id);
CREATE INDEX IF NOT EXISTS idx_project_files_uploaded ON project_files(uploaded_at DESC);

-- ============================================================================
-- v0.0.17: Memory Garden
-- ============================================================================

-- Memory Garden: Semantic compression and seed extraction
-- This table stores compressed memory representations (seeds)
CREATE TABLE IF NOT EXISTS memory_seeds (
    id TEXT PRIMARY KEY,
    project_id TEXT,            -- Optional: Link seed to a project
    type TEXT NOT NULL,         -- fact, pattern, preference, context
    content TEXT NOT NULL,      -- The seed content (compressed knowledge)
    embedding BLOB,             -- Vector embedding for semantic search
    source_memory_ids TEXT,     -- JSON array of source memory IDs
    confidence REAL DEFAULT 1.0,
    tier INTEGER DEFAULT 3,     -- Memory tier: 1 (working), 2 (episodic), 3 (semantic)
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    last_accessed_at DATETIME NOT NULL,
    access_count INTEGER DEFAULT 0,
    metadata TEXT               -- JSON blob for additional metadata
);

CREATE INDEX IF NOT EXISTS idx_seeds_project ON memory_seeds(project_id);
CREATE INDEX IF NOT EXISTS idx_seeds_type ON memory_seeds(type);
CREATE INDEX IF NOT EXISTS idx_seeds_tier ON memory_seeds(tier);
CREATE INDEX IF NOT EXISTS idx_seeds_confidence ON memory_seeds(confidence DESC);
CREATE INDEX IF NOT EXISTS idx_seeds_last_accessed ON memory_seeds(last_accessed_at DESC);

-- ============================================================================
-- v0.0.17: Trace Viewer
-- ============================================================================

-- Execution traces: Hierarchical trace spans for debugging and analysis
CREATE TABLE IF NOT EXISTS traces (
    id TEXT PRIMARY KEY,
    parent_id TEXT,             -- Parent trace for hierarchical structure
    session_id TEXT,            -- Link to conversation session
    trace_type TEXT NOT NULL,   -- query, tool_call, memory_retrieval, etc.
    name TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME,
    duration_ms INTEGER,
    status TEXT DEFAULT 'running', -- running, success, error
    metadata TEXT,              -- JSON blob: input, output, errors, etc.
    FOREIGN KEY (parent_id) REFERENCES traces(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_traces_session ON traces(session_id);
CREATE INDEX IF NOT EXISTS idx_traces_parent ON traces(parent_id);
CREATE INDEX IF NOT EXISTS idx_traces_started ON traces(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_traces_type ON traces(trace_type);
CREATE INDEX IF NOT EXISTS idx_traces_status ON traces(status);

-- ============================================================================
-- Integration: Link existing memories table to projects
-- ============================================================================

-- OPTIONAL: Add project_id to existing memories table
--
-- This ALTER TABLE is OPTIONAL and should ONLY be executed if:
-- 1. You want to enable project-scoped memory (feature from v0.0.17)
-- 2. The memories table exists in your database
-- 3. The project_id column does NOT already exist
--
-- To check if the column exists, run:
--   sqlite3 .dojo/memory.db "PRAGMA table_info(memories);" | grep project_id
--
-- If the command returns nothing, the column does not exist and you can safely run:
--   sqlite3 .dojo/memory.db "ALTER TABLE memories ADD COLUMN project_id TEXT;"
--   sqlite3 .dojo/memory.db "CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(project_id);"
--
-- NOTE: This is intentionally left commented to prevent unintended schema changes.
-- SQLite ALTER TABLE is safe and will NOT drop data, but this migration keeps it
-- optional to maintain compatibility with systems that may not need project scoping.

-- ALTER TABLE memories ADD COLUMN project_id TEXT;
-- CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(project_id);

-- ============================================================================
-- Migration Complete
-- ============================================================================
-- Tables Created:
--   v0.0.18: projects, project_templates, artifacts, artifact_versions, project_files
--   v0.0.17: memory_seeds, traces
--   System: schema_migrations (version tracking)
-- Integration: memories.project_id (requires manual ALTER if needed)
-- ============================================================================

-- Commit transaction
COMMIT;
