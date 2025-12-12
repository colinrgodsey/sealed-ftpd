# FTP Mimic Server

This project implements an FTP server that stores all its files and directories within a single SQLite database. It serves as a proof-of-concept to mimic FTP functionality with a non-traditional backend.

## Features

-   **SQLite Backend**: All file system operations (create, read, update, delete, list directories) are performed against a SQLite database.
-   **No Authentication**: The server is configured for anonymous access; any username/password combination will be accepted.
-   **Passive Mode Support**: The server supports FTP passive mode, configurable via command-line flags.
-   **High Concurrency**: Designed to handle several hundred concurrent users, optimized with SQLite WAL (Write-Ahead Logging) and connection pooling.
-   **File Size Limit**: A strict 10MB file size limit is enforced for all uploads. Files exceeding this limit are rejected and not stored.

## Building and Running

### Prerequisites

-   Go 1.24.4 or higher

### Build

To build the FTP Mimic server:

```bash
go build -o ftp-mimic-server ./cmd/ftpserver
```

### Run

To run the server:

```bash
./ftp-mimic-server
```

**Configuration Options:**

You can configure the server using command-line flags:

-   `--listen-addr`: Address for the FTP server to listen on (default: `127.0.0.1:2121`)
-   `--passive-port-start`: Start of the passive port range (default: `20000`)
-   `--passive-port-end`: End of the passive port range (default: `20009`)
-   `--connection-timeout`: Connection timeout duration (default: `5m`)
-   `--db-path`: Path to the SQLite database file (default: `./ftp-mimic.db`)
-   `--log-level`: Logging level (debug, info, warn, error) (default: `info`)

**Example:**

```bash
./ftp-mimic-server --listen-addr "0.0.0.0:21" --passive-port-start 50000 --passive-port-end 50010 --log-level debug
```

## Testing

Unit tests for individual components can be run with:

```bash
go test ./...
```

Integration tests, which start the server and interact with it using a client, can be found in the `tests/` directory.

## Development Plan

The development process is documented in the `docs/development-plan/` directory, with detailed implementation notes for each step in `docs/implementation-notes/`.

## API Documentation

API documentation can be generated and viewed using `go doc`:

```bash
go doc ./pkg/vfs
go doc ./pkg/db
go doc ./pkg/config
# etc.
```

---
