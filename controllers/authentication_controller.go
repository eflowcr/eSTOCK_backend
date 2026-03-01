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
		tools.ResponseBadRequest(ctx, "Login", "Carga útil incorrecta", "login")
		return
	}
	if errs := tools.ValidateStruct(&login); errs != nil {
		tools.ResponseValidationError(ctx, "Login", "login", errs)
		return
	}

	loginResponse, response := c.Service.Login(login)

	if response != nil {
		writeErrorResponse(ctx, "Login", "login", response)
		return
	}

	tools.ResponseOK(ctx, "Login", "Login exitoso", "login", loginResponse, true, loginResponse.Token)
}
