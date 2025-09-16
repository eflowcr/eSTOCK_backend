package main

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/routes"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	configuration.LoadConfig()

	db := tools.InitDB()

	r := gin.New()

	r.Use(gin.Recovery())

	r.Use(tools.CORSMiddleware())

	routes.RegisterRoutes(r, db)

	r.Run(":8080")
}
