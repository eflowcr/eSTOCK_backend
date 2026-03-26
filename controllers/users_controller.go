package controllers

import (
	"io"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type UserController struct {
	Service services.UserService
}

func NewUserController(service services.UserService) *UserController {
	return &UserController{
		Service: service,
	}
}

func (c *UserController) GetAllUsers(ctx *gin.Context) {
	users, response := c.Service.GetAllUsers()

	if response != nil {
		writeErrorResponse(ctx, "GetAllUsers", "get_all_users", response)
		return
	}

	tools.ResponseOK(ctx, "GetAllUsers", "Usuarios obtenidos con éxito", "get_all_users", users, false, "")
}

func (c *UserController) GetUserByID(ctx *gin.Context) {
	id := ctx.Param("id")
	user, response := c.Service.GetUserByID(id)

	if response != nil {
		writeErrorResponse(ctx, "GetUserByID", "get_user_by_id", response)
		return
	}

	if user == nil {
		tools.ResponseNotFound(ctx, "GetUserByID", "Usuario no encontrado", "get_user_by_id")
		return
	}

	tools.ResponseOK(ctx, "GetUserByID", "Usuario obtenido con éxito", "get_user_by_id", user, false, "")
}

func (c *UserController) CreateUser(ctx *gin.Context) {
	var user requests.User

	if err := ctx.ShouldBindJSON(&user); err != nil {
		tools.ResponseBadRequest(ctx, "CreateUser", "Cuerpo de solicitud inválido", "create_user")
		return
	}
	if errs := tools.ValidateStruct(&user); errs != nil {
		tools.ResponseValidationError(ctx, "CreateUser", "create_user", errs)
		return
	}

	response := c.Service.CreateUser(&user)

	if response != nil {
		writeErrorResponse(ctx, "CreateUser", "create_user", response)
		return
	}

	tools.ResponseCreated(ctx, "CreateUser", "Usuario creado con éxito", "create_user", nil, false, "")
}

func (c *UserController) UpdateUser(ctx *gin.Context) {
	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.ResponseBadRequest(ctx, "UpdateUser", "Cuerpo de solicitud inválido", "update_user")
		return
	}

	id := ctx.Param("id")
	response := c.Service.UpdateUser(id, data)

	if response != nil {
		writeErrorResponse(ctx, "UpdateUser", "update_user", response)
		return
	}

	tools.ResponseOK(ctx, "UpdateUser", "Usuario actualizado con éxito", "update_user", nil, false, "")
}

func (c *UserController) DeleteUser(ctx *gin.Context) {
	id := ctx.Param("id")
	response := c.Service.DeleteUser(id)

	if response != nil {
		writeErrorResponse(ctx, "DeleteUser", "delete_user", response)
		return
	}

	tools.ResponseOK(ctx, "DeleteUser", "Usuario eliminado con éxito", "delete_user", nil, false, "")
}

func (c *UserController) ImportUsersFromExcel(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportUsersFromExcel", "Error al subir el archivo", "import_users_from_excel")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportUsersFromExcel", "Error al abrir el archivo", "import_users_from_excel")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.ResponseBadRequest(ctx, "ImportUsersFromExcel", "Error al leer el contenido del archivo", "import_users_from_excel")
		return
	}

	importedUsers, errorResponses := c.Service.ImportUsersFromExcel(fileBytes)

	if len(importedUsers) == 0 && len(errorResponses) > 0 {
		resp := errorResponses[0]
		writeErrorResponse(ctx, "ImportUsersFromExcel", "import_users_from_excel", resp)
		return
	}

	tools.ResponseOK(ctx, "ImportUsersFromExcel", "Usuarios importados con éxito", "import_users_from_excel", gin.H{
		"imported_users": importedUsers,
		"errors":         errorResponses,
	}, false, "")
}

func (c *UserController) DownloadImportTemplate(ctx *gin.Context) {
	lang := ctx.DefaultQuery("lang", "es")
	data, err := c.Service.GenerateImportTemplate(lang)
	if err != nil {
		tools.ResponseBadRequest(ctx, "DownloadImportTemplate", "Error al generar la plantilla", "download_import_template")
		return
	}
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", `attachment; filename="ImportUsers.xlsx"`)
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func (c *UserController) ExportUsersToExcel(ctx *gin.Context) {
	excel, response := c.Service.ExportUsersToExcel()
	if response != nil {
		writeErrorResponse(ctx, "ExportUsersToExcel", "export_users_to_excel", response)
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename=users.xlsx")
	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excel)
}

func (c *UserController) UpdateUserPassword(ctx *gin.Context) {
	id := ctx.Param("id")
	password := ctx.PostForm("password")

	response := c.Service.UpdateUserPassword(id, password)
	if response != nil {
		writeErrorResponse(ctx, "ChangePassword", "change_password", response)
		return
	}

	tools.ResponseOK(ctx, "ChangePassword", "Contraseña cambiada con éxito", "change_password", nil, true, "")
}
