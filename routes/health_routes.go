package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterHealthRoutes mounts health endpoints on the engine (no auth, no /api prefix).
// GET /health returns 200 and {"status":"ok"}.
// GET /health/detailed returns component status including database ping.
func RegisterHealthRoutes(r *gin.Engine, db *gorm.DB) {
	r.GET("/health", healthCheck)
	r.GET("/health/detailed", detailedHealthCheck(db))
}

func healthCheck(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func detailedHealthCheck(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		components := make(map[string]componentHealth)
		overallStatus := "healthy"

		dbHealth := componentHealth{Status: "healthy", Message: "Database connection is healthy"}
		sqlDB, err := db.DB()
		if err != nil {
			dbHealth.Status = "unhealthy"
			dbHealth.Message = fmt.Sprintf("Database access failed: %v", err)
			overallStatus = "unhealthy"
		} else if err := sqlDB.Ping(); err != nil {
			dbHealth.Status = "unhealthy"
			dbHealth.Message = fmt.Sprintf("Database ping failed: %v", err)
			overallStatus = "unhealthy"
		}
		components["database"] = dbHealth

		statusCode := http.StatusOK
		if overallStatus != "healthy" {
			statusCode = http.StatusServiceUnavailable
		}

		ctx.JSON(statusCode, detailedHealthResponse{
			Status:     overallStatus,
			Timestamp: time.Now().Format(time.RFC3339),
			Components: components,
		})
	}
}

type detailedHealthResponse struct {
	Status     string                     `json:"status"`
	Timestamp  string                    `json:"timestamp"`
	Components map[string]componentHealth `json:"components"`
}

type componentHealth struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}
