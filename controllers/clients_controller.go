package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type ClientsController struct {
	Service  services.ClientsService
	TenantID string
}

func NewClientsController(service services.ClientsService, tenantID string) *ClientsController {
	return &ClientsController{Service: service, TenantID: tenantID}
}

func (c *ClientsController) List(ctx *gin.Context) {
	var clientType *string
	if t := ctx.Query("type"); t != "" {
		clientType = &t
	}
	var isActive *bool
	if a := ctx.Query("is_active"); a != "" {
		v := a == "true"
		isActive = &v
	}
	var search *string
	if s := ctx.Query("search"); s != "" {
		search = &s
	}

	clients, resp := c.Service.List(c.resolveTenantID(ctx), clientType, isActive, search)
	if resp != nil {
		writeErrorResponse(ctx, "ListClients", "list_clients", resp)
		return
	}
	tools.ResponseOK(ctx, "ListClients", "Clientes recuperados", "list_clients", clients, false, "")
}

func (c *ClientsController) GetByID(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "GetClientByID", "get_client", "ID de cliente inválido")
	if !ok {
		return
	}
	// HR1-M3: use tenant-scoped lookup to prevent cross-tenant client enumeration.
	client, resp := c.Service.GetByIDForTenant(id, c.resolveTenantID(ctx))
	if resp != nil {
		writeErrorResponse(ctx, "GetClientByID", "get_client", resp)
		return
	}
	tools.ResponseOK(ctx, "GetClientByID", "Cliente recuperado", "get_client", client, false, "")
}

func (c *ClientsController) Create(ctx *gin.Context) {
	var req requests.CreateClientRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "CreateClient", "Datos de solicitud inválidos", "create_client")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "CreateClient", "create_client", errs)
		return
	}

	userID, _ := ctx.Get(tools.ContextKeyUserID)
	var createdBy *string
	if uid, ok := userID.(string); ok && uid != "" {
		createdBy = &uid
	}

	client, resp := c.Service.Create(c.resolveTenantID(ctx), &req, createdBy)
	if resp != nil {
		writeErrorResponse(ctx, "CreateClient", "create_client", resp)
		return
	}
	tools.ResponseCreated(ctx, "CreateClient", "Cliente creado exitosamente", "create_client", client, false, "")
}

func (c *ClientsController) Update(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "UpdateClient", "update_client", "ID de cliente inválido")
	if !ok {
		return
	}
	var req requests.UpdateClientRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateClient", "Datos de solicitud inválidos", "update_client")
		return
	}
	if errs := tools.ValidateStruct(&req); errs != nil {
		tools.ResponseValidationError(ctx, "UpdateClient", "update_client", errs)
		return
	}

	// HR1-M3: tenantID is already threaded through Update.
	client, resp := c.Service.Update(id, &req, c.resolveTenantID(ctx))
	if resp != nil {
		writeErrorResponse(ctx, "UpdateClient", "update_client", resp)
		return
	}
	tools.ResponseOK(ctx, "UpdateClient", "Cliente actualizado", "update_client", client, false, "")
}

func (c *ClientsController) SoftDelete(ctx *gin.Context) {
	id, ok := tools.ParseRequiredParam(ctx, "id", "DeleteClient", "delete_client", "ID de cliente inválido")
	if !ok {
		return
	}
	// HR1-M3: pass tenantID to prevent cross-tenant soft-delete.
	if resp := c.Service.SoftDelete(id, c.resolveTenantID(ctx)); resp != nil {
		writeErrorResponse(ctx, "DeleteClient", "delete_client", resp)
		return
	}
	tools.ResponseOK(ctx, "DeleteClient", "Cliente eliminado", "delete_client", nil, false, "")
}

// resolveTenantID — S3.5 W5.5 (HR-S3.5 C1): JWT-first, env fallback only.
// The TenantID field stays as a non-JWT fallback (cron/admin/test paths only).
func (c *ClientsController) resolveTenantID(ctx *gin.Context) string {
	return tools.ResolveTenantID(ctx, c.TenantID)
}
