package main

import (
	"fmt"
	"log"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	if err := database.Seed(db, cfg); err != nil {
		log.Fatalf("Failed to seed database: %v", err)
	}

	r := server.NewRouter(db, cfg)
	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("Delcom Blog API listening on http://localhost%s (Swagger: http://localhost%s/swagger/index.html)", addr, addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}
