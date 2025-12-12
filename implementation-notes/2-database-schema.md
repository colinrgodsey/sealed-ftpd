# 2. Database Schema

## Implementation Details

-   Created `pkg/db` package.
-   Implemented `InitDB` function to initialize the SQLite database.
-   Defined `files` table schema:
    -   `id`: Auto-incrementing primary key.
    -   `path`: Unique full path of the file/directory.
    -   `parent_path`: Path of the parent directory for easier lookups.
    -   `name`: Name of the file/directory.
    -   `is_dir`: Boolean flag to distinguish directories.
    -   `size`: File size in bytes.
    -   `mod_time`: Modification timestamp.
    -   `content`: BLOB to store file content.
-   Added `ensureRoot` to make sure the root directory `/` always exists.
-   Added unit tests in `pkg/db/schema_test.go` to verify initialization and basic CRUD capability.

## Reasoning

-   A single table `files` simplifies the design as requested ("single SQLite database").
-   `path` is unique to act as a natural key for file lookups, which matches FTP's path-based operations.
-   `parent_path` is indexed to optimize directory listing (e.g., `SELECT * FROM files WHERE parent_path = ?`).
-   Storing content as a BLOB in the same table is the simplest approach for a "mimic" agent, though for a production system separate chunks or a separate table might be better for large files. Given the scope, this is appropriate.
