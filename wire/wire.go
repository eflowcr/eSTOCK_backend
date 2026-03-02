// Package wire provides constructors that build repository + service per domain.
// Route handlers call wire.NewX(db) or wire.NewX(db, config) to get a service
// without constructing repos directly. Keeps route registration thin.
package wire

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"gorm.io/gorm"
)

func NewArticles(db *gorm.DB) (ports.ArticlesRepository, *services.ArticlesService) {
	r := &repositories.ArticlesRepository{DB: db}
	return r, services.NewArticlesService(r)
}

func NewAdjustments(db *gorm.DB) (ports.AdjustmentsRepository, *services.AdjustmentsService) {
	r := &repositories.AdjustmentsRepository{DB: db}
	return r, services.NewAdjustmentsService(r)
}

func NewAuthentication(db *gorm.DB, config configuration.Config) (ports.AuthenticationRepository, *services.AuthenticationService) {
	r := &repositories.AuthenticationRepository{DB: db, JWTSecret: config.JWTSecret}
	return r, services.NewAuthenticationService(r)
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

func NewInventory(db *gorm.DB) (ports.InventoryRepository, *services.InventoryService) {
	r := &repositories.InventoryRepository{DB: db}
	return r, services.NewInventoryService(r)
}

func NewInventoryMovements(db *gorm.DB) (ports.InventoryMovementsRepository, *services.InventoryMovementsService) {
	r := &repositories.InventoryMovementsRepository{DB: db}
	return r, services.NewInventoryMovementsService(r)
}

func NewLocations(db *gorm.DB) (ports.LocationsRepository, *services.LocationsService) {
	r := &repositories.LocationsRepository{DB: db}
	return r, services.NewLocationsService(r)
}

func NewLots(db *gorm.DB) (ports.LotsRepository, *services.LotsService) {
	r := &repositories.LotsRepository{DB: db}
	return r, services.NewLotsService(r)
}

func NewPickingTask(db *gorm.DB) (ports.PickingTaskRepository, *services.PickingTaskService) {
	r := &repositories.PickingTaskRepository{DB: db}
	return r, services.NewPickingTaskService(r)
}

func NewPresentations(db *gorm.DB) (ports.PresentationsRepository, *services.PresentationsService) {
	r := &repositories.PresentationsRepository{DB: db}
	return r, services.NewPresentationsService(r)
}

func NewReceivingTasks(db *gorm.DB) (ports.ReceivingTasksRepository, *services.ReceivingTasksService) {
	r := &repositories.ReceivingTasksRepository{DB: db}
	return r, services.NewReceivingTasksService(r)
}

func NewSerials(db *gorm.DB) (ports.SerialsRepository, *services.SerialsService) {
	r := &repositories.SerialsRepository{DB: db}
	return r, services.NewSerialsService(r)
}

func NewStockAlerts(db *gorm.DB) (ports.StockAlertsRepository, *services.StockAlertsService) {
	r := &repositories.StockAlertsRepository{DB: db}
	return r, services.NewStockAlertsService(r)
}

func NewUsers(db *gorm.DB, config configuration.Config) (ports.UsersRepository, *services.UserService) {
	r := &repositories.UsersRepository{DB: db, JWTSecret: config.JWTSecret}
	return r, services.NewUserService(r)
}
