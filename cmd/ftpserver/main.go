package main

import (
	"fmt"        // For Sprintf
	stdlog "log" // Alias standard log
	"log/slog"   // Standard library slog
	"os"
	"strings"

	"github.com/colinrgodsey/sealed-ftpd/pkg/config" // New config package
	"github.com/colinrgodsey/sealed-ftpd/pkg/db"
	"github.com/colinrgodsey/sealed-ftpd/pkg/vfs"

	ftpserver "github.com/fclairamb/ftpserverlib"
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

	// Set the slog logger directly
	ftpServer.Logger = slogLogger

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
