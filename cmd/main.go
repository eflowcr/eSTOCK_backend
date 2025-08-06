package main

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/routes"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

func main() {
	configuration.LoadConfig()

	db := tools.InitDB()

	r := gin.Default()

	r.Use(tools.CORSMiddleware())

	routes.RegisterRoutes(r, db)

	r.Run(":8080")
}
