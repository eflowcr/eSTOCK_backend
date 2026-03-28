package main

import (
	"io"
	"os"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/routes"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	config, err := configuration.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("config load failed")
	}

	switch config.Environment {
	case "debug", "development":
		gin.SetMode(gin.DebugMode)
		gin.DefaultWriter = io.Discard // so only zerolog is used; no route dump or duplicate "Listening and serving"
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	dbURL := configuration.DatabaseURL(config)
	if err := tools.RunMigrations(config.MigrationURL, dbURL); err != nil {
		log.Fatal().Err(err).Msg("migrations failed")
	}
	log.Info().Msg("migrations applied")

	db := tools.InitDB(config)

	var pool *pgxpool.Pool
	if p, err := tools.InitPgxPool(config); err != nil {
		log.Fatal().Err(err).Msg("pgx pool failed")
	} else if p != nil {
		pool = p
		defer pool.Close()
	}

	redisClient := tools.InitRedis(config)
	if redisClient != nil {
		defer redisClient.Close()
	}

	r := gin.New()
	r.SetTrustedProxies(nil) // avoid "trust all proxies" warning; set explicitly if behind a reverse proxy
	r.Use(gin.Recovery())
	r.Use(tools.CORSMiddleware())
	r.Use(tools.RequestLogMiddleware())

	routes.RegisterRoutes(r, db, pool, config, redisClient)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api/docs/openapi.json")))

	log.Info().Str("address", config.ServerAddress).Msg("Server listening")
	if err := r.Run(config.ServerAddress); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
