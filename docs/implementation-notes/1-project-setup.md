# 1. Project Setup and Dependencies

## Implementation Details

-   Initialized Go module `ftp-mimic`.
-   Initialized Git repository.
-   Added `.gitignore` for Go projects.
-   Added dependencies:
    -   `github.com/mattn/go-sqlite3`: SQLite3 driver.
    -   `github.com/fclairamb/ftpserverlib`: FTP server library to handle protocol details, allowing us to focus on the VFS backend.

## Reasoning

-   `ftpserverlib` was chosen because it provides a clear interface for implementing a custom driver (FileSystem), which fits our requirement of backing the FTP server with a SQLite database.
-   `go-sqlite3` is the standard, stable driver for SQLite in Go.
