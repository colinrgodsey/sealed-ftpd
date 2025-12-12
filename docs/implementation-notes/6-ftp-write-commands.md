# 6. Write FTP Commands

## Implementation Details

-   **Write Commands Support**: The VFS implementation (`pkg/vfs`) already supports the underlying operations for write commands:
    -   `STOR`: Supported via `Create` / `OpenFile` with `os.O_CREATE | os.O_TRUNC`.
    -   `APPE`: Supported via `OpenFile` with `os.O_APPEND`.
    -   `DELE`: Supported via `Remove`.
    -   `MKD`: Supported via `Mkdir`.
    -   `RMD`: Supported via `Remove`.
    -   `RNFR`/`RNTO`: Supported via `Rename`.
-   **File Size Limit**: Updated `pkg/vfs/vfs.go` to return the specific error `ftpserver.ErrStorageExceeded` (FTP code 552) when the 10MB file size limit is reached during `Write` or `Close` operations.
-   **Testing**: Updated `pkg/vfs/vfs_test.go` to explicitly verify that `ftpserver.ErrStorageExceeded` is returned when writing beyond the limit. Verified that existing tests cover creation, deletion, and renaming.

## Challenges & Learnings

-   **Error Mapping**: While generic errors are handled, returning specific errors like `ftpserver.ErrStorageExceeded` is crucial for correct FTP protocol responses (e.g., 552 vs 550). I had to look up the correct error variable exposed by the library.
-   **Test Precision**: Updating the tests to check for the specific error value ensures that the FTP server will behave exactly as expected according to the requirements, rather than just "failing" with an unknown error.
