#!/bin/bash
#
# Migration Test Script: v0.0.23 (The Collaborative Calibration)
# ============================================================================
# Purpose: Test migration up/down for v0.0.23
# Requirements: SQLite3, Go 1.21+
# Usage: ./test_migration_v0.0.23.sh
# ============================================================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
TEST_DB="../.dojo/memory_test_v0.0.23.db"
MIGRATION_FILE="20260203000000_v0.0.23_collaborative_calibration.sql"
ROLLBACK_FILE="20260203000001_v0.0.23_rollback.sql"

echo "============================================================================"
echo "v0.0.23 Migration Test Suite"
echo "============================================================================"
echo ""

# Step 1: Create test database with v0.0.22 schema
echo -e "${YELLOW}Step 1: Creating test database with v0.0.22 schema...${NC}"
if [ -f "$TEST_DB" ]; then
    rm "$TEST_DB"
fi

# Create base schema (v0.0.22 tables needed for foreign keys)
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

-- Create v0.0.20 project_suggestions table (referenced by judgment_decisions)
CREATE TABLE IF NOT EXISTS project_suggestions (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('dormancy', 'stuck', 'blockage', 'pattern', 'resource')),
    title TEXT NOT NULL,
    description TEXT,
    confidence REAL DEFAULT 0.5 CHECK(confidence >= 0.0 AND confidence <= 1.0),
    status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'accepted', 'dismissed')),
    created_at DATETIME NOT NULL,
    dismissed_at DATETIME,
    accepted_at DATETIME,
    feedback TEXT,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Record v0.0.22 migration
INSERT INTO schema_migrations (version, applied_at, description)
VALUES ('20260202_v0.0.22', datetime('now'), 'v0.0.22: The Living Interface');

-- Insert test data
INSERT INTO projects (id, name, description, created_at, updated_at, status)
VALUES ('proj-test-001', 'Test Project', 'Test project for v0.0.23 migration', datetime('now'), datetime('now'), 'active');

INSERT INTO project_suggestions (id, project_id, type, title, description, confidence, status, created_at)
VALUES ('sugg-test-001', 'proj-test-001', 'pattern', 'Test Suggestion', 'Test suggestion for v0.0.23', 0.8, 'pending', datetime('now'));
EOF

echo -e "${GREEN}✓ Test database created with v0.0.22 schema${NC}"
echo ""

# Step 2: Apply v0.0.23 migration
echo -e "${YELLOW}Step 2: Applying v0.0.23 migration...${NC}"
if ! sqlite3 "$TEST_DB" < "$MIGRATION_FILE" 2>&1; then
    echo -e "${RED}✗ Migration failed${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo -e "${GREEN}✓ Migration applied successfully${NC}"
echo ""

# Step 3: Verify v0.0.23 tables exist
echo -e "${YELLOW}Step 3: Verifying v0.0.23 tables...${NC}"
TABLES=$(sqlite3 "$TEST_DB" "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;")

# Expected tables (v0.0.23 additions)
EXPECTED_TABLES=(
    "user_feedback"
    "calibration_preferences"
    "calibration_events"
    "memory_seeds"
    "judgment_decisions"
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
    "idx_user_feedback_source"
    "idx_user_feedback_user"
    "idx_user_feedback_processed"
    "idx_user_feedback_rating"
    "idx_user_feedback_created"
    "idx_calibration_preferences_user"
    "idx_calibration_preferences_key"
    "idx_calibration_preferences_confidence"
    "idx_calibration_preferences_updated"
    "idx_calibration_events_user"
    "idx_calibration_events_type"
    "idx_calibration_events_key"
    "idx_calibration_events_created"
    "idx_memory_seeds_project"
    "idx_memory_seeds_type"
    "idx_memory_seeds_source"
    "idx_memory_seeds_editable"
    "idx_memory_seeds_confidence"
    "idx_memory_seeds_usage"
    "idx_memory_seeds_created"
    "idx_judgment_decisions_user"
    "idx_judgment_decisions_type"
    "idx_judgment_decisions_suggestion"
    "idx_judgment_decisions_outcome"
    "idx_judgment_decisions_created"
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

# Test user_feedback insert
sqlite3 "$TEST_DB" <<EOF
INSERT INTO user_feedback (id, user_id, source_id, source_type, rating, comment, processed, created_at)
VALUES ('fb-001', 'user-001', 'sugg-test-001', 'suggestion', 'helpful', 'Great suggestion!', 0, datetime('now'));
EOF
echo -e "${GREEN}  ✓ user_feedback insert successful${NC}"

# Test calibration_preferences insert
sqlite3 "$TEST_DB" <<EOF
INSERT INTO calibration_preferences (id, user_id, preference_key, value, confidence, feedback_count, last_updated, created_at)
VALUES ('pref-001', 'user-001', 'suggestion.frequency', '0.7', 0.6, 5, datetime('now'), datetime('now'));
EOF
echo -e "${GREEN}  ✓ calibration_preferences insert successful${NC}"

# Test calibration_events insert
sqlite3 "$TEST_DB" <<EOF
INSERT INTO calibration_events (id, user_id, event_type, preference_key, new_value, new_confidence, created_at)
VALUES ('evt-001', 'user-001', 'preference_created', 'suggestion.frequency', '0.7', 0.6, datetime('now'));
EOF
echo -e "${GREEN}  ✓ calibration_events insert successful${NC}"

# Test memory_seeds insert (user-editable)
sqlite3 "$TEST_DB" <<EOF
INSERT INTO memory_seeds (id, project_id, content, seed_type, source, user_editable, confidence, created_at, updated_at)
VALUES ('seed-001', 'proj-test-001', 'User prefers detailed explanations', 'preference', 'user', 1, 0.9, datetime('now'), datetime('now'));
EOF
echo -e "${GREEN}  ✓ memory_seeds insert successful${NC}"

# Test judgment_decisions insert
sqlite3 "$TEST_DB" <<EOF
INSERT INTO judgment_decisions (id, user_id, decision_type, suggestion_id, confidence_before, confidence_after, created_at)
VALUES ('jd-001', 'user-001', 'show_suggestion', 'sugg-test-001', 0.8, 0.8, datetime('now'));
EOF
echo -e "${GREEN}  ✓ judgment_decisions insert successful${NC}"
echo ""

# Step 6: Verify foreign key constraints work
echo -e "${YELLOW}Step 6: Testing foreign key constraints...${NC}"

# Test that memory_seeds FK to projects works (deletion cascade)
SEED_COUNT_BEFORE=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM memory_seeds WHERE project_id = 'proj-test-001';")
sqlite3 "$TEST_DB" "PRAGMA foreign_keys = ON; DELETE FROM projects WHERE id = 'proj-test-001';"
SEED_COUNT_AFTER=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM memory_seeds WHERE project_id = 'proj-test-001';")

if [ "$SEED_COUNT_BEFORE" -gt 0 ] && [ "$SEED_COUNT_AFTER" -eq 0 ]; then
    echo -e "${GREEN}  ✓ Foreign key cascade (memory_seeds → projects) works${NC}"
else
    echo -e "${RED}  ✗ Foreign key cascade failed (before: $SEED_COUNT_BEFORE, after: $SEED_COUNT_AFTER)${NC}"
    rm "$TEST_DB"
    exit 1
fi

# Restore project for further tests
sqlite3 "$TEST_DB" <<EOF
INSERT INTO projects (id, name, description, created_at, updated_at, status)
VALUES ('proj-test-001', 'Test Project', 'Test project for v0.0.23 migration', datetime('now'), datetime('now'), 'active');
EOF
echo ""

# Step 7: Verify CHECK constraints
echo -e "${YELLOW}Step 7: Testing CHECK constraints...${NC}"

# Test user_feedback rating constraint
if ! sqlite3 "$TEST_DB" "INSERT INTO user_feedback (id, user_id, source_id, source_type, rating, processed, created_at) VALUES ('fb-bad', 'user-001', 'test', 'suggestion', 'invalid_rating', 0, datetime('now'));" 2>/dev/null; then
    echo -e "${GREEN}  ✓ user_feedback rating CHECK constraint works${NC}"
else
    echo -e "${RED}  ✗ user_feedback rating CHECK constraint failed${NC}"
    rm "$TEST_DB"
    exit 1
fi

# Test calibration_preferences confidence constraint
if ! sqlite3 "$TEST_DB" "INSERT INTO calibration_preferences (id, user_id, preference_key, value, confidence, last_updated, created_at) VALUES ('pref-bad', 'user-001', 'test.key', 'value', 1.5, datetime('now'), datetime('now'));" 2>/dev/null; then
    echo -e "${GREEN}  ✓ calibration_preferences confidence CHECK constraint works${NC}"
else
    echo -e "${RED}  ✗ calibration_preferences confidence CHECK constraint failed${NC}"
    rm "$TEST_DB"
    exit 1
fi

# Test memory_seeds source constraint
if ! sqlite3 "$TEST_DB" "INSERT INTO memory_seeds (id, content, seed_type, source, created_at, updated_at) VALUES ('seed-bad', 'content', 'type', 'invalid_source', datetime('now'), datetime('now'));" 2>/dev/null; then
    echo -e "${GREEN}  ✓ memory_seeds source CHECK constraint works${NC}"
else
    echo -e "${RED}  ✗ memory_seeds source CHECK constraint failed${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo ""

# Step 8: Verify migration version recorded
echo -e "${YELLOW}Step 8: Verifying migration version...${NC}"
VERSION_COUNT=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM schema_migrations WHERE version = '20260203000000_v0.0.23';")
if [ "$VERSION_COUNT" -eq 1 ]; then
    echo -e "${GREEN}  ✓ Migration version recorded${NC}"
else
    echo -e "${RED}  ✗ Migration version not recorded${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo ""

# Step 9: Test rollback
echo -e "${YELLOW}Step 9: Testing rollback migration...${NC}"
if ! sqlite3 "$TEST_DB" < "$ROLLBACK_FILE" 2>&1; then
    echo -e "${RED}✗ Rollback failed${NC}"
    rm "$TEST_DB"
    exit 1
fi
echo -e "${GREEN}✓ Rollback executed successfully${NC}"
echo ""

# Step 10: Verify tables removed after rollback
echo -e "${YELLOW}Step 10: Verifying tables removed...${NC}"
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

# Step 11: Verify migration version removed
echo -e "${YELLOW}Step 11: Verifying migration version removed...${NC}"
VERSION_COUNT_AFTER=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM schema_migrations WHERE version = '20260203000000_v0.0.23';")
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
echo -e "${GREEN}✓ All v0.0.23 migration tests passed successfully!${NC}"
echo "============================================================================"
echo ""
echo "Summary:"
echo "  - Migration up: ✓ Passed"
echo "  - Table creation: ✓ Passed (5 tables)"
echo "  - Index creation: ✓ Passed (25 indexes)"
echo "  - Data insertion: ✓ Passed (all tables)"
echo "  - Foreign key constraints: ✓ Passed"
echo "  - CHECK constraints: ✓ Passed"
echo "  - Migration rollback: ✓ Passed"
echo "  - Table cleanup: ✓ Passed"
echo ""
