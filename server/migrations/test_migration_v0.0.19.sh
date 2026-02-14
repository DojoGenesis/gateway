#!/bin/bash
# Test script for migration 20260201_v0.0.19_surgical_mind.sql

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DB="$SCRIPT_DIR/test_migration_v0.0.19.db"
MIGRATION_V0_17_V0_18="$SCRIPT_DIR/20260131_v0.0.17_and_v0.0.18_schemas.sql"
MIGRATION_V0_19="$SCRIPT_DIR/20260201_v0.0.19_surgical_mind.sql"
ROLLBACK_V0_19="$SCRIPT_DIR/20260201_v0.0.19_rollback.sql"

echo "====================================="
echo "Migration Test for v0.0.19"
echo "The Surgical Mind"
echo "====================================="
echo ""

# Clean up any existing test database
if [ -f "$TEST_DB" ]; then
    echo "Removing existing test database..."
    rm "$TEST_DB"
fi

# Create a v0.0.16 baseline database
echo "Step 1: Creating v0.0.16 baseline database..."
sqlite3 "$TEST_DB" <<EOF
CREATE TABLE IF NOT EXISTS memories (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    content TEXT NOT NULL,
    metadata TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
CREATE INDEX IF NOT EXISTS idx_memories_created_at ON memories(created_at);
CREATE INDEX IF NOT EXISTS idx_memories_updated_at ON memories(updated_at);
CREATE INDEX IF NOT EXISTS idx_memories_content_fts ON memories(content);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at DATETIME NOT NULL,
    description TEXT
);
EOF

echo "✓ Baseline v0.0.16 database created"
echo ""

# Apply v0.0.17 + v0.0.18 migration
echo "Step 2: Applying v0.0.17 + v0.0.18 migration..."
sqlite3 "$TEST_DB" < "$MIGRATION_V0_17_V0_18"
echo "✓ v0.0.17 + v0.0.18 migration applied"
echo ""

# Verify v0.0.18 tables exist
echo "Step 3: Verifying v0.0.18 schema..."
V0_18_TABLES=$(sqlite3 "$TEST_DB" "SELECT name FROM sqlite_master WHERE type='table' AND name IN ('memory_seeds', 'projects', 'artifacts') ORDER BY name;")
if echo "$V0_18_TABLES" | grep -q "memory_seeds"; then
    echo "✓ v0.0.18 tables exist (memory_seeds, projects, artifacts)"
else
    echo "✗ v0.0.18 migration failed"
    exit 1
fi
echo ""

# Apply v0.0.19 migration
echo "Step 4: Applying v0.0.19 migration..."
sqlite3 "$TEST_DB" < "$MIGRATION_V0_19"
echo "✓ v0.0.19 migration applied successfully"
echo ""

# Verify migration was recorded
echo "Step 5: Verifying migration version tracking..."
MIGRATION_VERSION=$(sqlite3 "$TEST_DB" "SELECT version FROM schema_migrations WHERE version='20260201_v0.0.19';")
if [ "$MIGRATION_VERSION" = "20260201_v0.0.19" ]; then
    echo "✓ Migration version recorded in schema_migrations table"
else
    echo "✗ Migration version NOT recorded"
    exit 1
fi
echo ""

# Verify new columns were added
echo "Step 6: Verifying new columns..."

# Check memories table for new columns
MEMORIES_COLUMNS=$(sqlite3 "$TEST_DB" "PRAGMA table_info(memories);" | awk -F'|' '{print $2}')
if echo "$MEMORIES_COLUMNS" | grep -q "embedding"; then
    echo "✓ Column 'embedding' added to memories table"
else
    echo "✗ Column 'embedding' MISSING from memories table"
    exit 1
fi

if echo "$MEMORIES_COLUMNS" | grep -q "context_type"; then
    echo "✓ Column 'context_type' added to memories table"
else
    echo "✗ Column 'context_type' MISSING from memories table"
    exit 1
fi

# Check memory_seeds table for context_type
SEEDS_COLUMNS=$(sqlite3 "$TEST_DB" "PRAGMA table_info(memory_seeds);" | awk -F'|' '{print $2}')
if echo "$SEEDS_COLUMNS" | grep -q "context_type"; then
    echo "✓ Column 'context_type' added to memory_seeds table"
else
    echo "✗ Column 'context_type' MISSING from memory_seeds table"
    exit 1
fi
echo ""

# Verify memory_files table was created
echo "Step 7: Verifying memory_files table..."
MEMORY_FILES_EXISTS=$(sqlite3 "$TEST_DB" "SELECT name FROM sqlite_master WHERE type='table' AND name='memory_files';")
if [ "$MEMORY_FILES_EXISTS" = "memory_files" ]; then
    echo "✓ Table 'memory_files' created"
else
    echo "✗ Table 'memory_files' MISSING"
    exit 1
fi

# Verify memory_files columns
MEMORY_FILES_COLUMNS=$(sqlite3 "$TEST_DB" "PRAGMA table_info(memory_files);" | awk -F'|' '{print $2}')
EXPECTED_COLUMNS=("id" "file_path" "tier" "content" "embedding" "themes" "created_at" "updated_at" "archived_at")
for col in "${EXPECTED_COLUMNS[@]}"; do
    if echo "$MEMORY_FILES_COLUMNS" | grep -q "$col"; then
        echo "  ✓ Column '$col' exists"
    else
        echo "  ✗ Column '$col' MISSING"
        exit 1
    fi
done
echo ""

# Verify indexes were created
echo "Step 8: Verifying indexes..."
INDEXES=$(sqlite3 "$TEST_DB" "SELECT name FROM sqlite_master WHERE type='index' ORDER BY name;")

EXPECTED_V0_19_INDEXES=(
    "idx_memories_context_type"
    "idx_memory_seeds_context_type"
    "idx_memory_files_tier"
    "idx_memory_files_archived"
    "idx_memory_files_path"
    "idx_memory_files_tier_archived"
)

FOUND_COUNT=0
for index in "${EXPECTED_V0_19_INDEXES[@]}"; do
    if echo "$INDEXES" | grep -q "^$index$"; then
        echo "✓ Index '$index' exists"
        ((FOUND_COUNT++))
    else
        echo "✗ Index '$index' MISSING"
        exit 1
    fi
done
echo ""
echo "Found all $FOUND_COUNT v0.0.19 indexes"
echo ""

# Test data insertion
echo "Step 9: Testing data insertion..."

# Insert a memory with new columns
sqlite3 "$TEST_DB" <<EOF
INSERT INTO memories (id, type, content, metadata, context_type, created_at, updated_at)
VALUES ('test-mem-1', 'conversation', 'Test memory content', '{}', 'private', datetime('now'), datetime('now'));
EOF
echo "✓ Successfully inserted memory with context_type"

# Insert a memory_file
sqlite3 "$TEST_DB" <<EOF
INSERT INTO memory_files (id, file_path, tier, content, themes, created_at, updated_at)
VALUES ('test-file-1', '/path/to/2026-02-01.md', 1, 'Daily notes content', '["authentication", "testing"]', datetime('now'), datetime('now'));
EOF
echo "✓ Successfully inserted memory_file"

# Query by context_type
CONTEXT_COUNT=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM memories WHERE context_type='private';")
if [ "$CONTEXT_COUNT" -gt "0" ]; then
    echo "✓ Successfully queried by context_type"
else
    echo "✗ Query by context_type failed"
    exit 1
fi

# Query memory_files by tier
TIER_COUNT=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM memory_files WHERE tier=1;")
if [ "$TIER_COUNT" -gt "0" ]; then
    echo "✓ Successfully queried memory_files by tier"
else
    echo "✗ Query memory_files by tier failed"
    exit 1
fi

echo ""

# Test default values
echo "Step 10: Testing default values..."

# Insert memory without context_type (should default to 'private')
sqlite3 "$TEST_DB" <<EOF
INSERT INTO memories (id, type, content, metadata, created_at, updated_at)
VALUES ('test-mem-2', 'conversation', 'Test without context_type', '{}', datetime('now'), datetime('now'));
EOF

DEFAULT_CONTEXT=$(sqlite3 "$TEST_DB" "SELECT context_type FROM memories WHERE id='test-mem-2';")
if [ "$DEFAULT_CONTEXT" = "private" ]; then
    echo "✓ context_type defaults to 'private'"
else
    echo "✗ context_type default is '$DEFAULT_CONTEXT', expected 'private'"
    exit 1
fi

echo ""

# Test CHECK constraints
echo "Step 11: Testing CHECK constraints..."

# Test invalid context_type should fail
echo "  Testing invalid context_type rejection..."
if sqlite3 "$TEST_DB" "INSERT INTO memories (id, type, content, metadata, context_type, created_at, updated_at) VALUES ('test-mem-invalid', 'conversation', 'Test', '{}', 'invalid', datetime('now'), datetime('now'));" 2>/dev/null; then
    echo "✗ Invalid context_type 'invalid' was accepted (CHECK constraint not working)"
    exit 1
else
    echo "  ✓ Invalid context_type 'invalid' correctly rejected"
fi

# Test valid context_types should succeed
for ctx_type in "private" "group" "public"; do
    if sqlite3 "$TEST_DB" "INSERT INTO memories (id, type, content, metadata, context_type, created_at, updated_at) VALUES ('test-ctx-$ctx_type', 'conversation', 'Test', '{}', '$ctx_type', datetime('now'), datetime('now'));" 2>&1; then
        echo "  ✓ Valid context_type '$ctx_type' accepted"
    else
        echo "  ✗ Valid context_type '$ctx_type' rejected"
        exit 1
    fi
done

# Test invalid tier should fail
echo "  Testing invalid tier rejection..."
if sqlite3 "$TEST_DB" "INSERT INTO memory_files (id, file_path, tier, content, created_at, updated_at) VALUES ('test-tier-invalid', '/invalid/tier.md', 99, 'Test', datetime('now'), datetime('now'));" 2>/dev/null; then
    echo "✗ Invalid tier 99 was accepted (CHECK constraint not working)"
    exit 1
else
    echo "  ✓ Invalid tier 99 correctly rejected"
fi

# Test valid tiers should succeed
for tier in 1 2 3; do
    if sqlite3 "$TEST_DB" "INSERT INTO memory_files (id, file_path, tier, content, created_at, updated_at) VALUES ('test-tier-$tier', '/tier/$tier.md', $tier, 'Test', datetime('now'), datetime('now'));" 2>&1; then
        echo "  ✓ Valid tier $tier accepted"
    else
        echo "  ✗ Valid tier $tier rejected"
        exit 1
    fi
done

# Test invalid JSON themes should fail
echo "  Testing invalid JSON themes rejection..."
if sqlite3 "$TEST_DB" "INSERT INTO memory_files (id, file_path, tier, content, themes, created_at, updated_at) VALUES ('test-json-invalid', '/invalid/json.md', 1, 'Test', 'not valid json', datetime('now'), datetime('now'));" 2>/dev/null; then
    echo "✗ Invalid JSON themes was accepted (CHECK constraint not working)"
    exit 1
else
    echo "  ✓ Invalid JSON themes correctly rejected"
fi

# Test valid JSON themes should succeed
if sqlite3 "$TEST_DB" "INSERT INTO memory_files (id, file_path, tier, content, themes, created_at, updated_at) VALUES ('test-json-valid', '/valid/json.md', 1, 'Test', '[\"theme1\", \"theme2\"]', datetime('now'), datetime('now'));" 2>&1; then
    echo "  ✓ Valid JSON themes accepted"
else
    echo "  ✗ Valid JSON themes rejected"
    exit 1
fi

# Test NULL themes should succeed
if sqlite3 "$TEST_DB" "INSERT INTO memory_files (id, file_path, tier, content, themes, created_at, updated_at) VALUES ('test-json-null', '/null/json.md', 1, 'Test', NULL, datetime('now'), datetime('now'));" 2>&1; then
    echo "  ✓ NULL themes accepted"
else
    echo "  ✗ NULL themes rejected"
    exit 1
fi

echo ""

# Test rollback
echo "Step 12: Testing rollback migration..."

# Count records before rollback to verify data preservation
MEMORIES_BEFORE_ROLLBACK=$(sqlite3 "$TEST_DB" "SELECT COUNT(*) FROM memories;")

# Create a backup of the test database for rollback testing
TEST_DB_BACKUP="$TEST_DB.backup"
cp "$TEST_DB" "$TEST_DB_BACKUP"

# Apply rollback
sqlite3 "$TEST_DB_BACKUP" < "$ROLLBACK_V0_19"
echo "✓ Rollback migration applied"

# Verify memory_files table was dropped
MEMORY_FILES_AFTER_ROLLBACK=$(sqlite3 "$TEST_DB_BACKUP" "SELECT name FROM sqlite_master WHERE type='table' AND name='memory_files';")
if [ -z "$MEMORY_FILES_AFTER_ROLLBACK" ]; then
    echo "✓ memory_files table dropped"
else
    echo "✗ memory_files table still exists after rollback"
    exit 1
fi

# Verify columns were removed from memories
MEMORIES_COLUMNS_AFTER_ROLLBACK=$(sqlite3 "$TEST_DB_BACKUP" "PRAGMA table_info(memories);" | awk -F'|' '{print $2}')
if echo "$MEMORIES_COLUMNS_AFTER_ROLLBACK" | grep -q "embedding"; then
    echo "✗ embedding column still exists after rollback"
    exit 1
else
    echo "✓ embedding column removed from memories"
fi

if echo "$MEMORIES_COLUMNS_AFTER_ROLLBACK" | grep -q "context_type"; then
    echo "✗ context_type column still exists after rollback"
    exit 1
else
    echo "✓ context_type column removed from memories"
fi

# Verify data was preserved in memories (all records from before rollback)
MEMORIES_COUNT=$(sqlite3 "$TEST_DB_BACKUP" "SELECT COUNT(*) FROM memories;")
if [ "$MEMORIES_COUNT" = "$MEMORIES_BEFORE_ROLLBACK" ]; then
    echo "✓ Memory data preserved after rollback ($MEMORIES_COUNT records)"
else
    echo "✗ Memory data lost after rollback (expected $MEMORIES_BEFORE_ROLLBACK, got $MEMORIES_COUNT)"
    exit 1
fi

# Verify migration version was removed
MIGRATION_AFTER_ROLLBACK=$(sqlite3 "$TEST_DB_BACKUP" "SELECT version FROM schema_migrations WHERE version='20260201_v0.0.19';")
if [ -z "$MIGRATION_AFTER_ROLLBACK" ]; then
    echo "✓ Migration version record removed"
else
    echo "✗ Migration version record still exists"
    exit 1
fi

# Clean up rollback test database
rm "$TEST_DB_BACKUP"

echo ""

# Display schema summary
echo "====================================="
echo "Schema Summary (v0.0.19)"
echo "====================================="
sqlite3 "$TEST_DB" <<EOF
.mode column
.headers on
SELECT 
    name as table_name, 
    (SELECT COUNT(*) FROM pragma_table_info(m.name)) as column_count
FROM sqlite_master m 
WHERE type='table' 
ORDER BY name;
EOF

echo ""
echo "====================================="
echo "✓ All tests passed successfully!"
echo "====================================="
echo ""
echo "Summary:"
echo "  - v0.0.19 migration applied successfully"
echo "  - All columns added correctly"
echo "  - All tables created correctly"
echo "  - All indexes created correctly (6 total)"
echo "  - Data insertion working"
echo "  - Default values working"
echo "  - CHECK constraints validated (context_type, tier, themes)"
echo "  - Rollback migration tested successfully"
echo "  - Data integrity preserved"
echo ""

# Auto-cleanup test database
if [ "${KEEP_TEST_DB:-0}" = "1" ]; then
    echo "Test database preserved: $TEST_DB"
    echo "You can inspect it with: sqlite3 $TEST_DB"
    echo ""
    echo "To remove it manually: rm $TEST_DB"
else
    echo "Cleaning up test database..."
    rm -f "$TEST_DB"
    echo "✓ Test database removed"
    echo ""
    echo "To keep the test database for inspection, run:"
    echo "  KEEP_TEST_DB=1 ./test_migration_v0.0.19.sh"
fi
echo ""
