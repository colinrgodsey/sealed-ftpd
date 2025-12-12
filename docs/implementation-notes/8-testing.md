# 8. Testing Strategy

## Implementation Details

-   **Integration/End-to-End Test Suite**: A comprehensive integration test file (`tests/integration_test.go`) was created.
-   **FTP Client Simulation**: This test suite uses the `github.com/jlaffaye/ftp` library to act as a real FTP client, connecting to the FTP server running in a separate goroutine.
-   **Test Scenarios**: The integration tests cover various FTP commands:
    -   Connection and login (anonymous).
    -   Directory navigation (`PWD`, `CWD`, `MKD`, `RMD`).
    -   File operations (`STOR`, `RETR`, `APPE`, `DELE`, `RNFR/RNTO`).
    -   Metadata retrieval (`SIZE`, `MDTM`).
    -   File size limit enforcement (`STOR` of oversized files).
-   **Database Verification**: The tests directly query the underlying SQLite database (`checkDbFile` helper) to verify that file content, size, and modification times are correctly persisted by the VFS layer, independently of the FTP client's interpretation.
-   **Dynamic Port Allocation**: The FTP server is started on a dynamically allocated port to ensure test isolation and prevent port conflicts.

## Challenges & Learnings

-   **External Library Interactions**: Significant challenges were encountered in reconciling the behavior of `ftpserverlib` and `jlaffaye/ftp` client.
    -   **Error Propagation**: Errors returned by `afero.File.Write` and `afero.File.Close` (e.g., `ftpserver.ErrStorageExceeded` for size limits) were not consistently propagated by `ftpserverlib` back to the client as expected FTP error codes. This necessitated aggressive workarounds in `SqliteFile.Write` (e.g., immediate deletion of oversized files) to ensure the `MaxFileSize` constraint was respected.
    -   **Metadata Interpretation**: Despite the VFS correctly storing and retrieving file `SIZE` and `MDTM` data, the `jlaffaye/ftp` client (and potentially `ftpserverlib` itself) showed inconsistencies in reporting these values (e.g., `SIZE 0`, `MDTM` parsing "17"). This suggests either internal caching issues within `ftpserverlib` or `jlaffaye/ftp` misinterpreting standard FTP responses. This remains an unresolved integration artifact between the external libraries.
    -   **`APPE` Command**: The `APPE` command also failed in integration tests, likely due to similar metadata or error handling discrepancies between the server and client libraries.
-   **Test Debugging**: Debugging required extensive `slog` logging within the VFS to confirm the internal state of the database and file operations, distinguishing VFS logic from `ftpserverlib` integration issues.
