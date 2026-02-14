#!/bin/bash
#
# Migration Test Script: v0.0.24 (The Agent Workspace)
# ============================================================================
# Purpose: Test migration up/down for v0.0.24
# Requirements: SQLite3, Go 1.21+
# Usage: ./test_migration_v0.0.24.sh
# ============================================================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
TEST_DB="../.dojo/memory_test_v0.0.24.db"
MIGRATION_FILE="20260204_v0.0.24_agent_workspace.sql"
ROLLBACK_FILE="20260204_v0.0.24_rollback.sql"

echo "============================================================================"
echo "v0.0.24 Migration Test Suite"
echo "============================================================================"
echo ""

# Step 1: Create test database with v0.0.23 schema
echo -e "${YELLOW}Step 1: Creating test database with v0.0.23 schema...${NC}"
if [ -f "$TEST_DB" ]; then
    rm "$TEST_DB"
fi

# Create base schema (v0.0.23 tables needed for foreign keys)
sqlite3 "$TEST_DB" <<EOF
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at DATETIME NOT NULL,
    description TEXT
);

-- Create v0.0.20 projects table (needed for foreign keys)
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    last_accessed_at DATETIME,
    status TEXT DEFAULT 'active' CHECK(status IN ('active', 'archived', 'deleted')),
    suggestion_enabled BOOLEAN DEFAULT FALSE,
    suggestion_sensitivity REAL DEFAULT 0.7
);

-- Record v0.0.23 migration
INSERT INTO schema_migrations (version, applied_at, description)
VALUES ('20260203000000_v0.0.23', datetime('now'), 'v0.0.23: The Collaborative Calibration');

-- Insert test data
INSERT INTO projects (id, name, description, created_at, updated_at, status)
VALUES ('proj-test-001', 'Test Project 1', 'Test project for v0.0.24 migration', datetime('now'), datetime('now'), 'active');

INSERT INTO projects (id, name, description, created_at, updated_at, status)
VALUES ('proj-test-002', 'Test Project 2', 'Another test project', datetime('now'), datetime('now'), 'active');
EOF

echo -e "${GREEN}✓ Test database created with v0.0.23 schema${NC}"
echo ""

# Step 2: Apply v0.0.24 migration
echo -e "${YELLOW}Step 2: Applying v0.0.24 migration...${NC}"
if ! sqlite3 "$TEST_DB" < "$MIGRATION_FILE" 2>&1; then
    echo -e "${RED}✗ Migration failed${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo -e "${GREEN}✓ Migration applied successfully${NC}"
echo ""

# Step 3: Verify v0.0.24 tables exist
echo -e "${YELLOW}Step 3: Verifying v0.0.24 tables...${NC}"
TABLES=$(sqlite3 "$TEST_DB" "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;")

# Expected tables (v0.0.24 additions)
EXPECTED_TABLES=(
    "workspaces"
    "workspace_projects"
    "agents"
    "agent_capabilities"
)

for table in "${EXPECTED_TABLES[@]}"; do
    if echo "$TABLES" | grep -q "^${table}$"; then
        echo -e "${GREEN}  ✓ Table '${table}' exists${NC}"
    else
        echo -e "${RED}  ✗ Table '${table}' missing${NC}"
        rm "$TEST_DB"
        exit 1
    fi
done
echo ""

# Step 4: Verify indexes
echo -e "${YELLOW}Step 4: Verifying indexes...${NC}"
INDEXES=$(sqlite3 "$TEST_DB" "SELECT name FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%' ORDER BY name;")

EXPECTED_INDEXES=(
    "idx_workspaces_user"
    "idx_workspaces_created"
    "idx_workspaces_updated"
    "idx_workspace_projects_workspace"
    "idx_workspace_projects_project"
    "idx_workspace_projects_added"
    "idx_agents_type"
    "idx_agents_status"
    "idx_agents_name"
    "idx_agents_created"
    "idx_agent_capabilities_agent"
    "idx_agent_capabilities_type"
    "idx_agent_capabilities_name"
)

for index in "${EXPECTED_INDEXES[@]}"; do
    if echo "$INDEXES" | grep -q "^${index}$"; then
        echo -e "${GREEN}  ✓ Index '${index}' exists${NC}"
    else
        echo -e "${RED}  ✗ Index '${index}' missing${NC}"
        rm "$TEST_DB"
        exit 1
    fi
done
echo ""

# Step 5: Test inserting data into new tables
echo -e "${YELLOW}Step 5: Testing data insertion...${NC}"

# Test workspaces insert
sqlite3 "$TEST_DB" <<EOF
INSERT INTO workspaces (id, user_id, name, description, created_at, updated_at)
VALUES ('ws-001', 'user-001', 'My Workspace', 'A test workspace', datetime('now'), datetime('now'));
EOF
echo -e "${GREEN}  ✓ workspaces insert successful${NC}"

# Test workspace_projects insert
sqlite3 "$TEST_DB" <<EOF
INSERT INTO workspace_projects (workspace_id, project_id, added_at)
VALUES ('ws-001', 'proj-test-001', datetime('now'));
EOF
echo -e "${GREEN}  ✓ workspace_projects insert successful${NC}"

# Test agents insert
sqlite3 "$TEST_DB" <<EOF
INSERT INTO agents (id, name, description, type, status, model_name, created_at, updated_at)
VALUES ('agent-001', 'Primary Agent', 'Main AI assistant', 'primary', 'active', 'gpt-4.1-mini', datetime('now'), datetime('now'));
EOF
echo -e "${GREEN}  ✓ agents insert successful${NC}"

# Test agent_capabilities insert
sqlite3 "$TEST_DB" <<EOF
INSERT INTO agent_capabilities (id, agent_id, capability_type, name, description, created_at)
VALUES ('cap-001', 'agent-001', 'tool', 'file_write', 'Can write files to disk', datetime('now'));
EOF
echo -e "${GREEN}  ✓ agent_capabilities insert successful${NC}"
echo ""

# Step 6: Verify foreign key constraints work
echo -e "${YELLOW}Step 6: Testing foreign key constraints...${NC}"

# Test that workspace_projects FK to workspaces works (deletion cascade)
WP_COUNT_BEFORE=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM workspace_projects WHERE workspace_id = 'ws-001';")
sqlite3 "$TEST_DB" "PRAGMA foreign_keys = ON; DELETE FROM workspaces WHERE id = 'ws-001';"
WP_COUNT_AFTER=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM workspace_projects WHERE workspace_id = 'ws-001';")

if [ "$WP_COUNT_BEFORE" -gt 0 ] && [ "$WP_COUNT_AFTER" -eq 0 ]; then
    echo -e "${GREEN}  ✓ Foreign key cascade (workspace_projects → workspaces) works${NC}"
else
    echo -e "${RED}  ✗ Foreign key cascade failed (before: $WP_COUNT_BEFORE, after: $WP_COUNT_AFTER)${NC}"
    rm "$TEST_DB"
    exit 1
fi

# Test that agent_capabilities FK to agents works (deletion cascade)
CAP_COUNT_BEFORE=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM agent_capabilities WHERE agent_id = 'agent-001';")
sqlite3 "$TEST_DB" "PRAGMA foreign_keys = ON; DELETE FROM agents WHERE id = 'agent-001';"
CAP_COUNT_AFTER=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM agent_capabilities WHERE agent_id = 'agent-001';")

if [ "$CAP_COUNT_BEFORE" -gt 0 ] && [ "$CAP_COUNT_AFTER" -eq 0 ]; then
    echo -e "${GREEN}  ✓ Foreign key cascade (agent_capabilities → agents) works${NC}"
else
    echo -e "${RED}  ✗ Foreign key cascade failed (before: $CAP_COUNT_BEFORE, after: $CAP_COUNT_AFTER)${NC}"
    rm "$TEST_DB"
    exit 1
fi

# Restore data for further tests
sqlite3 "$TEST_DB" <<EOF
INSERT INTO workspaces (id, user_id, name, description, created_at, updated_at)
VALUES ('ws-001', 'user-001', 'My Workspace', 'A test workspace', datetime('now'), datetime('now'));

INSERT INTO agents (id, name, description, type, status, model_name, created_at, updated_at)
VALUES ('agent-001', 'Primary Agent', 'Main AI assistant', 'primary', 'active', 'gpt-4.1-mini', datetime('now'), datetime('now'));

INSERT INTO agent_capabilities (id, agent_id, capability_type, name, description, created_at)
VALUES ('cap-001', 'agent-001', 'tool', 'file_write', 'Can write files to disk', datetime('now'));
EOF
echo ""

# Step 7: Verify CHECK constraints
echo -e "${YELLOW}Step 7: Testing CHECK constraints...${NC}"

# Test agents type constraint
if ! sqlite3 "$TEST_DB" "INSERT INTO agents (id, name, description, type, status, created_at, updated_at) VALUES ('agent-bad', 'Bad Agent', 'Invalid type', 'invalid_type', 'active', datetime('now'), datetime('now'));" 2>/dev/null; then
    echo -e "${GREEN}  ✓ agents type CHECK constraint works${NC}"
else
    echo -e "${RED}  ✗ agents type CHECK constraint failed${NC}"
    rm "$TEST_DB"
    exit 1
fi

# Test agents status constraint
if ! sqlite3 "$TEST_DB" "INSERT INTO agents (id, name, description, type, status, created_at, updated_at) VALUES ('agent-bad2', 'Bad Agent 2', 'Invalid status', 'primary', 'invalid_status', datetime('now'), datetime('now'));" 2>/dev/null; then
    echo -e "${GREEN}  ✓ agents status CHECK constraint works${NC}"
else
    echo -e "${RED}  ✗ agents status CHECK constraint failed${NC}"
    rm "$TEST_DB"
    exit 1
fi

# Test agent_capabilities capability_type constraint
if ! sqlite3 "$TEST_DB" "INSERT INTO agent_capabilities (id, agent_id, capability_type, name, created_at) VALUES ('cap-bad', 'agent-001', 'invalid_type', 'test', datetime('now'));" 2>/dev/null; then
    echo -e "${GREEN}  ✓ agent_capabilities capability_type CHECK constraint works${NC}"
else
    echo -e "${RED}  ✗ agent_capabilities capability_type CHECK constraint failed${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo ""

# Step 8: Verify UNIQUE constraints
echo -e "${YELLOW}Step 8: Testing UNIQUE constraints...${NC}"

# Test workspaces unique (user_id, name) constraint
if ! sqlite3 "$TEST_DB" "INSERT INTO workspaces (id, user_id, name, description, created_at, updated_at) VALUES ('ws-002', 'user-001', 'My Workspace', 'Duplicate name', datetime('now'), datetime('now'));" 2>/dev/null; then
    echo -e "${GREEN}  ✓ workspaces UNIQUE(user_id, name) constraint works${NC}"
else
    echo -e "${RED}  ✗ workspaces UNIQUE(user_id, name) constraint failed${NC}"
    rm "$TEST_DB"
    exit 1
fi

# Test agents unique name constraint
if ! sqlite3 "$TEST_DB" "INSERT INTO agents (id, name, description, type, status, created_at, updated_at) VALUES ('agent-002', 'Primary Agent', 'Duplicate name', 'primary', 'active', datetime('now'), datetime('now'));" 2>/dev/null; then
    echo -e "${GREEN}  ✓ agents UNIQUE(name) constraint works${NC}"
else
    echo -e "${RED}  ✗ agents UNIQUE(name) constraint failed${NC}"
    rm "$TEST_DB"
    exit 1
fi

# Test agent_capabilities unique (agent_id, capability_type, name) constraint
if ! sqlite3 "$TEST_DB" "INSERT INTO agent_capabilities (id, agent_id, capability_type, name, created_at) VALUES ('cap-002', 'agent-001', 'tool', 'file_write', datetime('now'));" 2>/dev/null; then
    echo -e "${GREEN}  ✓ agent_capabilities UNIQUE(agent_id, capability_type, name) constraint works${NC}"
else
    echo -e "${RED}  ✗ agent_capabilities UNIQUE(agent_id, capability_type, name) constraint failed${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo ""

# Step 9: Verify migration version recorded
echo -e "${YELLOW}Step 9: Verifying migration version...${NC}"
VERSION_COUNT=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM schema_migrations WHERE version = '20260204_v0.0.24';")
if [ "$VERSION_COUNT" -eq 1 ]; then
    echo -e "${GREEN}  ✓ Migration version recorded${NC}"
else
    echo -e "${RED}  ✗ Migration version not recorded${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo ""

# Step 10: Test rollback
echo -e "${YELLOW}Step 10: Testing rollback migration...${NC}"
if ! sqlite3 "$TEST_DB" < "$ROLLBACK_FILE" 2>&1; then
    echo -e "${RED}✗ Rollback failed${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo -e "${GREEN}✓ Rollback executed successfully${NC}"
echo ""

# Step 11: Verify tables removed after rollback
echo -e "${YELLOW}Step 11: Verifying tables removed...${NC}"
TABLES_AFTER=$(sqlite3 "$TEST_DB" "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;")

for table in "${EXPECTED_TABLES[@]}"; do
    if echo "$TABLES_AFTER" | grep -q "^${table}$"; then
        echo -e "${RED}  ✗ Table '${table}' still exists after rollback${NC}"
        rm "$TEST_DB"
        exit 1
    else
        echo -e "${GREEN}  ✓ Table '${table}' removed${NC}"
    fi
done
echo ""

# Step 12: Verify migration version removed
echo -e "${YELLOW}Step 12: Verifying migration version removed...${NC}"
VERSION_COUNT_AFTER=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM schema_migrations WHERE version = '20260204_v0.0.24';")
if [ "$VERSION_COUNT_AFTER" -eq 0 ]; then
    echo -e "${GREEN}  ✓ Migration version removed${NC}"
else
    echo -e "${RED}  ✗ Migration version still present${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo ""

# Step 13: Verify projects table still exists (data preserved)
echo -e "${YELLOW}Step 13: Verifying projects table preserved...${NC}"
if echo "$TABLES_AFTER" | grep -q "^projects$"; then
    echo -e "${GREEN}  ✓ Projects table preserved after rollback${NC}"
else
    echo -e "${RED}  ✗ Projects table was deleted${NC}"
    rm "$TEST_DB"
    exit 1
fi

PROJECT_COUNT=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM projects;")
if [ "$PROJECT_COUNT" -eq 2 ]; then
    echo -e "${GREEN}  ✓ Project data preserved (count: $PROJECT_COUNT)${NC}"
else
    echo -e "${RED}  ✗ Project data lost (expected: 2, found: $PROJECT_COUNT)${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo ""

# Cleanup
echo -e "${YELLOW}Cleaning up test database...${NC}"
rm "$TEST_DB"
echo -e "${GREEN}✓ Test database removed${NC}"
echo ""

# Success
echo "============================================================================"
echo -e "${GREEN}✓ All v0.0.24 migration tests passed successfully!${NC}"
echo "============================================================================"
echo ""
echo "Summary:"
echo "  - Migration up: ✓ Passed"
echo "  - Table creation: ✓ Passed (4 tables)"
echo "  - Index creation: ✓ Passed (13 indexes)"
echo "  - Data insertion: ✓ Passed (all tables)"
echo "  - Foreign key constraints: ✓ Passed (2 cascades)"
echo "  - CHECK constraints: ✓ Passed (3 constraints)"
echo "  - UNIQUE constraints: ✓ Passed (3 constraints)"
echo "  - Migration rollback: ✓ Passed"
echo "  - Table cleanup: ✓ Passed"
echo "  - Data preservation: ✓ Passed (projects table intact)"
echo ""
