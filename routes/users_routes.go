package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterUserRoutes(router *gin.RouterGroup, db *gorm.DB) {
	userRepository := &repositories.UsersRepository{DB: db}
	userService := services.NewUserService(userRepository)

	userController := controllers.NewUserController(*userService)

	public := router.Group("/users")
	public.POST("/register", userController.CreateUser)

	protected := router.Group("/users")
	protected.Use(tools.JWTAuthMiddleware())
	{
		protected.GET("/", userController.GetAllUsers)
		protected.GET("/:id", userController.GetUserByID)
		protected.POST("/", userController.CreateUser)
		protected.PUT("/:id", userController.UpdateUser)
		protected.DELETE("/:id", userController.DeleteUser)
		protected.POST("/import", userController.ImportUsersFromExcel)
		protected.GET("/export", userController.ExportUsersToExcel)
		protected.PUT("/:id/:password", userController.UpdateUserPassword)
	}
}
