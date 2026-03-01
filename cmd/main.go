package main

import (
	"log"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/routes"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

func main() {
	config, err := configuration.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	switch config.Environment {
	case "debug", "development":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	dbURL := configuration.DatabaseURL(config)
	if err := tools.RunMigrations(config.MigrationURL, dbURL); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	db := tools.InitDB(config)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(tools.CORSMiddleware())

	routes.RegisterRoutes(r, db, config)

	if err := r.Run(config.ServerAddress); err != nil {
		log.Fatalf("server: %v", err)
	}
}
