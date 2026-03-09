package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

// PreferencesController handles GET/PUT /api/user/preferences. Requires JWT auth.
type PreferencesController struct {
	Repo ports.UserPreferencesRepository
}

// NewPreferencesController returns a new PreferencesController.
func NewPreferencesController(repo ports.UserPreferencesRepository) *PreferencesController {
	return &PreferencesController{Repo: repo}
}

// preferencesResponse is the JSON shape for GET/PUT (matches backend_template).
type preferencesResponse struct {
	Theme    string `json:"theme"`
	Language string `json:"language"`
	Notifications struct {
		Email     bool `json:"email"`
		Push      bool `json:"push"`
		Marketing bool `json:"marketing"`
	} `json:"notifications"`
	Privacy struct {
		ProfileVisibility string `json:"profile_visibility"`
		DataSharing       bool   `json:"data_sharing"`
	} `json:"privacy"`
}

// updatePreferencesRequest is the body for PUT.
type updatePreferencesRequest struct {
	Theme    string `json:"theme"`
	Language string `json:"language"`
	Notifications struct {
		Email     bool `json:"email"`
		Push      bool `json:"push"`
		Marketing bool `json:"marketing"`
	} `json:"notifications"`
	Privacy struct {
		ProfileVisibility string `json:"profile_visibility"`
		DataSharing       bool   `json:"data_sharing"`
	} `json:"privacy"`
}

func toResponse(p *ports.PreferencesEntry) preferencesResponse {
	if p == nil {
		return preferencesResponse{}
	}
	return preferencesResponse{
		Theme:    p.Theme,
		Language: p.Language,
		Notifications: struct {
			Email     bool `json:"email"`
			Push      bool `json:"push"`
			Marketing bool `json:"marketing"`
		}{
			Email:     p.EmailNotifications,
			Push:      p.PushNotifications,
			Marketing: p.MarketingNotifications,
		},
		Privacy: struct {
			ProfileVisibility string `json:"profile_visibility"`
			DataSharing       bool   `json:"data_sharing"`
		}{
			ProfileVisibility: p.ProfileVisibility,
			DataSharing:       p.DataSharing,
		},
	}
}

func applyDefaults(req *updatePreferencesRequest) {
	if req.Theme == "" {
		req.Theme = "system"
	}
	if req.Language == "" {
		req.Language = "en"
	}
	if req.Privacy.ProfileVisibility == "" {
		req.Privacy.ProfileVisibility = "private"
	}
}

// GetPreferences handles GET /api/user/preferences. Creates defaults if missing.
func (c *PreferencesController) GetPreferences(ctx *gin.Context) {
	uid := ctx.GetString(tools.ContextKeyUserID)
	if uid == "" {
		tools.ResponseUnauthorized(ctx, "GetPreferences", "Usuario no identificado", "get_preferences")
		return
	}
	if c.Repo == nil {
		tools.ResponseInternal(ctx, "GetPreferences", "Preferencias no disponibles", "get_preferences")
		return
	}
	prefs, err := c.Repo.GetUserPreferences(ctx.Request.Context(), uid)
	if err != nil {
		tools.ResponseInternal(ctx, "GetPreferences", "Error al obtener preferencias", "get_preferences")
		return
	}
	if prefs == nil {
		prefs, err = c.Repo.GetOrCreateUserPreferences(ctx.Request.Context(), uid)
		if err != nil {
			tools.ResponseInternal(ctx, "GetPreferences", "Error al crear preferencias por defecto", "get_preferences")
			return
		}
	}
	tools.ResponseOK(ctx, "GetPreferences", "Preferencias obtenidas", "get_preferences", toResponse(prefs), false, "")
}

// UpdatePreferences handles PUT /api/user/preferences.
func (c *PreferencesController) UpdatePreferences(ctx *gin.Context) {
	uid := ctx.GetString(tools.ContextKeyUserID)
	if uid == "" {
		tools.ResponseUnauthorized(ctx, "UpdatePreferences", "Usuario no identificado", "update_preferences")
		return
	}
	var req updatePreferencesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "UpdatePreferences", "Cuerpo inválido", "update_preferences")
		return
	}
	applyDefaults(&req)
	if c.Repo == nil {
		tools.ResponseInternal(ctx, "UpdatePreferences", "Preferencias no disponibles", "update_preferences")
		return
	}
	// Ensure preferences exist before update
	_, err := c.Repo.GetOrCreateUserPreferences(ctx.Request.Context(), uid)
	if err != nil {
		tools.ResponseInternal(ctx, "UpdatePreferences", "Error al crear preferencias", "update_preferences")
		return
	}
	prefs, err := c.Repo.UpdateUserPreferences(ctx.Request.Context(), ports.UpdatePreferencesParams{
		UserID:                 uid,
		Theme:                  req.Theme,
		Language:               req.Language,
		EmailNotifications:     req.Notifications.Email,
		PushNotifications:      req.Notifications.Push,
		MarketingNotifications: req.Notifications.Marketing,
		ProfileVisibility:      req.Privacy.ProfileVisibility,
		DataSharing:            req.Privacy.DataSharing,
	})
	if err != nil {
		tools.ResponseInternal(ctx, "UpdatePreferences", "Error al actualizar preferencias", "update_preferences")
		return
	}
	tools.ResponseOK(ctx, "UpdatePreferences", "Preferencias actualizadas", "update_preferences", toResponse(prefs), false, "")
}
