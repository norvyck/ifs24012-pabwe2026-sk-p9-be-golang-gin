package server

import (
	"fmt"
	"log"

	"app/internal/config"
	"app/internal/database"

	"github.com/gin-gonic/gin"
)

var (
	loadConfigFn = config.LoadConfig
	connectDBFn  = database.Connect
	migrateDBFn  = database.Migrate
	seedDBFn     = database.Seed
	listenFn     = func(r *gin.Engine, addr string) error { return r.Run(addr) }
	fatalFn      = log.Fatalf
)

func Run() {
	cfg := loadConfigFn()

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := connectDBFn(cfg)
	if err != nil {
		fatalFn("Failed to connect database: %v", err)
		return
	}

	if err := migrateDBFn(db); err != nil {
		fatalFn("Failed to migrate database: %v", err)
		return
	}

	if err := seedDBFn(db, cfg); err != nil {
		fatalFn("Failed to seed database: %v", err)
		return
	}

	r := NewRouter(db, cfg)
	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("Delcom Blog API listening on http://localhost%s (Swagger: http://localhost%s/swagger/index.html)", addr, addr)
	if err := listenFn(r, addr); err != nil {
		fatalFn("Server stopped with error: %v", err)
	}
}
