# 7. Concurrency and Error Handling

## Implementation Details

-   **SQLite Optimization**:
    -   Updated `pkg/db/InitDB` to append `?_journal_mode=WAL&_busy_timeout=5000` to the database connection string. This enables Write-Ahead Logging (WAL) for better concurrency and sets a busy timeout to handle lock contention gracefully.
    -   Configured connection pooling on the `*sql.DB` object:
        -   `SetMaxOpenConns(100)`: Allows up to 100 concurrent open connections (and thus file descriptors/concurrent operations).
        -   `SetMaxIdleConns(10)`: Keeps 10 idle connections ready.
        -   `SetConnMaxLifetime(1 * time.Hour)`: Recycles connections periodically.
-   **Testing**:
    -   Added `TestConcurrentReads` to `pkg/vfs/vfs_test.go`. This test spawns 50 concurrent goroutines that open and read the same file simultaneously, verifying that the WAL mode allows concurrent readers without blocking or errors.

## Challenges & Learnings

-   **Connection Pooling**: Tuning `MaxOpenConns` is important. Setting it too high might exhaust file descriptors (though less of an issue with a single SQLite file, it matters for the Go SQL driver's internal management). Setting it too low would bottleneck the "hundreds of concurrent users" requirement. 100 seems like a reasonable starting point for a single-node SQLite instance.
-   **WAL Mode**: Enabling WAL is critical for any SQLite application expecting concurrent reads and writes. Without it, readers block writers and vice versa.
