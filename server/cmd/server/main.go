package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"sealdice-mcsm/server/config"
	"sealdice-mcsm/server/internal/api"
	"sealdice-mcsm/server/internal/data"
	"sealdice-mcsm/server/internal/service"
	"sealdice-mcsm/server/pkg/mcsm"
)

func main() {
	cfg := config.Load()

	// Repo
	repo, err := data.NewSQLiteRepo(cfg.DBPath)
	if err != nil {
		log.Fatal("Failed to init DB:", err)
	}
	defer repo.Close()

	// Clients
	mcClient := mcsm.NewClient(cfg.MCSM.URL, cfg.MCSM.APIKey)

	// Service
	svc := service.NewService(cfg, repo, mcClient)

	// API
	handler := api.NewHandler(svc, cfg)

	// Router
	r := gin.Default()
	handler.SetupRoutes(r)

	// Run
	log.Printf("Server starting on %s", cfg.Server.Port)
	if err := r.Run(cfg.Server.Port); err != nil {
		log.Fatal(err)
	}
}
