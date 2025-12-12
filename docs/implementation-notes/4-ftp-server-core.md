# 4. FTP Server Core

## Implementation Details

-   **Server Entrypoint**: Created `cmd/ftpserver/main.go` to serve as the main application entry point.
-   **Database Initialization**: The `main.go` initializes the SQLite database using `pkg/db.InitDB`.
-   **Driver Integration**: An instance of `pkg/vfs.MainDriver` is created and passed to `ftpserver.NewFtpServer`.
-   **Passive Mode Configuration**: The `MainDriver` now accepts passive port start and end values. `main.go` uses `flag` to parse command-line arguments `-passive-port-start` and `-passive-port-end` (defaulting to 20000-20009) and passes them to the driver.
-   **Logging**:
    -   Integrated `github.com/fclairamb/go-log` using its `slog` adapter (`github.com/fclairamb/go-log/slog`).
    -   A standard `log/slog` logger is created and wrapped to provide a compatible logger for `ftpserver.Logger`.

## Challenges & Learnings

-   **ftpserverlib Configuration**: Direct access to `ftpServer.Settings` is not permitted as it's an unexported field. Configuration must be done via the `MainDriver.GetSettings()` method.
-   **Logger Interface**: The `ftpserverlib` expects an instance of `github.com/fclairamb/go-log.Logger` interface. The standard `log` package's `Logger` is not compatible due to differing method signatures (e.g., missing `Debug` method).
-   **go-log Adapter Usage**: The `github.com/fclairamb/go-log` package is an adapter library. To instantiate its `Logger`, one must use a specific subpackage (like `github.com/fclairamb/go-log/slog`) to wrap an existing logger (like `log/slog`). This required careful reading of its documentation and import paths.
-   **PortRange Access**: `PassiveTransferPortRange` is an interface (`ftpserver.PasvPortGetter`). To access its `Start` and `End` fields, it must be type-asserted to `ftpserver.PortRange`.
