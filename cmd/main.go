package main

import (
	"context"
	"fmt"
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

	tools.EnsureDefaultAdmin(db, config)

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
	// TODO(CS5 — S3.5): nil disables X-Forwarded-For trust — correct for direct-deploy but if
	// app runs behind k3s ingress/LB, rate limiter will see the proxy IP for all requests, effectively
	// limiting all signups from the same source. Confirm deployment topology and configure trusted
	// proxy CIDRs (e.g. r.SetTrustedProxies([]string{"10.0.0.0/8"})) before go-live.
	r.SetTrustedProxies(nil)
	r.Use(gin.Recovery())
	r.Use(tools.CORSMiddleware())
	r.Use(tools.RequestLogMiddleware())

	routes.RegisterRoutes(r, db, pool, config, redisClient, notifSvc)

	// Cron: stock alerts + stale reservations + lot expiration notifications.
	go func() {
		time.Sleep(30 * time.Second)

		// S3.5 W2-B: analyzer runs once per active tenant; tools.RunStockAlertAnalysis
		// iterates the tenants table and invokes this callback for each tenant UUID.
		analyzer := func(tenantID string) error {
			repo := &repositories.StockAlertsRepository{DB: db, Redis: redisClient}
			svc := services.NewStockAlertsService(repo)
			if _, resp := svc.Analyze(tenantID); resp != nil && resp.Error != nil {
				return resp.Error
			}
			if _, resp := svc.LotExpiration(tenantID); resp != nil && resp.Error != nil {
				return resp.Error
			}
			return nil
		}

		// S3.5 W5.5 (HR-S3.5 C3): notify only the admins of the tenant whose lot is
		// expiring. Pre-W5.5 the closure queried admins globally, so a tenant 2 expiring
		// lot would email tenant 1 admins. The cron helper now passes tenantID per call.
		var lotNotifyFn func(tenantID, eventType, title, body string) error
		if notifSvc != nil {
			lotNotifyFn = func(tenantID, eventType, title, body string) error {
				var adminIDs []string
				if err := db.Table("users").
					Joins("JOIN roles ON users.role_id = roles.id").
					Where("users.tenant_id = ? AND LOWER(roles.name) = 'admin' AND users.is_active = true AND users.deleted_at IS NULL", tenantID).
					Pluck("users.id", &adminIDs).Error; err != nil {
					log.Warn().Err(err).Str("tenant_id", tenantID).Msg("cron: query admins for lot expiration notify failed")
					return nil
				}
				ctx := context.Background()
				for _, uid := range adminIDs {
					if err := notifSvc.Send(ctx, uid, eventType, title, body, "lot", ""); err != nil {
						log.Warn().Err(err).Str("tenant_id", tenantID).Str("user_id", uid).Msg("cron: lot expiration notify send failed")
					}
				}
				return nil
			}
		}

		// S3.5 W5.5 (HR-S3.5 C3): same per-tenant scoping for low-stock alerts.
		var lowStockNotifyFn func(tenantID, sku, message string) error
		if notifSvc != nil {
			lowStockNotifyFn = func(tenantID, sku, message string) error {
				var adminIDs []string
				if err := db.Table("users").
					Joins("JOIN roles ON users.role_id = roles.id").
					Where("users.tenant_id = ? AND LOWER(roles.name) = 'admin' AND users.is_active = true AND users.deleted_at IS NULL", tenantID).
					Pluck("users.id", &adminIDs).Error; err != nil {
					log.Warn().Err(err).Str("tenant_id", tenantID).Msg("cron: query admins for low_stock notify failed")
					return nil
				}
				title := "Alerta: stock bajo — " + sku
				ctx := context.Background()
				for _, uid := range adminIDs {
					if err := notifSvc.Send(ctx, uid, "low_stock", title, message, "stock_alert", sku); err != nil {
						log.Warn().Err(err).Str("tenant_id", tenantID).Str("sku", sku).Str("user_id", uid).Msg("cron: low_stock notify send failed")
					}
				}
				return nil
			}
		}

		// S3-W5-C: trialSendFn sends trial lifecycle emails directly via the email sender
		// (not via in-app notifications) since trial tenants may not have user accounts yet.
		// Fire-and-forget: errors are logged but never block the cron.
		var trialSendFn func(ctx context.Context, toEmail, tenantName, templateType string, daysLeft int) error
		if emailSender := wire.EmailSenderForConfig(config); emailSender != nil {
			trialSendFn = func(ctx context.Context, toEmail, tenantName, templateType string, daysLeft int) error {
				subject, htmlBody, textBody := tools.RenderTrialEmail(templateType, tenantName, daysLeft)
				if err := emailSender.Send(ctx, toEmail, subject, htmlBody, textBody); err != nil {
					log.Warn().Err(err).
						Str("email", toEmail).
						Str("template", templateType).
						Msg("cron: trial email send failed")
					return fmt.Errorf("trial email send: %w", err)
				}
				log.Info().Str("email", toEmail).Str("template", templateType).Msg("cron: trial email sent")
				return nil
			}
		}

		log.Info().Msg("cron: first run (post-startup)")
		tools.CronDispatch(db, analyzer, lotNotifyFn, lowStockNotifyFn, trialSendFn)

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			tools.CronDispatch(db, analyzer, lotNotifyFn, lowStockNotifyFn, trialSendFn)
		}
	}()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api/docs/openapi.json")))

	log.Info().Str("address", config.ServerAddress).Msg("Server listening")
	if err := r.Run(config.ServerAddress); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
