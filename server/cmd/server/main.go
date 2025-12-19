package main

import (
	"flag"
	"log"
	"net/http"

	"sealdice-mcsm/internal/app"
	"sealdice-mcsm/internal/infra/fs"
	"sealdice-mcsm/internal/infra/mcsm"
	"sealdice-mcsm/internal/repo"
	"sealdice-mcsm/internal/transport/ws"
)

func main() {
	// Flags
	port := flag.String("port", "8080", "Server port")
	dbPath := flag.String("db", "mcsm.db", "SQLite database path")
	mcsmURL := flag.String("mcsm-url", "http://localhost:23333", "MCSM API Base URL")
	mcsmKey := flag.String("mcsm-key", "your-api-key", "MCSM API Key")
	fsBase := flag.String("fs-base", ".", "Base directory for file reading (QR code)")
	flag.Parse()

	// 1. Repository
	log.Printf("Initializing SQLite repository at %s...", *dbPath)
	r, err := repo.NewSQLiteRepo(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer r.Close()

	// 2. Infrastructure
	log.Printf("Initializing MCSM client (%s)...", *mcsmURL)
	m := mcsm.NewClient(*mcsmURL, *mcsmKey)

	log.Printf("Initializing File Reader (base: %s)...", *fsBase)
	f := fs.NewFileReader(*fsBase)

	// 3. Application Service
	svc := app.NewService(r, m, f)

	// 4. WebSocket Server
	server := ws.NewServer(svc)

	// 5. HTTP Handler
	http.Handle("/ws", server)

	// Start Server
	log.Printf("Server starting on :%s", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
