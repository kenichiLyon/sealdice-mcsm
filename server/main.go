package main

import (
	"log"
	"net/http"

	"sealdice-mcsm/server/app"
	"sealdice-mcsm/server/config"
	"sealdice-mcsm/server/infra"
	"sealdice-mcsm/server/repo"
	"sealdice-mcsm/server/ws"
)

func main() {
	// 1. Config
	cfg := config.Load()

	// 2. Repository
	log.Printf("Initializing SQLite repository at %s...", cfg.DBPath)
	r, err := repo.NewSQLiteRepo(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer r.Close()

	// 3. Infrastructure
	log.Printf("Initializing MCSM client (%s)...", cfg.MCSMURL)
	m := infra.NewMCSMClient(cfg.MCSMURL, cfg.MCSMKey)

	log.Printf("Initializing File Reader (base: %s)...", cfg.FSBase)
	f := infra.NewQRCodeReader(cfg.FSBase)

	// 4. Application Service
	application := app.NewApplication(r, m, f)

	// 5. WebSocket Server
	server := ws.NewServer(application)

	// 6. HTTP Handler
	http.Handle("/ws", server)

	// Start Server
	log.Printf("Server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
