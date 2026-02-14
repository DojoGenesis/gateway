-- ============================================================================
-- Rollback Migration: v0.0.24 to v0.0.23
-- ============================================================================
-- Migration Date: February 4, 2026
-- Purpose: Rollback database from v0.0.24 (The Agent Workspace) to v0.0.23
-- Target: SQLite database at .dojo/memory.db
--
-- WARNING: This script will:
-- - DROP 4 tables (workspaces, workspace_projects, agents, agent_capabilities)
-- - DELETE all workspace data, workspace-project relationships, and agent registry data
--
-- BACKUP YOUR DATABASE BEFORE RUNNING THIS SCRIPT!
--
-- ⚠️  DATA LOSS WARNING:
-- This rollback will permanently delete:
-- - All workspaces and their configurations
-- - All workspace-project relationships (projects themselves are preserved)
-- - All agent registry data and capability information
--
-- ⚠️  PERFORMANCE WARNING - LARGE DATABASES:
-- Estimated rollback time:
-- - Small DB (<100 workspaces): <1 second
-- - Medium DB (100-1,000 workspaces): 1-3 seconds
-- - Large DB (1,000-10,000 workspaces): 3-10 seconds
--
-- For production databases with >1,000 workspaces:
-- 1. Schedule rollback during maintenance window
-- 2. Stop application to prevent write conflicts
-- 3. Test rollback on a backup copy first
-- ============================================================================

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Begin transaction to ensure atomic rollback
BEGIN TRANSACTION;

-- ============================================================================
-- Drop v0.0.24 Indexes (for cleaner rollback)
-- ============================================================================

-- Agent Capabilities indexes
DROP INDEX IF EXISTS idx_agent_capabilities_name;
DROP INDEX IF EXISTS idx_agent_capabilities_type;
DROP INDEX IF EXISTS idx_agent_capabilities_agent;

-- Agents indexes
DROP INDEX IF EXISTS idx_agents_created;
DROP INDEX IF EXISTS idx_agents_name;
DROP INDEX IF EXISTS idx_agents_status;
DROP INDEX IF EXISTS idx_agents_type;

-- Workspace Projects indexes
DROP INDEX IF EXISTS idx_workspace_projects_added;
DROP INDEX IF EXISTS idx_workspace_projects_project;
DROP INDEX IF EXISTS idx_workspace_projects_workspace;

-- Workspaces indexes
DROP INDEX IF EXISTS idx_workspaces_updated;
DROP INDEX IF EXISTS idx_workspaces_created;
DROP INDEX IF EXISTS idx_workspaces_user;

-- ============================================================================
-- Drop v0.0.24 Tables (in reverse order of dependencies)
-- ============================================================================

-- Drop tables with foreign key dependencies first
DROP TABLE IF EXISTS agent_capabilities;
DROP TABLE IF EXISTS workspace_projects;

-- Drop remaining tables
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS workspaces;

-- ============================================================================
-- Remove Migration Version Record
-- ============================================================================

-- Delete the migration version record
DELETE FROM schema_migrations WHERE version = '20260204_v0.0.24';

-- ============================================================================
-- Rollback Complete
-- ============================================================================
-- Tables Dropped: 4
--   - workspaces
--   - workspace_projects
--   - agents
--   - agent_capabilities
-- Indexes Dropped: 13
-- Data Preserved: Projects table and all existing v0.0.23 data remain intact
-- Migration Record: Removed from schema_migrations
-- Database State: Reverted to v0.0.23
-- ============================================================================

-- Commit transaction
COMMIT;
