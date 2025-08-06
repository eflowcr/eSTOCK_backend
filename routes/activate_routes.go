package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	api := r.Group("/api")

	RegisterAuthenticationRoutes(api, db)
	RegisterEncryptionRoutes(api)
}
