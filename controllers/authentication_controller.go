package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type AuthenticationController struct {
	Service services.AuthenticationService
}

func NewAuthenticationController(service services.AuthenticationService) *AuthenticationController {
	return &AuthenticationController{
		Service: service,
	}
}

func (c *AuthenticationController) Login(ctx *gin.Context) {
	var login requests.Login

	if err := ctx.ShouldBind(&login); err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid request data"})
		return
	}

	token, response := c.Service.Login(login)

	if response != nil {
		tools.Response(ctx, "Login", false, response.Message, "login", nil, false, "")
		return
	}

	tools.Response(ctx, "Login", true, "Login successful", "login", gin.H{"token": token}, false, "")
}
