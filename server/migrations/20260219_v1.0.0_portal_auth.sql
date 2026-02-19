-- Migration: v1.0.0 Portal Auth — extend local_users for portal credential storage
-- Date: 2026-02-19
-- Requires: v0.0.30 (local_users table must exist)
-- Database: .dojo/dojo.db (SQLite)

PRAGMA foreign_keys = ON;
BEGIN TRANSACTION;

INSERT OR IGNORE INTO schema_migrations (version, applied_at, description)
VALUES ('20260219_v1.0.0_portal_auth', datetime('now'),
        'Portal auth: add email, password_hash, display_name to local_users');

-- Extend local_users with portal credential fields
-- ALTER TABLE in SQLite only supports ADD COLUMN
ALTER TABLE local_users ADD COLUMN email TEXT;
ALTER TABLE local_users ADD COLUMN password_hash TEXT;
ALTER TABLE local_users ADD COLUMN display_name TEXT;

-- Unique constraint on email (non-null emails must be unique)
-- SQLite does not support ADD CONSTRAINT on existing tables.
-- Enforce uniqueness via unique index with WHERE filter.
CREATE UNIQUE INDEX IF NOT EXISTS idx_local_users_email
    ON local_users(email) WHERE email IS NOT NULL;

COMMIT;
