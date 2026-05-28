package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterArticleSuppliersRoutes — CRUD for /api/article-suppliers/* (closes S3-5).
// Permissions piggyback on the "articles" resource (read/create/update/delete) — whoever
// manages the catalog manages its suppliers; avoids needing a new seeded permission resource.
func RegisterArticleSuppliersRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository) {
	if db == nil {
		return
	}
	_, svc := wire.NewArticleSuppliers(db)
	ctrl := controllers.NewArticleSuppliersController(svc, config.TenantID)

	route := router.Group("/article-suppliers")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "articles", "read")
		create := tools.RequirePermission(rolesRepo, "articles", "create")
		update := tools.RequirePermission(rolesRepo, "articles", "update")
		del := tools.RequirePermission(rolesRepo, "articles", "delete")

		route.GET("/", read, ctrl.List)
		route.GET("/:id", read, ctrl.GetByID)
		route.POST("/", create, ctrl.Create)
		route.PATCH("/:id", update, ctrl.Update)
		route.DELETE("/:id", del, ctrl.SoftDelete)
	}
}
