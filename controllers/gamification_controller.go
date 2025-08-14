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

func (c *GamificationController) Badges(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	badges, errResp := c.Service.Badges(userId)

	if errResp != nil {
		tools.Response(ctx, "Badges", false, errResp.Message, "badges", nil, false, "")
		return
	}

	if badges == nil {
		tools.Response(ctx, "Badges", false, "No badges found", "badges", nil, false, "")
		return
	}

	tools.Response(ctx, "Badges", true, "Badges retrieved successfully", "badges", badges, false, "")
}

func (c *GamificationController) GetAllBadges(ctx *gin.Context) {
	badges, errResp := c.Service.GetAllBadges()

	if errResp != nil {
		tools.Response(ctx, "GetAllBadges", false, errResp.Message, "all_badges", nil, false, "")
		return
	}

	if badges == nil {
		tools.Response(ctx, "GetAllBadges", false, "No badges found", "all_badges", nil, false, "")
		return
	}

	tools.Response(ctx, "GetAllBadges", true, "All badges retrieved successfully", "all_badges", badges, false, "")
}
