#!/bin/bash
#
# Migration Test Script: v0.0.22 (The Living Interface)
# ============================================================================
# Purpose: Test migration up/down for v0.0.22
# Requirements: SQLite3, Go 1.21+
# Usage: ./test_migration_v0.0.22.sh
# ============================================================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
TEST_DB="../.dojo/memory_test_v0.0.22.db"
MIGRATION_FILE="20260202_v0.0.22_living_interface.sql"
ROLLBACK_FILE="20260202_v0.0.22_rollback.sql"

echo "============================================================================"
echo "v0.0.22 Migration Test Suite"
echo "============================================================================"
echo ""

# Step 1: Create test database with v0.0.20 schema
echo -e "${YELLOW}Step 1: Creating test database with v0.0.20 schema...${NC}"
if [ -f "$TEST_DB" ]; then
    rm "$TEST_DB"
fi

# Create base schema (simplified v0.0.20 tables needed for foreign keys)
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

-- Record v0.0.20 migration
INSERT INTO schema_migrations (version, applied_at, description)
VALUES ('20260201_v0.0.20', datetime('now'), 'v0.0.20: The Compassionate Companion');

-- Insert test data
INSERT INTO projects (id, name, description, created_at, updated_at, status)
VALUES ('proj-test-001', 'Test Project', 'Test project for migration', datetime('now'), datetime('now'), 'active');
EOF

echo -e "${GREEN}✓ Test database created with v0.0.20 schema${NC}"
echo ""

# Step 2: Apply v0.0.22 migration
echo -e "${YELLOW}Step 2: Applying v0.0.22 migration...${NC}"
if ! sqlite3 "$TEST_DB" < "$MIGRATION_FILE" 2>&1; then
    echo -e "${RED}✗ Migration failed${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo -e "${GREEN}✓ Migration applied successfully${NC}"
echo ""

# Step 3: Verify v0.0.22 tables exist
echo -e "${YELLOW}Step 3: Verifying v0.0.22 tables...${NC}"
TABLES=$(sqlite3 "$TEST_DB" "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;")

# Expected tables (v0.0.22 additions)
EXPECTED_TABLES=(
    "layout_transitions"
    "user_preferences"
    "command_history"
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
    "idx_layout_transitions_session"
    "idx_layout_transitions_project"
    "idx_layout_transitions_created"
    "idx_layout_transitions_trigger"
    "idx_layout_transitions_modes"
    "idx_command_history_session"
    "idx_command_history_project"
    "idx_command_history_created"
    "idx_command_history_type"
    "idx_command_history_success"
    "idx_command_history_project_type"
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

# Step 5: Verify user_preferences default row
echo -e "${YELLOW}Step 5: Verifying user_preferences default row...${NC}"
PREF_COUNT=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM user_preferences WHERE id = 'default';")
if [ "$PREF_COUNT" -eq 1 ]; then
    echo -e "${GREEN}  ✓ Default preferences row exists${NC}"
    
    # Verify default values
    NOTIFY_MEMORY=$(sqlite3 "$TEST_DB" "SELECT notify_memory_compression FROM user_preferences WHERE id = 'default';")
    NOTIFY_SEEDS=$(sqlite3 "$TEST_DB" "SELECT notify_seeds_extracted FROM user_preferences WHERE id = 'default';")
    NOTIFY_GOALS=$(sqlite3 "$TEST_DB" "SELECT notify_goal_milestone FROM user_preferences WHERE id = 'default';")
    NOTIFY_DORMANCY=$(sqlite3 "$TEST_DB" "SELECT notify_project_dormancy FROM user_preferences WHERE id = 'default';")
    NOTIFY_PATTERNS=$(sqlite3 "$TEST_DB" "SELECT notify_common_patterns FROM user_preferences WHERE id = 'default';")
    
    if [ "$NOTIFY_MEMORY" -eq 1 ] && [ "$NOTIFY_SEEDS" -eq 1 ] && [ "$NOTIFY_GOALS" -eq 1 ] && \
       [ "$NOTIFY_DORMANCY" -eq 1 ] && [ "$NOTIFY_PATTERNS" -eq 1 ]; then
        echo -e "${GREEN}  ✓ All notification defaults are enabled${NC}"
    else
        echo -e "${RED}  ✗ Notification defaults are incorrect${NC}"
        rm "$TEST_DB"
        exit 1
    fi
else
    echo -e "${RED}  ✗ Default preferences row missing or duplicate${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo ""

# Step 6: Test inserting data into new tables
echo -e "${YELLOW}Step 6: Testing data insertion...${NC}"

# Test layout_transitions insert
sqlite3 "$TEST_DB" <<EOF
INSERT INTO layout_transitions (id, from_mode, to_mode, trigger_type, session_id, project_id, created_at, duration_ms)
VALUES ('lt-001', 'Focus', 'Create', 'automatic', 'sess-001', 'proj-test-001', datetime('now'), 5000);
EOF
echo -e "${GREEN}  ✓ layout_transitions insert successful${NC}"

# Test command_history insert
sqlite3 "$TEST_DB" <<EOF
INSERT INTO command_history (id, command_type, command_text, session_id, project_id, confidence, success, created_at)
VALUES ('cmd-001', 'show_goals', '/goals', 'sess-001', 'proj-test-001', 1.0, 1, datetime('now'));
EOF
echo -e "${GREEN}  ✓ command_history insert successful${NC}"

# Test updating user_preferences
sqlite3 "$TEST_DB" <<EOF
UPDATE user_preferences SET notify_memory_compression = 0, updated_at = datetime('now') WHERE id = 'default';
EOF
UPDATED_NOTIFY=$(sqlite3 "$TEST_DB" "SELECT notify_memory_compression FROM user_preferences WHERE id = 'default';")
if [ "$UPDATED_NOTIFY" -eq 0 ]; then
    echo -e "${GREEN}  ✓ user_preferences update successful${NC}"
else
    echo -e "${RED}  ✗ user_preferences update failed${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo ""

# Step 7: Verify migration version recorded
echo -e "${YELLOW}Step 7: Verifying migration version...${NC}"
VERSION_COUNT=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM schema_migrations WHERE version = '20260202_v0.0.22';")
if [ "$VERSION_COUNT" -eq 1 ]; then
    echo -e "${GREEN}  ✓ Migration version recorded${NC}"
else
    echo -e "${RED}  ✗ Migration version not recorded${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo ""

# Step 8: Test rollback
echo -e "${YELLOW}Step 8: Testing rollback migration...${NC}"
if ! sqlite3 "$TEST_DB" < "$ROLLBACK_FILE" 2>&1; then
    echo -e "${RED}✗ Rollback failed${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo -e "${GREEN}✓ Rollback executed successfully${NC}"
echo ""

# Step 9: Verify tables removed after rollback
echo -e "${YELLOW}Step 9: Verifying tables removed...${NC}"
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

# Step 10: Verify migration version removed
echo -e "${YELLOW}Step 10: Verifying migration version removed...${NC}"
VERSION_COUNT_AFTER=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM schema_migrations WHERE version = '20260202_v0.0.22';")
if [ "$VERSION_COUNT_AFTER" -eq 0 ]; then
    echo -e "${GREEN}  ✓ Migration version removed${NC}"
else
    echo -e "${RED}  ✗ Migration version still present${NC}"
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
echo -e "${GREEN}✓ All v0.0.22 migration tests passed successfully!${NC}"
echo "============================================================================"
echo ""
echo "Summary:"
echo "  - Migration up: ✓ Passed"
echo "  - Table creation: ✓ Passed (3 tables)"
echo "  - Index creation: ✓ Passed (11 indexes)"
echo "  - Default data: ✓ Passed (user_preferences)"
echo "  - Data insertion: ✓ Passed"
echo "  - Migration rollback: ✓ Passed"
echo "  - Table cleanup: ✓ Passed"
echo ""
