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
	r := &repositories.AuthenticationRepository{DB: db, JWTSecret: config.JWTSecret}
	return r, services.NewAuthenticationService(r, nil)
}

// NewAuthenticationWithRoles builds the auth service with roles repo so login response includes permissions.
func NewAuthenticationWithRoles(db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository) (ports.AuthenticationRepository, *services.AuthenticationService) {
	r := &repositories.AuthenticationRepository{DB: db, JWTSecret: config.JWTSecret}
	return r, services.NewAuthenticationService(r, rolesRepo)
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
func NewInventory(db *gorm.DB, pool *pgxpool.Pool) (ports.InventoryRepository, *services.InventoryService) {
	r := &repositories.InventoryRepository{DB: db}
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
		r = repositories.NewLotsRepositorySQLC(queries)
		articlesRepo, _ = NewArticles(db, pool)
	} else {
		r = &repositories.LotsRepository{DB: db}
	}
	return r, services.NewLotsService(r, articlesRepo)
}

func NewPickingTask(db *gorm.DB) (ports.PickingTaskRepository, *services.PickingTaskService) {
	r := &repositories.PickingTaskRepository{DB: db}
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

func NewReceivingTasks(db *gorm.DB) (ports.ReceivingTasksRepository, *services.ReceivingTasksService) {
	r := &repositories.ReceivingTasksRepository{DB: db}
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

func NewUsers(db *gorm.DB, config configuration.Config) (ports.UsersRepository, *services.UserService) {
	r := &repositories.UsersRepository{DB: db, JWTSecret: config.JWTSecret}
	return r, services.NewUserService(r)
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
