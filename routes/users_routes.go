package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterUserRoutes(router *gin.RouterGroup, db *gorm.DB) {
	userRepository := &repositories.UsersRepository{DB: db}
	userService := services.NewUserService(userRepository)

	userController := controllers.NewUserController(*userService)

	route := router.Group("/users")
	{
		route.GET("/", userController.GetAllUsers)
		route.GET("/:id", userController.GetUserByID)
		route.POST("/", userController.CreateUser)
		route.PUT("/:id", userController.UpdateUser)
		route.DELETE("/:id", userController.DeleteUser)
	}
}
