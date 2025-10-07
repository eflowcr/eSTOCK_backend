package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
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
		tools.Response(ctx, "GamificationStats", false, errResp.Message, "gamification_stats", nil, false, "", errResp.Handled)
		return
	}

	if stats == nil {
		tools.Response(ctx, "GamificationStats", false, "No se encontraron estadísticas de gamificación", "gamification_stats", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GamificationStats", true, "Estadísticas de gamificación obtenidas con éxito", "gamification_stats", stats, false, "", false)
}

func (c *GamificationController) Badges(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	badges, errResp := c.Service.Badges(userId)

	if errResp != nil {
		tools.Response(ctx, "Badges", false, errResp.Message, "badges", nil, false, "", errResp.Handled)
		return
	}

	if badges == nil {
		tools.Response(ctx, "Badges", false, "No se encontraron insignias", "badges", nil, false, "", false)
		return
	}

	tools.Response(ctx, "Badges", true, "Insignias obtenidas con éxito", "badges", badges, false, "", false)
}

func (c *GamificationController) GetAllBadges(ctx *gin.Context) {
	badges, errResp := c.Service.GetAllBadges()

	if errResp != nil {
		tools.Response(ctx, "GetAllBadges", false, errResp.Message, "all_badges", nil, false, "", errResp.Handled)
		return
	}

	if badges == nil {
		tools.Response(ctx, "GetAllBadges", false, "No se encontraron insignias", "all_badges", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetAllBadges", true, "Todas las insignias obtenidas con éxito", "all_badges", badges, false, "", false)
}

func (c *GamificationController) CompleteTasks(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, _ := tools.GetUserId(token)

	var task requests.CompleteTasks
	if err := ctx.ShouldBindJSON(&task); err != nil {
		tools.Response(ctx, "CompleteTasks", false, "Datos de solicitud no válidos", "complete_tasks", nil, false, "", false)
		return
	}

	tasks, errResp := c.Service.CompleteTasks(userId, task)

	if errResp != nil {
		tools.Response(ctx, "CompleteTasks", false, errResp.Message, "complete_tasks", nil, false, "", errResp.Handled)
		return
	}

	if tasks == nil {
		tools.Response(ctx, "CompleteTasks", false, "No se completaron tareas o no se otorgaron insignias", "complete_tasks", nil, false, "", false)
		return
	}

	tools.Response(ctx, "CompleteTasks", true, "Tareas completadas e insignias otorgadas con éxito", "complete_tasks", tasks, false, "", false)
}

func (c *GamificationController) GetAllUserStats(ctx *gin.Context) {
	stats, errResp := c.Service.GetAllUserStats()

	if errResp != nil {
		tools.Response(ctx, "GetAllUserStats", false, errResp.Message, "all_user_stats", nil, false, "", errResp.Handled)
		return
	}

	if stats == nil {
		tools.Response(ctx, "GetAllUserStats", false, "No se encontraron estadísticas de usuario", "all_user_stats", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetAllUserStats", true, "Todas las estadísticas de usuario obtenidas con éxito", "all_user_stats", stats, false, "", false)
}
