# 3. Virtual File System (VFS)

## Implementation Details

-   **Refactored Architecture**: Adopted the correct `ftpserverlib` v0.27.0 architecture.
    -   Implemented `MainDriver` to handle authentication and return a `ClientDriver`.
    -   Implemented `SQLiteFs` (ClientDriver) which embeds/implements `afero.Fs`.
    -   Implemented `SqliteFile` which implements `afero.File`.
-   **Afero Compliance**: The VFS now strictly adheres to the `github.com/spf13/afero` interface, which is required by `ftpserverlib`.
-   **Operations**:
    -   `Open`, `Create`, `OpenFile`: Handle file opening with appropriate `os.O_*` flags (Read, Write, Create, Truncate, Append).
    -   `Mkdir`, `MkdirAll`: Create directories in SQLite.
    -   `Remove`, `RemoveAll`: Delete files/directories from SQLite.
    -   `Rename`: Rename files/directories (atomic within DB transaction).
    -   `Stat`: Return `os.FileInfo` from DB metadata.
-   **File Size Limit**: Enforced 10MB limit in `SqliteFile.Write` and `Close`.
-   **Tests**: Updated `pkg/vfs/vfs_test.go` to test `MainDriver` entry point and full `afero.Fs` compliance (Mkdir, FileOps, Readdir, Remove, Rename).

## Reasoning

-   `ftpserverlib` v0.27.0 relies on `afero.Fs` for its filesystem abstraction. Directly implementing this interface ensures seamless integration with the library's core logic.
-   The previous attempt to mimic a different `Driver` interface was incorrect based on the library's actual API.
-   `SQLiteFs` provides a robust, SQL-backed filesystem that looks like a standard OS filesystem to the FTP server.

## Challenges & Learnings

-   **Package Name vs. Import Path**: We encountered persistent "undefined" errors because `github.com/fclairamb/ftpserverlib` is imported as `ftpserver` by default, but we initially tried to use `ftpserverlib` as the package name. Running `go run` on a small snippet helped clarify the actual package name used by the compiler.
-   **Interface Compliance**: The initial implementation attempted to implement a `Driver` interface that didn't match the library's v0.27.0 architecture. Reading the documentation revealed that `ClientDriver` embeds `afero.Fs`. This required a significant refactor to implement the standard `afero.Fs` interface (Open, Create, Stat, Rename, etc.) instead of custom methods.
-   **Dependencies**: Explicitly checking `go.mod` and understanding transitive dependencies (like `afero`) was crucial.
-   **"Undefined" Symbols**: Errors like `undefined: ftpserver.ErrNotExist` were confusing until we realized that `ftpserverlib` reuses standard `os` package errors (`os.ErrNotExist`) and types (`os.FileInfo`, `os.O_*` flags) rather than defining its own for everything.
