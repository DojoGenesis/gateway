# Database Migrations

This directory contains SQL migration scripts for Dojo Genesis.

## Migration Files

### `20260201_v0.0.19_surgical_mind.sql`

**v0.0.19: The Surgical Mind**

This migration adds:

**Vector Embeddings:**
- `embedding BLOB` column to `memories` table
- Supports 768-dimensional float32 vectors for semantic search

**Context Type (Multi-User Foundations):**
- `context_type TEXT DEFAULT 'private'` to `memories` table
- `context_type TEXT DEFAULT 'private'` to `memory_seeds` table
- Privacy rules: non-private contexts only get Tier 1 data

**Hierarchical Memory:**
- `memory_files` table for file-based memory tracking
- Three tiers: 1 (raw daily notes), 2 (curated wisdom), 3 (compressed archive)
- 4 specialized indexes for efficient search and filtering

**Test Script:** `test_migration_v0.0.19.sh`

---

### `20260131_v0.0.17_and_v0.0.18_schemas.sql`

**Unified migration for v0.0.17 (The Thoughtful System) and v0.0.18 (The Creative Studio)**

This migration adds:

**v0.0.18 Tables:**
- `projects` - Project workspace organizational units
- `project_templates` - Pre-defined project structures
- `artifacts` - Persistent, version-controlled outputs
- `artifact_versions` - Version history with diffs
- `project_files` - Files within project directories

**v0.0.17 Tables:**
- `memory_seeds` - Compressed semantic memory representations
- `traces` - Hierarchical execution traces for debugging

**Integration:**
- Adds `project_id` column to `memories` table (commented out - requires manual execution if needed)

## Testing Migrations

### Test v0.0.19 Migration

Run the v0.0.19 test script to verify the migration on a clean database:

```bash
cd go_backend/migrations
./test_migration_v0.0.19.sh
```

The test script will:
1. Create a baseline v0.0.16 database
2. Apply v0.0.17 + v0.0.18 migration
3. Apply v0.0.19 migration
4. Verify all columns and tables were created
5. Verify all 6 new indexes were created
6. Test data insertion with new columns
7. Test default values (context_type = 'private')
8. Test rollback migration
9. Verify data integrity after rollback

### Test v0.0.17 + v0.0.18 Migration

Run the test script to verify the migration on a clean v0.0.16 database:

```bash
cd go_backend/migrations
./test_migration.sh
```

The test script will:
1. Create a baseline v0.0.16 database with the `memories` table
2. Apply the migration
3. Verify all tables and indexes were created
4. Test data insertion
5. Verify foreign key constraints

## Applying Migrations Manually

### ⚠️ **IMPORTANT: Backup Your Database First!**

**Before applying any migration, always create a backup of your database:**

```bash
# Backup the database
cp .dojo/memory.db .dojo/memory.db.backup-$(date +%Y%m%d-%H%M%S)

# Verify the backup was created
ls -lh .dojo/*.backup*
```

If something goes wrong, you can restore from backup:

```bash
# Restore from backup (replace with your backup filename)
cp .dojo/memory.db.backup-YYYYMMDD-HHMMSS .dojo/memory.db
```

### 🚀 **Deployment Order for v0.0.19**

**CRITICAL: Always apply database migrations BEFORE deploying application code.**

The v0.0.19 release adds new database columns (`embedding`, `context_type`) and a new table (`memory_files`). The application code expects these to exist.

**Correct deployment sequence:**

1. **Stop the application** (prevents write conflicts during migration)
   ```bash
   # Stop Dojo Genesis backend
   pkill -f "go run main.go" || systemctl stop dojo-genesis
   ```

2. **Backup the database** (see above)

3. **Apply the migration**
   ```bash
   sqlite3 .dojo/memory.db < go_backend/migrations/20260201_v0.0.19_surgical_mind.sql
   ```

4. **Verify migration succeeded**
   ```bash
   sqlite3 .dojo/memory.db "SELECT version FROM schema_migrations WHERE version='20260201_v0.0.19';"
   # Should output: 20260201_v0.0.19
   ```

5. **Deploy new application code** (build and start)
   ```bash
   cd go_backend && go build -o dojo-genesis
   ./dojo-genesis
   ```

**What happens if you deploy code first?**
- Application will fail to start (missing database columns)
- Write operations will fail with "no such column: context_type" errors
- Memory search operations will fail with "no such table: memory_files" errors

**Rollback procedure** (if deployment fails):
1. Stop application
2. Apply rollback migration (see "Rolling Back v0.0.19 Migration" below)
3. Deploy previous application version (v0.0.18)
4. Restart application

### Applying v0.0.19 Migration

To apply the v0.0.19 migration to an existing v0.0.18 database:

```bash
sqlite3 /path/to/database.db < 20260201_v0.0.19_surgical_mind.sql
```

For the default Dojo Genesis database:

```bash
sqlite3 .dojo/memory.db < go_backend/migrations/20260201_v0.0.19_surgical_mind.sql
```

### Rolling Back v0.0.19 Migration

If you need to rollback the v0.0.19 migration:

```bash
# BACKUP FIRST (see warning above)
sqlite3 .dojo/memory.db < go_backend/migrations/20260201_v0.0.19_rollback.sql
```

**Note:** The rollback script will:
- **Drop** the `memory_files` table and all its data
- **Remove** `embedding` and `context_type` columns from `memories`
- **Remove** `context_type` column from `memory_seeds`
- **Preserve** all existing data in `memories` and `memory_seeds`

### Applying v0.0.17 + v0.0.18 Migration

To apply this migration to an existing database:

```bash
sqlite3 /path/to/database.db < 20260131_v0.0.17_and_v0.0.18_schemas.sql
```

For the default Dojo Genesis database:

```bash
sqlite3 .dojo/memory.db < go_backend/migrations/20260131_v0.0.17_and_v0.0.18_schemas.sql
```

### Rolling Back v0.0.17 + v0.0.18 Migration

If you need to rollback this migration:

```bash
# BACKUP FIRST (see warning above)
sqlite3 .dojo/memory.db < go_backend/migrations/20260131_v0.0.17_and_v0.0.18_rollback.sql
```

**Note:** The rollback script will **permanently delete** all data in the v0.0.17 and v0.0.18 tables.

### Performance Expectations

**Migration Time:**
- **Empty database:** < 100ms
- **Small database (<1000 memories):** < 500ms
- **Large database (>10,000 memories):** < 2 seconds

**Impact During Migration:**
- The database is locked during migration (transaction-based)
- All read/write operations are blocked until migration completes
- Recommended to run during maintenance window or when application is stopped

**Post-Migration Performance:**
- No significant performance impact on existing queries
- New indexes may slightly increase write time (< 5%)
- Query performance for new tables optimized with 23 strategic indexes

## Schema Overview

### v0.0.19: The Surgical Mind

**Vector Embeddings** (`memories.embedding`, `memory_seeds.embedding` - already existed, `memory_files.embedding`)
- Stores 768-dimensional float32 vectors as binary BLOB
- Enables semantic search via cosine similarity
- Generated asynchronously by the active LLM provider
- NULL by default for existing records

**Context Types** (`memories.context_type`, `memory_seeds.context_type`)
- Values: `private` (default), `group`, `public`
- Enables privacy rules for multi-user support
- Non-private contexts only load Tier 1 data
- Indexed for efficient filtering

**Memory Files** (`memory_files`)
- Tracks file-based memories for hierarchical system
- Three tiers:
  - Tier 1: Raw daily notes (`memory/YYYY-MM-DD.md`)
  - Tier 2: Curated wisdom (`MEMORY.md`)
  - Tier 3: Compressed archive (`memory/archive/YYYY-MM.jsonl.gz`)
- Stores full content, embeddings, and extracted themes (JSON array)
- Supports archiving with `archived_at` timestamp
- 4 indexes for efficient search and filtering

### v0.0.18: Project Workspace & Artifact Engine

**Projects** (`projects`)
- Organizational units for grouping conversations, artifacts, and memory
- Each project has its own directory: `~/DojoProjects/{project_name}/`
- Supports templates and status tracking (active/archived/deleted)

**Artifacts** (`artifacts`, `artifact_versions`)
- Five types: document, diagram, code_project, data_viz, image
- Git-like versioning with full content and diffs
- Linked to projects and sessions
- Exportable to standard formats

**Project Files** (`project_files`)
- Tracks all files within project directories
- Supports upload tracking and metadata

### v0.0.17: Memory Garden & Trace Viewer

**Memory Seeds** (`memory_seeds`)
- Compressed semantic representations of knowledge
- Organized into tiers: 1 (working), 2 (episodic), 3 (semantic)
- Supports vector embeddings for semantic search
- Tracks confidence and access patterns

**Traces** (`traces`)
- Hierarchical execution traces
- Supports parent-child relationships for nested operations
- Tracks timing, status, and metadata
- Enables debugging and performance analysis

## Integration Points

Both v0.0.17 and v0.0.18 extend the existing `memories` table with a `project_id` column to enable project-scoped context. This is commented out in the migration and should be applied manually if needed:

```sql
ALTER TABLE memories ADD COLUMN project_id TEXT;
CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(project_id);
```

## Notes

- All migrations use `CREATE TABLE IF NOT EXISTS` for idempotency
- Foreign key constraints are defined but require `PRAGMA foreign_keys = ON` at runtime
- JSON columns (settings, metadata) are stored as TEXT and parsed by the application
- Indexes are optimized for common query patterns (filtering by project, type, date)
