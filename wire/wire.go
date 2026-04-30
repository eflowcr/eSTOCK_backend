// Package wire provides constructors that build repository + service per domain.
// Route handlers call wire.NewX(db) or wire.NewX(db, config) to get a service
// without constructing repos directly. Keeps route registration thin.
package wire

import (
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// NewArticles builds ArticlesRepository and ArticlesService. When pool is non-nil (Postgres), uses
// ArticlesRepositorySQLC; otherwise uses GORM ArticlesRepository (e.g. sqlserver).
func NewArticles(db *gorm.DB, pool *pgxpool.Pool) (ports.ArticlesRepository, *services.ArticlesService) {
	var r ports.ArticlesRepository
	if pool != nil {
		queries := sqlc.New(pool)
		r = repositories.NewArticlesRepositorySQLC(queries)
	} else {
		r = &repositories.ArticlesRepository{DB: db}
	}
	return r, services.NewArticlesService(r)
}

// NewArticlesWithDeps builds ArticlesService with optional CategoriesRepo and LocationsRepo for M2 validation.
func NewArticlesWithDeps(db *gorm.DB, pool *pgxpool.Pool) (ports.ArticlesRepository, *services.ArticlesService) {
	repo, svc := NewArticles(db, pool)
	if pool != nil {
		_, catSvc := NewCategories(pool)
		_, locSvc := NewLocations(db, pool)
		if catSvc != nil {
			svc.WithCategoriesRepo(catSvc)
		}
		if locSvc != nil {
			svc.WithLocationsRepo(locSvc)
		}
	}
	return repo, svc
}

// NewAuditLog builds AuditLogRepository and AuditService. Requires pool (Postgres); no GORM fallback for audit.
func NewAuditLog(pool *pgxpool.Pool) (ports.AuditLogRepository, *services.AuditService) {
	if pool == nil {
		return nil, nil
	}
	queries := sqlc.New(pool)
	r := repositories.NewAuditLogsRepositorySQLC(queries)
	return r, services.NewAuditService(r)
}

// NewRoles builds RolesRepository for RBAC (GetRolePermissions). Returns a caching wrapper (TTL 2 min)
// so permission checks scale without hitting DB every request; returns nil if pool is nil.
func NewRoles(pool *pgxpool.Pool) ports.RolesRepository {
	if pool == nil {
		return nil
	}
	queries := sqlc.New(pool)
	base := repositories.NewRolesRepositorySQLC(queries)
	return repositories.NewRolesRepositoryCache(base, 2*time.Minute)
}

func NewAdjustments(db *gorm.DB, pool *pgxpool.Pool) (ports.AdjustmentsRepository, *services.AdjustmentsService) {
	r := &repositories.AdjustmentsRepository{DB: db}
	var reasonRepo ports.AdjustmentReasonCodesRepository
	if pool != nil {
		queries := sqlc.New(pool)
		reasonRepo = repositories.NewAdjustmentReasonCodesRepositorySQLC(queries)
	}
	return r, services.NewAdjustmentsService(r, reasonRepo)
}

func NewAuthentication(db *gorm.DB, config configuration.Config) (ports.AuthenticationRepository, *services.AuthenticationService) {
	r := &repositories.AuthenticationRepository{
		DB:          db,
		JWTSecret:   config.JWTSecret,
		Config:      config,
		EmailSender: EmailSenderForConfig(config),
	}
	return r, services.NewAuthenticationService(r, nil)
}

// NewAuthenticationWithRoles builds the auth service with roles repo so login response includes permissions.
// S3.8: also threads rolesRepo into the repository so the issued JWT embeds the permissions claim.
func NewAuthenticationWithRoles(db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository) (ports.AuthenticationRepository, *services.AuthenticationService) {
	r := &repositories.AuthenticationRepository{
		DB:              db,
		JWTSecret:       config.JWTSecret,
		Config:          config,
		EmailSender:     EmailSenderForConfig(config),
		RolesRepository: rolesRepo,
	}
	return r, services.NewAuthenticationService(r, rolesRepo)
}

// NewAuthenticationWithAudit builds the auth service with roles repo and audit service.
// S3.8: also threads rolesRepo into the repository so the issued JWT embeds the permissions claim.
func NewAuthenticationWithAudit(db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository, auditSvc *services.AuditService) (ports.AuthenticationRepository, *services.AuthenticationService) {
	r := &repositories.AuthenticationRepository{
		DB:              db,
		JWTSecret:       config.JWTSecret,
		Config:          config,
		EmailSender:     EmailSenderForConfig(config),
		AuditService:    auditSvc,
		RolesRepository: rolesRepo,
	}
	return r, services.NewAuthenticationService(r, rolesRepo)
}

// EmailSenderForConfig returns the appropriate EmailSender for the current environment.
//
// Priority order:
//  1. VPS_MANAGER_BASE_URL + VPS_MANAGER_API_KEY → GatewayEmailSender (routes via VPS Manager → Brevo)
//  2. RESEND_API_KEY set                         → ResendEmailSender (legacy Resend API)
//  3. SMTP_HOST set                              → SMTPEmailSender (generic SMTP/STARTTLS)
//  4. None set                                   → LoggerEmailSender (dev/test fallback)
func EmailSenderForConfig(config configuration.Config) tools.EmailSender {
	if config.VPSManagerBaseURL != "" && config.VPSManagerAPIKey != "" {
		fromAddr := config.VPSManagerFromAddr
		if fromAddr == "" {
			fromAddr = "noreply@eflowsuite.com"
		}
		return tools.NewGatewayEmailSender(config.VPSManagerBaseURL, config.VPSManagerAPIKey, fromAddr, "eSTOCK")
	}
	if config.ResendAPIKey != "" {
		fromAddr := config.ResendFromAddress
		if fromAddr == "" {
			fromAddr = "noreply@estock.app"
		}
		return &tools.ResendEmailSender{APIKey: config.ResendAPIKey, FromAddr: fromAddr, AppName: "eSTOCK"}
	}
	if config.SMTPHost != "" {
		return &tools.SMTPEmailSender{
			Host:     config.SMTPHost,
			Port:     config.SMTPPort,
			Username: config.SMTPUsername,
			Password: config.SMTPPassword,
			FromAddr: config.EmailFrom,
			AppName:  config.EmailFromName,
		}
	}
	return &tools.LoggerEmailSender{}
}

func NewDashboard(db *gorm.DB) (ports.DashboardRepository, *services.DashboardService) {
	r := &repositories.DashboardRepository{DB: db}
	return r, services.NewDashboardService(r)
}

func NewEncryption(config configuration.Config) (ports.EncryptionRepository, *services.EncryptionService) {
	r := &repositories.EncryptionRepository{JWTSecret: config.JWTSecret}
	return r, services.NewEncryptionService(r)
}

func NewGamification(db *gorm.DB) (ports.GamificationRepository, *services.GamificationService) {
	r := &repositories.GamificationRepository{DB: db}
	return r, services.NewGamificationService(r)
}

// NewInventory builds InventoryRepository and InventoryService. When pool is non-nil, injects
// ArticlesRepository so GetPickSuggestionsBySKU sorts by rotation (FIFO/FEFO) then quantity.
//
// S3.5 W2-A: tenantID is left empty here to preserve the legacy signature for
// internal callers (sales orders, backorders) that compose their own inventory
// repo. Routes that handle live HTTP traffic must use NewInventoryWithConfig.
func NewInventory(db *gorm.DB, pool *pgxpool.Pool) (ports.InventoryRepository, *services.InventoryService) {
	return NewInventoryWithConfig(db, pool, configuration.Config{})
}

// NewInventoryWithConfig is identical to NewInventory but stamps the configured
// tenant_id on every inventory_lots row created via this repository.
func NewInventoryWithConfig(db *gorm.DB, pool *pgxpool.Pool, config configuration.Config) (ports.InventoryRepository, *services.InventoryService) {
	r := &repositories.InventoryRepository{DB: db, TenantID: config.TenantID}
	var articlesRepo ports.ArticlesRepository
	if pool != nil {
		articlesRepo, _ = NewArticles(db, pool)
	}
	return r, services.NewInventoryService(r, articlesRepo)
}

func NewInventoryMovements(db *gorm.DB) (ports.InventoryMovementsRepository, *services.InventoryMovementsService) {
	r := &repositories.InventoryMovementsRepository{DB: db}
	return r, services.NewInventoryMovementsService(r)
}

// NewLocations builds LocationsRepository and LocationsService. When pool is non-nil, uses
// LocationsRepositorySQLC (CRUD via sqlc; Excel import/export delegated to GORM).
func NewLocations(db *gorm.DB, pool *pgxpool.Pool) (ports.LocationsRepository, *services.LocationsService) {
	var r ports.LocationsRepository
	if pool != nil {
		queries := sqlc.New(pool)
		gormLoc := &repositories.LocationsRepository{DB: db}
		r = repositories.NewLocationsRepositorySQLC(queries, gormLoc)
	} else {
		r = &repositories.LocationsRepository{DB: db}
	}
	return r, services.NewLocationsService(r)
}

// NewLots builds LotsRepository and LotsService. When pool is non-nil, uses LotsRepositorySQLC and
// injects ArticlesRepository so GetLotsBySKU returns lots in rotation order (FIFO/FEFO) for picking/receiving.
func NewLots(db *gorm.DB, pool *pgxpool.Pool) (ports.LotsRepository, *services.LotsService) {
	var r ports.LotsRepository
	var articlesRepo ports.ArticlesRepository
	if pool != nil {
		queries := sqlc.New(pool)
		r = repositories.NewLotsRepositorySQLCWithGORM(queries, db)
		articlesRepo, _ = NewArticles(db, pool)
	} else {
		r = &repositories.LotsRepository{DB: db}
	}
	return r, services.NewLotsService(r, articlesRepo)
}

// NewPickingTask builds PickingTaskRepository and PickingTaskService.
// soRepo is optional (nil-safe): when non-nil, SO3 cross-domain link is active and
// CompletePickingTask will update picked quantities on the linked sales order.
func NewPickingTask(db *gorm.DB, auditSvc *services.AuditService, notifSvc *services.NotificationsService, soRepo repositories.SOPickedQtyUpdater) (ports.PickingTaskRepository, *services.PickingTaskService) {
	r := &repositories.PickingTaskRepository{
		DB:               db,
		AuditService:     auditSvc,
		NotificationsSvc: notifSvc,
		SORepository:     soRepo,
	}
	return r, services.NewPickingTaskService(r)
}

// NewPresentations builds PresentationsRepository and PresentationsService. When pool is non-nil (Postgres), uses
// PresentationsRepositorySQLC; otherwise uses GORM PresentationsRepository.
func NewPresentations(db *gorm.DB, pool *pgxpool.Pool) (ports.PresentationsRepository, *services.PresentationsService) {
	var r ports.PresentationsRepository
	if pool != nil {
		queries := sqlc.New(pool)
		r = repositories.NewPresentationsRepositorySQLC(queries)
	} else {
		r = &repositories.PresentationsRepository{DB: db}
	}
	return r, services.NewPresentationsService(r)
}

func NewReceivingTasks(db *gorm.DB, notifSvc *services.NotificationsService) (ports.ReceivingTasksRepository, *services.ReceivingTasksService) {
	r := &repositories.ReceivingTasksRepository{DB: db, NotificationsSvc: notifSvc}
	return r, services.NewReceivingTasksService(r)
}

// NewSerials builds SerialsRepository and SerialsService. When pool is non-nil, uses SerialsRepositorySQLC.
func NewSerials(db *gorm.DB, pool *pgxpool.Pool) (ports.SerialsRepository, *services.SerialsService) {
	var r ports.SerialsRepository
	if pool != nil {
		queries := sqlc.New(pool)
		r = repositories.NewSerialsRepositorySQLC(queries)
	} else {
		r = &repositories.SerialsRepository{DB: db}
	}
	return r, services.NewSerialsService(r)
}

func NewStockAlerts(db *gorm.DB, redisClient *redis.Client) (ports.StockAlertsRepository, *services.StockAlertsService) {
	r := &repositories.StockAlertsRepository{DB: db, Redis: redisClient}
	return r, services.NewStockAlertsService(r)
}

func NewUsers(db *gorm.DB, config configuration.Config, notifSvc *services.NotificationsService) (ports.UsersRepository, *services.UserService) {
	r := &repositories.UsersRepository{DB: db, JWTSecret: config.JWTSecret, NotificationsSvc: notifSvc}
	return r, services.NewUserService(r)
}

// NewNotifications builds NotificationsRepository and NotificationsService.
func NewNotifications(db *gorm.DB, emailSender tools.EmailSender, tenantID string) (ports.NotificationsRepository, *services.NotificationsService) {
	r := &repositories.NotificationsRepository{DB: db}
	return r, services.NewNotificationsService(r, emailSender, tenantID)
}

// NewLocationTypes builds LocationTypesRepository and LocationTypesService. Requires pool (Postgres).
func NewLocationTypes(pool *pgxpool.Pool) (ports.LocationTypesRepository, *services.LocationTypesService) {
	if pool == nil {
		return nil, nil
	}
	queries := sqlc.New(pool)
	r := repositories.NewLocationTypesRepositorySQLC(queries)
	return r, services.NewLocationTypesService(r)
}

// NewPresentationTypes builds PresentationTypesRepository and PresentationTypesService. Requires pool (Postgres).
func NewPresentationTypes(pool *pgxpool.Pool) (ports.PresentationTypesRepository, *services.PresentationTypesService) {
	if pool == nil {
		return nil, nil
	}
	queries := sqlc.New(pool)
	r := repositories.NewPresentationTypesRepositorySQLC(queries)
	return r, services.NewPresentationTypesService(r)
}

// NewAdjustmentReasonCodes builds AdjustmentReasonCodesRepository and AdjustmentReasonCodesService. Requires pool (Postgres).
func NewAdjustmentReasonCodes(pool *pgxpool.Pool) (ports.AdjustmentReasonCodesRepository, *services.AdjustmentReasonCodesService) {
	if pool == nil {
		return nil, nil
	}
	queries := sqlc.New(pool)
	r := repositories.NewAdjustmentReasonCodesRepositorySQLC(queries)
	return r, services.NewAdjustmentReasonCodesService(r)
}

// NewPresentationConversions builds PresentationConversionsRepository and PresentationConversionsService. Requires pool (Postgres).
func NewPresentationConversions(pool *pgxpool.Pool) (ports.PresentationConversionsRepository, *services.PresentationConversionsService) {
	if pool == nil {
		return nil, nil
	}
	queries := sqlc.New(pool)
	r := repositories.NewPresentationConversionsRepositorySQLC(queries)
	return r, services.NewPresentationConversionsService(r)
}

// NewStockTransfers builds StockTransfersRepository and StockTransfersService. Requires pool (Postgres).
func NewStockTransfers(pool *pgxpool.Pool) (ports.StockTransfersRepository, *services.StockTransfersService) {
	if pool == nil {
		return nil, nil
	}
	queries := sqlc.New(pool)
	r := repositories.NewStockTransfersRepositorySQLC(queries)
	return r, services.NewStockTransfersService(r)
}

// NewUserPreferences builds UserPreferencesRepository. Returns nil if pool is nil (no Postgres).
func NewUserPreferences(pool *pgxpool.Pool) ports.UserPreferencesRepository {
	if pool == nil {
		return nil
	}
	queries := sqlc.New(pool)
	return repositories.NewUserPreferencesRepositorySQLC(queries)
}

// NewClients builds ClientsRepository and ClientsService. Requires pool (Postgres).
func NewClients(pool *pgxpool.Pool) (ports.ClientsRepository, *services.ClientsService) {
	if pool == nil {
		return nil, nil
	}
	queries := sqlc.New(pool)
	r := repositories.NewClientsRepositorySQLC(queries, pool)
	return r, services.NewClientsService(r)
}

// NewCategories builds CategoriesRepository and CategoriesService. Requires pool (Postgres).
func NewCategories(pool *pgxpool.Pool) (ports.CategoriesRepository, *services.CategoriesService) {
	if pool == nil {
		return nil, nil
	}
	queries := sqlc.New(pool)
	r := repositories.NewCategoriesRepositorySQLC(queries, pool)
	return r, services.NewCategoriesService(r)
}

// NewStockSettings builds StockSettingsRepository and StockSettingsService. Requires pool (Postgres).
func NewStockSettings(pool *pgxpool.Pool) (ports.StockSettingsRepository, *services.StockSettingsService) {
	if pool == nil {
		return nil, nil
	}
	queries := sqlc.New(pool)
	r := repositories.NewStockSettingsRepositorySQLC(queries)
	return r, services.NewStockSettingsService(r)
}

// NewPurchaseOrders builds PurchaseOrdersRepository and PurchaseOrdersService.
// Uses GORM (consistent with ReceivingTasksRepository and PickingTaskRepository).
func NewPurchaseOrders(db *gorm.DB) (ports.PurchaseOrdersRepository, *services.PurchaseOrdersService) {
	r := &repositories.PurchaseOrdersRepository{DB: db}
	return r, services.NewPurchaseOrdersService(r)
}

// NewSalesOrders builds SalesOrdersRepository and SalesOrdersService (S3-W2-B).
// Injects InventoryService for FEFO pick suggestions on submit.
func NewSalesOrders(db *gorm.DB, config configuration.Config) (ports.SalesOrdersRepository, *services.SalesOrdersService) {
	invRepo := &repositories.InventoryRepository{DB: db}
	invSvc := services.NewInventoryService(invRepo, nil)

	r := &repositories.SalesOrdersRepository{
		DB:           db,
		InventorySvc: invSvc,
	}
	return r, services.NewSalesOrdersService(r)
}

// NewDeliveryNotes builds DeliveryNotesRepository and DeliveryNotesService (S3-W3-A DN3).
func NewDeliveryNotes(db *gorm.DB) (ports.DeliveryNotesRepository, *services.DeliveryNotesService) {
	r := &repositories.DeliveryNotesRepository{DB: db}
	return r, services.NewDeliveryNotesService(r, db)
}

// NewBackorders builds BackordersRepository and BackordersService (S3-W3-A BO1+BO2).
// Injects InventoryService for FEFO pick suggestions on fulfill.
func NewBackorders(db *gorm.DB) (ports.BackordersRepository, *services.BackordersService) {
	invRepo := &repositories.InventoryRepository{DB: db}
	invSvc := services.NewInventoryService(invRepo, nil)
	r := &repositories.BackordersRepository{
		DB:           db,
		InventorySvc: invSvc,
	}
	return r, services.NewBackordersService(r)
}

// NewPickingTaskWithDN builds PickingTaskRepository with SO + DN PDF generator injected (S3-W3-A).
func NewPickingTaskWithDN(db *gorm.DB, auditSvc *services.AuditService, notifSvc *services.NotificationsService, soRepo repositories.SOPickedQtyUpdater) (ports.PickingTaskRepository, *services.PickingTaskService) {
	_, dnSvc := NewDeliveryNotes(db)
	r := &repositories.PickingTaskRepository{
		DB:               db,
		AuditService:     auditSvc,
		NotificationsSvc: notifSvc,
		SORepository:     soRepo,
		DNPDFGen:         dnSvc,
	}
	return r, services.NewPickingTaskService(r)
}
