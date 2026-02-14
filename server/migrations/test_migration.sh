#!/bin/bash
# Test script for migration 20260131_v0.0.17_and_v0.0.18_schemas.sql

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DB="$SCRIPT_DIR/test_migration.db"
MIGRATION_FILE="$SCRIPT_DIR/20260131_v0.0.17_and_v0.0.18_schemas.sql"

echo "====================================="
echo "Migration Test for v0.0.17 + v0.0.18"
echo "====================================="
echo ""

# Clean up any existing test database
if [ -f "$TEST_DB" ]; then
    echo "Removing existing test database..."
    rm "$TEST_DB"
fi

# Create a v0.0.16 baseline database (just memories table)
echo "Creating v0.0.16 baseline database..."
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
EOF

echo "✓ Baseline v0.0.16 database created"
echo ""

# Apply the migration
echo "Applying migration..."
sqlite3 "$TEST_DB" < "$MIGRATION_FILE"
echo "✓ Migration applied successfully"
echo ""

# Verify PRAGMA foreign_keys setting
echo "Verifying PRAGMA settings..."
PRAGMA_CHECK=$(sqlite3 "$TEST_DB" "PRAGMA foreign_keys;")
if [ "$PRAGMA_CHECK" = "1" ]; then
    echo "✓ Foreign keys are enabled"
else
    echo "⚠ Warning: Foreign keys are not enabled in this session"
    echo "  Note: PRAGMA foreign_keys must be set per connection"
fi
echo ""

# Verify migration was recorded
echo "Verifying migration version tracking..."
MIGRATION_VERSION=$(sqlite3 "$TEST_DB" "SELECT version FROM schema_migrations WHERE version='20260131_v0.0.17_and_v0.0.18';")
if [ "$MIGRATION_VERSION" = "20260131_v0.0.17_and_v0.0.18" ]; then
    echo "✓ Migration version recorded in schema_migrations table"
else
    echo "✗ Migration version NOT recorded"
    exit 1
fi
echo ""

# Verify tables were created
echo "Verifying tables..."
TABLES=$(sqlite3 "$TEST_DB" "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;")

EXPECTED_TABLES=(
    "artifacts"
    "artifact_versions"
    "memories"
    "memory_seeds"
    "project_files"
    "project_templates"
    "projects"
    "schema_migrations"
    "traces"
)

for table in "${EXPECTED_TABLES[@]}"; do
    if echo "$TABLES" | grep -q "^$table$"; then
        echo "✓ Table '$table' exists"
    else
        echo "✗ Table '$table' MISSING"
        exit 1
    fi
done
echo ""

# Verify indexes were created
echo "Verifying indexes..."
INDEXES=$(sqlite3 "$TEST_DB" "SELECT name FROM sqlite_master WHERE type='index' ORDER BY name;")

EXPECTED_INDEXES=(
    "idx_artifacts_project"
    "idx_artifacts_type"
    "idx_artifacts_updated"
    "idx_projects_status"
    "idx_projects_last_accessed"
    "idx_seeds_project"
    "idx_seeds_type"
    "idx_traces_session"
    "idx_versions_artifact"
)

FOUND_COUNT=0
for index in "${EXPECTED_INDEXES[@]}"; do
    if echo "$INDEXES" | grep -q "^$index$"; then
        echo "✓ Index '$index' exists"
        ((FOUND_COUNT++))
    else
        echo "  Index '$index' not found (may be ok)"
    fi
done
echo ""
echo "Found $FOUND_COUNT indexes"
echo ""

# Test data insertion
echo "Testing data insertion..."

# Insert a project
sqlite3 "$TEST_DB" <<EOF
INSERT INTO projects (id, name, description, created_at, updated_at, last_accessed_at)
VALUES ('test-proj-1', 'Test Project', 'A test project', datetime('now'), datetime('now'), datetime('now'));
EOF
echo "✓ Successfully inserted project"

# Insert an artifact
sqlite3 "$TEST_DB" <<EOF
INSERT INTO artifacts (id, project_id, type, name, latest_version, created_at, updated_at)
VALUES ('test-artifact-1', 'test-proj-1', 'document', 'Test Document', 1, datetime('now'), datetime('now'));
EOF
echo "✓ Successfully inserted artifact"

# Insert an artifact version
sqlite3 "$TEST_DB" <<EOF
INSERT INTO artifact_versions (id, artifact_id, version, content, created_at, created_by)
VALUES ('test-version-1', 'test-artifact-1', 1, 'Test content', datetime('now'), 'test-user');
EOF
echo "✓ Successfully inserted artifact version"

# Insert a memory seed
sqlite3 "$TEST_DB" <<EOF
INSERT INTO memory_seeds (id, project_id, type, content, tier, created_at, updated_at, last_accessed_at)
VALUES ('test-seed-1', 'test-proj-1', 'fact', 'Test seed content', 3, datetime('now'), datetime('now'), datetime('now'));
EOF
echo "✓ Successfully inserted memory seed"

# Insert a trace
sqlite3 "$TEST_DB" <<EOF
INSERT INTO traces (id, trace_type, name, started_at)
VALUES ('test-trace-1', 'query', 'Test Query', datetime('now'));
EOF
echo "✓ Successfully inserted trace"

echo ""

# Test foreign key constraints
echo "Testing foreign key constraints..."
sqlite3 "$TEST_DB" "PRAGMA foreign_keys = ON;"

# Try to insert artifact with non-existent project (should fail)
if sqlite3 "$TEST_DB" "PRAGMA foreign_keys = ON; INSERT INTO artifacts (id, project_id, type, name, latest_version, created_at, updated_at) VALUES ('bad-artifact', 'nonexistent', 'document', 'Bad', 1, datetime('now'), datetime('now'));" 2>/dev/null; then
    echo "✗ Foreign key constraint not enforced for artifacts.project_id"
    exit 1
else
    echo "✓ Foreign key constraint working for artifacts.project_id"
fi

echo ""

# Display schema for verification
echo "====================================="
echo "Schema Summary"
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
echo "✓ Migration test completed successfully!"
echo "====================================="
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
    echo "  KEEP_TEST_DB=1 ./test_migration.sh"
fi
echo ""
