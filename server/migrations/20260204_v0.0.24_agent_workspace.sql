-- ============================================================================
-- Migration: v0.0.24 (The Agent Workspace)
-- ============================================================================
-- Migration Date: February 4, 2026
-- Migration Version: 20260204_v0.0.24
-- 
-- **REQUIRES: v0.0.23 as base version**
-- Base Version: v0.0.23 (Migration: 20260203000000_v0.0.23)
-- Target Database: SQLite at .dojo/memory.db
--
-- This migration adds:
-- - Workspaces: Top-level containers for organizing projects
-- - Workspace-Project Relationships: Many-to-many linking
-- - Agent Registry: Central catalog of all available agents
-- - Agent Capabilities: Detailed agent capability tracking
--
-- IMPORTANT NOTES:
-- - This migration is atomic (transaction-wrapped) and can be safely retried
-- - All operations preserve existing data
-- - New tables support workspace organization and agent discovery
-- - Default behavior: No workspaces created (opt-in via UI)
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
VALUES ('20260204_v0.0.24', datetime('now'), 'v0.0.24: The Agent Workspace - Workspaces, Agent Registry');

-- ============================================================================
-- Workspaces: Project Organization
-- ============================================================================

-- A top-level container for organizing projects
-- Users can create multiple workspaces to organize their work
-- Each workspace can contain multiple projects (many-to-many)
CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE(user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_workspaces_user ON workspaces(user_id);
CREATE INDEX IF NOT EXISTS idx_workspaces_created ON workspaces(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workspaces_updated ON workspaces(updated_at DESC);

-- ============================================================================
-- Workspace-Project Relationships: Many-to-Many
-- ============================================================================

-- Join table linking projects to workspaces
-- A project can belong to multiple workspaces
-- A workspace can contain multiple projects
CREATE TABLE IF NOT EXISTS workspace_projects (
    workspace_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    added_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (workspace_id, project_id),
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_workspace_projects_workspace ON workspace_projects(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_projects_project ON workspace_projects(project_id);
CREATE INDEX IF NOT EXISTS idx_workspace_projects_added ON workspace_projects(added_at DESC);

-- ============================================================================
-- Agent Registry: Central Catalog
-- ============================================================================

-- A registry for all agents in the system
-- Agent types: primary (main assistant), specialist (domain expert), utility (helper)
-- Agent status: active (available), inactive (disabled), experimental (testing)
CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('primary', 'specialist', 'utility')),
    status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'inactive', 'experimental')),
    model_name TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_agents_type ON agents(type);
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name);
CREATE INDEX IF NOT EXISTS idx_agents_created ON agents(created_at DESC);

-- ============================================================================
-- Agent Capabilities: Detailed Tracking
-- ============================================================================

-- Describes the specific capabilities of each agent
-- Capability types: tool (can use a tool), skill (has a skill), model (uses a model)
-- Examples: 'file_write', 'web_search', 'gpt-4.1-mini'
CREATE TABLE IF NOT EXISTS agent_capabilities (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    capability_type TEXT NOT NULL CHECK(capability_type IN ('tool', 'skill', 'model')),
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    UNIQUE(agent_id, capability_type, name)
);

CREATE INDEX IF NOT EXISTS idx_agent_capabilities_agent ON agent_capabilities(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_capabilities_type ON agent_capabilities(capability_type);
CREATE INDEX IF NOT EXISTS idx_agent_capabilities_name ON agent_capabilities(name);

-- ============================================================================
-- Migration Complete
-- ============================================================================
-- Tables Created: 4 total
--   - workspaces (3 indexes)
--   - workspace_projects (3 indexes)
--   - agents (4 indexes)
--   - agent_capabilities (3 indexes)
-- Indexes Created: 13 total
-- Default Data: None (tables start empty, will be seeded separately)
-- Backward Compatibility: Full (all changes are additive)
-- Note: projects table already exists from v0.0.20+
-- ============================================================================

-- Commit transaction
COMMIT;
