package controllers

import (
	"context"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
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

func (c *AuthenticationController) ForgotPassword(ctx *gin.Context) {
	var req requests.ForgotPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "ForgotPassword", "Formato inválido", "forgot_password")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "ForgotPassword", "forgot_password", errs)
		return
	}

	// Dispatch in background — prevents timing-based user enumeration and avoids blocking the response.
	// Use context.Background() so the job is not cancelled when the HTTP response is written.
	go func(email string) {
		if resp := c.Service.RequestPasswordReset(context.Background(), email); resp != nil && resp.Error != nil {
			log.Error().Err(resp.Error).Str("email", email).Msg("forgot password background error")
		}
	}(req.Email)

	tools.ResponseOK(ctx, "ForgotPassword",
		"Si el email existe, recibirás un enlace en los próximos minutos",
		"forgot_password", nil, false, "")
}

func (c *AuthenticationController) ResetPassword(ctx *gin.Context) {
	var req requests.ResetPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "ResetPassword", "Formato inválido", "reset_password")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "ResetPassword", "reset_password", errs)
		return
	}
	if resp := c.Service.ResetPassword(ctx.Request.Context(), req.Token, req.NewPassword); resp != nil {
		writeErrorResponse(ctx, "ResetPassword", "reset_password", resp)
		return
	}
	tools.ResponseOK(ctx, "ResetPassword", "Contraseña actualizada. Inicia sesión con tu nueva contraseña.", "reset_password", nil, false, "")
}
