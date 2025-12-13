package main

import (
	"fmt" // For Sprintf
	stdlog "log" // Alias standard log
	"log/slog"   // Standard library slog
	"os"
	"strings"

	"ftp-mimic/pkg/config" // New config package
	"ftp-mimic/pkg/db"
	"ftp-mimic/pkg/vfs"

	ftpserver "github.com/fclairamb/ftpserverlib"
	golog_slog "github.com/fclairamb/go-log/slog" // Adapter for slog
)

func main() {
	cfg := config.ParseFlags()

	// Initialize the database
	sqliteDB, err := db.InitDB(cfg.DBPath)
	if err != nil {
		stdlog.Fatalf("Failed to initialize database: %v", err)
	}
	defer sqliteDB.Close()

	// Create our MainDriver
	mainDriver := vfs.NewMainDriver(sqliteDB, cfg.PassivePortStart, cfg.PassivePortEnd, cfg.ListenAddr, cfg.ConnectionTimeout)

	// Create the FTP server
	ftpServer := ftpserver.NewFtpServer(mainDriver)

	// Determine log level
	var logLevel slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// Create a standard slog logger
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Wrap the slog logger with the go-log adapter
	ftpServer.Logger = golog_slog.NewWrap(slogLogger)

	settings, err := mainDriver.GetSettings()
	if err != nil {
		stdlog.Fatalf("Failed to get server settings: %v", err)
	}

	// Cast PassiveTransferPortRange to PortRange to access Start and End
	portRange := settings.PassiveTransferPortRange.(ftpserver.PortRange)
	passivePorts := fmt.Sprintf("%d-%d", portRange.Start, portRange.End)
	stdlog.Printf("Starting FTP server on %s with passive ports %s...", settings.ListenAddr, passivePorts)
	if err := ftpServer.ListenAndServe(); err != nil {
		stdlog.Fatalf("FTP server failed: %v", err)
	}
}
