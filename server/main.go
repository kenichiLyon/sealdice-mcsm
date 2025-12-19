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
	cfg := config.Load()
	r, err := repo.NewSQLiteRepo(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	mc := infra.NewClient(cfg.MCSMBaseURL, cfg.MCSMApiKey)
	fs := infra.NewLocalFS(cfg.FSBase)
	application := app.NewApplication(cfg, r, mc, fs)
	server := ws.NewServer(application)

	http.Handle("/ws", server)
	log.Printf("listening on %s", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, nil))
}

