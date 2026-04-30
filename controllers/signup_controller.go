package controllers

import (
	"net/http"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

// SignupController handles public self-service tenant signup endpoints.
type SignupController struct {
	Service *services.SignupService
}

// NewSignupController constructs SignupController.
func NewSignupController(svc *services.SignupService) *SignupController {
	return &SignupController{Service: svc}
}

// InitiateSignup handles POST /api/signup.
// Validates the request, stores a pending signup token, and sends a verification email.
// Returns 202 Accepted — the actual account creation happens on verify.
func (c *SignupController) InitiateSignup(ctx *gin.Context) {
	var req requests.SignupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "InitiateSignup", "Formato de solicitud inválido", "signup")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "InitiateSignup", "signup", errs)
		return
	}

	origin := ctx.GetHeader("Origin")
	if resp := c.Service.InitiateSignup(ctx.Request.Context(), req, origin); resp != nil {
		writeErrorResponse(ctx, "InitiateSignup", "signup", resp)
		return
	}

	// 202 Accepted — send a minimal response to avoid user enumeration.
	ctx.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"message": "Revisa tu correo electrónico para completar el registro",
	})
}

// VerifySignup handles POST /api/signup/verify.
// Validates the token, creates the tenant + admin user + demo data, and returns a JWT.
func (c *SignupController) VerifySignup(ctx *gin.Context) {
	var req requests.SignupVerifyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "VerifySignup", "Formato de solicitud inválido", "signup_verify")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "VerifySignup", "signup_verify", errs)
		return
	}

	result, resp := c.Service.VerifySignup(ctx.Request.Context(), req.Token)
	if resp != nil {
		writeErrorResponse(ctx, "VerifySignup", "signup_verify", resp)
		return
	}

	tools.ResponseCreated(ctx, "VerifySignup", "Cuenta creada con éxito. ¡Bienvenido a eSTOCK!", "signup_verify", result, true, result.Token)
}
