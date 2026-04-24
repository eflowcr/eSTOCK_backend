package main

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/routes"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
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
		gin.DefaultWriter = io.Discard
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

	// Build shared email sender and notifications service.
	emailSender := wire.EmailSenderForConfig(config)
	var notifSvc *services.NotificationsService
	if db != nil {
		_, notifSvc = wire.NewNotifications(db, emailSender, config.TenantID)
	}

	r := gin.New()
	r.SetTrustedProxies(nil)
	r.Use(gin.Recovery())
	r.Use(tools.CORSMiddleware())
	r.Use(tools.RequestLogMiddleware())

	routes.RegisterRoutes(r, db, pool, config, redisClient, notifSvc)

	// Cron: stock alerts + stale reservations + lot expiration notifications.
	go func() {
		time.Sleep(30 * time.Second)

		analyzer := func() error {
			repo := &repositories.StockAlertsRepository{DB: db, Redis: redisClient}
			svc := services.NewStockAlertsService(repo)
			if _, resp := svc.Analyze(); resp != nil && resp.Error != nil {
				return resp.Error
			}
			if _, resp := svc.LotExpiration(); resp != nil && resp.Error != nil {
				return resp.Error
			}
			return nil
		}

		// lotNotifyFn notifies all admin users for expiring lots.
		var lotNotifyFn func(eventType, title, body string) error
		if notifSvc != nil {
			lotNotifyFn = func(eventType, title, body string) error {
				var adminIDs []string
				if err := db.Table("users").
					Joins("JOIN roles ON users.role_id = roles.id").
					Where("LOWER(roles.name) = 'admin' AND users.is_active = true AND users.deleted_at IS NULL").
					Pluck("users.id", &adminIDs).Error; err != nil {
					log.Warn().Err(err).Msg("cron: query admins for lot expiration notify failed")
					return nil
				}
				ctx := context.Background()
				for _, uid := range adminIDs {
					if err := notifSvc.Send(ctx, uid, eventType, title, body, "lot", ""); err != nil {
						log.Warn().Err(err).Str("user_id", uid).Msg("cron: lot expiration notify send failed")
					}
				}
				return nil
			}
		}

		// HR1-M5: lowStockNotifyFn notifies all admin users for unresolved low-stock alerts.
		var lowStockNotifyFn func(sku, message string) error
		if notifSvc != nil {
			lowStockNotifyFn = func(sku, message string) error {
				var adminIDs []string
				if err := db.Table("users").
					Joins("JOIN roles ON users.role_id = roles.id").
					Where("LOWER(roles.name) = 'admin' AND users.is_active = true AND users.deleted_at IS NULL").
					Pluck("users.id", &adminIDs).Error; err != nil {
					log.Warn().Err(err).Msg("cron: query admins for low_stock notify failed")
					return nil
				}
				title := "Alerta: stock bajo — " + sku
				ctx := context.Background()
				for _, uid := range adminIDs {
					if err := notifSvc.Send(ctx, uid, "low_stock", title, message, "stock_alert", sku); err != nil {
						log.Warn().Err(err).Str("sku", sku).Str("user_id", uid).Msg("cron: low_stock notify send failed")
					}
				}
				return nil
			}
		}

		log.Info().Msg("cron: first run (post-startup)")
		tools.CronDispatch(db, analyzer, lotNotifyFn, lowStockNotifyFn)

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			tools.CronDispatch(db, analyzer, lotNotifyFn, lowStockNotifyFn)
		}
	}()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api/docs/openapi.json")))

	log.Info().Str("address", config.ServerAddress).Msg("Server listening")
	if err := r.Run(config.ServerAddress); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
