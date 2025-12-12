# SQLite FTP Emulator

This project is an FTP emulator that keeps all files in a single SQLite
database and provides full FTP functionality to access the "files"
stored in this SQLite database. The FTP structure should look and function
exactly like an FTP file system, but the backing data store is all in SQLite.

It will be written in Go, and should follow Go best practices (especially around
concurrency, coroutines, flow control).

## Scope & Constraints

*   **Authentication:** The server requires no authentication (anonymous access or accepts any credentials).
*   **Networking:** Priority is on **Passive Mode** support to ensure compatibility with modern network environments.
*   **Performance:** The system is designed for **high concurrency** (supporting several hundred concurrent users).
*   **Storage Limits:** To maintain performance while storing files as BLOBs in SQLite, there is a strict **10MB file size limit**.

## Development and Integration

The docs folder in this project stores this high-level project overview. There
is also a folder called `development-plan` that should store the high-level steps
taken to complete this integration. The `implementation-notes` folder
should be used to compile notes and reasoning for each step of the implementation
when that step of the implementation is done.

The development plan folder should be markdown files, prefixed with a number to
indicate the sequence for each step (these can be thought of as ticket numbers).
The implementation notes should have a markdown file with implementation notes
for each of these steps when that step is performed. These notes should include
thoughts around the integration, challenges faced, and any learnings.
Each implementation step should
also be completed with a new git commit to the local repo.

## Standards

Each integration step (and commit) should have unit tests written whenever there
is code written. All unit tests (`go test ./...`) must be run and pass before that integration step is
complete and ready to commit.