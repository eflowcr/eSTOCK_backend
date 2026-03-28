package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type GamificationController struct {
	Service   services.GamificationService
	JWTSecret string
}

func NewGamificationController(service services.GamificationService, jwtSecret string) *GamificationController {
	return &GamificationController{
		Service:   service,
		JWTSecret: jwtSecret,
	}
}

func (c *GamificationController) GamificationStats(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, userIdErr := tools.GetUserId(c.JWTSecret, token)
	if userIdErr != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	stats, errResp := c.Service.GamificationStats(userId)

	if errResp != nil {
		writeErrorResponse(ctx, "GamificationStats", "gamification_stats", errResp)
		return
	}

	if stats == nil {
		tools.ResponseNotFound(ctx, "GamificationStats", "No se encontraron estadísticas de gamificación", "gamification_stats")
		return
	}

	tools.ResponseOK(ctx, "GamificationStats", "Estadísticas de gamificación obtenidas con éxito", "gamification_stats", stats, false, "")
}

func (c *GamificationController) Badges(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, userIdErr := tools.GetUserId(c.JWTSecret, token)
	if userIdErr != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	badges, errResp := c.Service.Badges(userId)

	if errResp != nil {
		writeErrorResponse(ctx, "Badges", "badges", errResp)
		return
	}

	if badges == nil {
		tools.ResponseNotFound(ctx, "Badges", "No se encontraron insignias", "badges")
		return
	}

	tools.ResponseOK(ctx, "Badges", "Insignias obtenidas con éxito", "badges", badges, false, "")
}

func (c *GamificationController) GetAllBadges(ctx *gin.Context) {
	badges, errResp := c.Service.GetAllBadges()

	if errResp != nil {
		writeErrorResponse(ctx, "GetAllBadges", "all_badges", errResp)
		return
	}

	if badges == nil {
		tools.ResponseOK(ctx, "GetAllBadges", "No se encontraron insignias", "all_badges", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "GetAllBadges", "Todas las insignias obtenidas con éxito", "all_badges", badges, false, "")
}

func (c *GamificationController) CompleteTasks(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	userId, userIdErr := tools.GetUserId(c.JWTSecret, token)
	if userIdErr != nil {
		tools.ResponseUnauthorized(ctx, "GetUserId", "Token inválido", "invalid_token")
		return
	}

	var task requests.CompleteTasks
	if err := ctx.ShouldBindJSON(&task); err != nil {
		tools.ResponseBadRequest(ctx, "CompleteTasks", "Datos de solicitud no válidos", "complete_tasks")
		return
	}
	if errs := tools.ValidateStruct(&task); errs != nil {
		tools.ResponseValidationError(ctx, "CompleteTasks", "complete_tasks", errs)
		return
	}

	tasks, errResp := c.Service.CompleteTasks(userId, task)

	if errResp != nil {
		writeErrorResponse(ctx, "CompleteTasks", "complete_tasks", errResp)
		return
	}

	if tasks == nil {
		tools.ResponseOK(ctx, "CompleteTasks", "No se completaron tareas o no se otorgaron insignias", "complete_tasks", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "CompleteTasks", "Tareas completadas e insignias otorgadas con éxito", "complete_tasks", tasks, false, "")
}

func (c *GamificationController) GetAllUserStats(ctx *gin.Context) {
	stats, errResp := c.Service.GetAllUserStats()

	if errResp != nil {
		writeErrorResponse(ctx, "GetAllUserStats", "all_user_stats", errResp)
		return
	}

	if stats == nil {
		tools.ResponseOK(ctx, "GetAllUserStats", "No se encontraron estadísticas de usuario", "all_user_stats", nil, false, "")
		return
	}

	tools.ResponseOK(ctx, "GetAllUserStats", "Todas las estadísticas de usuario obtenidas con éxito", "all_user_stats", stats, false, "")
}
