package config

import (
	"flag"
)

type Config struct {
	Port    string
	DBPath  string
	MCSMURL string
	MCSMKey string
	FSBase  string
}

func Load() *Config {
	port := flag.String("port", "8080", "Server port")
	dbPath := flag.String("db", "mcsm.db", "SQLite database path")
	mcsmURL := flag.String("mcsm-url", "http://localhost:23333", "MCSM API Base URL")
	mcsmKey := flag.String("mcsm-key", "your-api-key", "MCSM API Key")
	fsBase := flag.String("fs-base", ".", "Base directory for file reading (QR code)")
	flag.Parse()

	return &Config{
		Port:    *port,
		DBPath:  *dbPath,
		MCSMURL: *mcsmURL,
		MCSMKey: *mcsmKey,
		FSBase:  *fsBase,
	}
}
