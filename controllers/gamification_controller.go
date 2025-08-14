package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type GamificationController struct {
	Service services.GamificationService
}

func NewGamificationController(service services.GamificationService) *GamificationController {
	return &GamificationController{
		Service: service,
	}
}

func (c *GamificationController) GamificationStats(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	stats, errResp := c.Service.GamificationStats(userId)

	if errResp != nil {
		tools.Response(ctx, "GamificationStats", false, errResp.Message, "gamification_stats", nil, false, "")
		return
	}

	if stats == nil {
		tools.Response(ctx, "GamificationStats", false, "No gamification stats found", "gamification_stats", nil, false, "")
		return
	}

	tools.Response(ctx, "GamificationStats", true, "Gamification stats retrieved successfully", "gamification_stats", stats, false, "")
}
