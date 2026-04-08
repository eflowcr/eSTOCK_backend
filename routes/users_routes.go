package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var _ ports.UsersRepository = (*repositories.UsersRepository)(nil)

func RegisterUserRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	_, userService := wire.NewUsers(db, config)
	userController := controllers.NewUserController(*userService)

	protected := router.Group("/users")
	protected.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		protected.GET("/", userController.GetAllUsers)
		protected.GET("/:id", userController.GetUserByID)
		protected.POST("/", userController.CreateUser)
		protected.PUT("/:id", userController.UpdateUser)
		protected.DELETE("/:id", userController.DeleteUser)
		protected.GET("/import/template", userController.DownloadImportTemplate)
		protected.POST("/import", userController.ImportUsersFromExcel)
		protected.GET("/export", userController.ExportUsersToExcel)
		protected.PUT("/:id/password", userController.UpdateUserPassword)
	}
}
