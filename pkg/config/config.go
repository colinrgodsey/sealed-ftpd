package config

import (
	"flag"
	"time"
)

// Config holds all application configuration
type Config struct {
	ListenAddr        string
	PassivePortStart  int
	PassivePortEnd    int
	ConnectionTimeout time.Duration
	DBPath            string
	LogLevel          string
}

// ParseFlags parses command-line flags into a Config struct
func ParseFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.ListenAddr, "listen-addr", "127.0.0.1:2121", "Address to listen on (e.g., 0.0.0.0:2121)")
	flag.IntVar(&cfg.PassivePortStart, "passive-port-start", 20000, "Start of the passive port range")
	flag.IntVar(&cfg.PassivePortEnd, "passive-port-end", 20009, "End of the passive port range")
	flag.DurationVar(&cfg.ConnectionTimeout, "connection-timeout", 5*time.Minute, "Connection timeout duration (e.g., 5m)")
	flag.StringVar(&cfg.DBPath, "db-path", "./ftp-mimic.db", "Path to the SQLite database file")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Logging level (debug, info, warn, error)")

	flag.Parse()

	return cfg
}
