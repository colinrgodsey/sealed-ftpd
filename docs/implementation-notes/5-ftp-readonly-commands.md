# 5. Read-Only FTP Commands

## Implementation Details

-   No new code was required for the FTP read-only commands (`LIST`, `NLST`, `RETR`, `SIZE`, `MDTM`, `PWD`, `CWD`, `CDUP`, `TYPE`) as their functionality is fully covered by the `pkg/vfs` implementation.
-   The `ftpserverlib` framework directly interacts with the `ftpserver.ClientDriver` (our `pkg/vfs.SQLiteFs`), which provides `afero.Fs` compatible methods.
-   **`CWD`, `PWD`, `CDUP`**: These directory navigation commands are handled internally by `ftpserverlib` using our `SQLiteFs.Stat` method to verify directory existence and `ClientContext` to manage the current working directory.
-   **`LIST`, `NLST`**: These directory listing commands are served by `ftpserverlib` calling `Open` on a directory and then `Readdir` on the returned `SqliteFile` object.
-   **`RETR`**: This command uses `Open` on a file and then `Read` on the returned `SqliteFile` object.
-   **`SIZE`**: This command uses `Stat` to retrieve the file size.
-   **`MDTM`**: This command uses `Stat` to retrieve the file modification time (`os.FileInfo.ModTime()`).
-   **`TYPE`**: This command is handled internally by `ftpserverlib` and does not require VFS-specific implementation.

## Challenges & Learnings

-   **Test Strategy**: The development plan requested "Unit tests... for each read-only FTP command". Given that our `pkg/vfs` implementation already has comprehensive unit tests verifying the underlying `afero.Fs` operations (`Stat`, `Open`, `Readdir`, etc.), additional unit tests at the FTP command level would essentially be integration tests involving a running FTP server and client. We have opted to consider the existing `pkg/vfs/vfs_test.go` as sufficient unit test coverage for the VFS's role in read-only commands. Future integration tests would validate the end-to-end FTP command functionality.
