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
		tools.Response(ctx, "GetAllUsers", false, response.Message, "get_all_users", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "GetAllUsers", true, "Usuarios obtenidos con éxito", "get_all_users", users, false, "", false)
}

func (c *UserController) GetUserByID(ctx *gin.Context) {
	id := ctx.Param("id")
	user, response := c.Service.GetUserByID(id)

	if response != nil {
		tools.Response(ctx, "GetUserByID", false, response.Message, "get_user_by_id", nil, false, "", response.Handled)
		return
	}

	if user == nil {
		tools.Response(ctx, "GetUserByID", false, "Usuario no encontrado", "get_user_by_id", nil, false, "", false)
		return
	}

	tools.Response(ctx, "GetUserByID", true, "Usuario obtenido con éxito", "get_user_by_id", user, false, "", false)
}

func (c *UserController) CreateUser(ctx *gin.Context) {
	var user requests.User

	if err := ctx.ShouldBindJSON(&user); err != nil {
		tools.Response(ctx, "CreateUser", false, "Cuerpo de solicitud inválido", "create_user", nil, false, "", false)
		return
	}

	response := c.Service.CreateUser(&user)

	if response != nil {
		tools.Response(ctx, "CreateUser", false, response.Message, "create_user", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "CreateUser", true, "Usuario creado con éxito", "create_user", nil, false, "", false)
}

func (c *UserController) UpdateUser(ctx *gin.Context) {
	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.Response(ctx, "UpdateUser", false, "Cuerpo de solicitud inválido", "update_user", nil, false, "", false)
		return
	}

	id := ctx.Param("id")
	response := c.Service.UpdateUser(id, data)

	if response != nil {
		tools.Response(ctx, "UpdateUser", false, response.Message, "update_user", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "UpdateUser", true, "Usuario actualizado con éxito", "update_user", nil, false, "", false)
}

func (c *UserController) DeleteUser(ctx *gin.Context) {
	id := ctx.Param("id")
	response := c.Service.DeleteUser(id)

	if response != nil {
		tools.Response(ctx, "DeleteUser", false, response.Message, "delete_user", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "DeleteUser", true, "Usuario eliminado con éxito", "delete_user", nil, false, "", false)
}

func (c *UserController) ImportUsersFromExcel(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		tools.Response(ctx, "ImportUsersFromExcel", false, "Error al subir el archivo: "+err.Error(), "import_users_from_excel", nil, false, "", false)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		tools.Response(ctx, "ImportUsersFromExcel", false, "Error al abrir el archivo: "+err.Error(), "import_users_from_excel", nil, false, "", false)
		return
	}
	defer file.Close()

	// Leer archivo como []byte
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		tools.Response(ctx, "ImportUsersFromExcel", false, "Error al leer el contenido del archivo: "+err.Error(), "import_users_from_excel", nil, false, "", false)
		return
	}

	importedUsers, errorResponses := c.Service.ImportUsersFromExcel(fileBytes)

	if len(importedUsers) == 0 && len(errorResponses) > 0 {
		// Mostrar el primer error (puedes hacer un resumen si querés)
		resp := errorResponses[0]
		tools.Response(ctx, "ImportUsersFromExcel", false, resp.Message, "import_users_from_excel", nil, false, "", resp.Handled)
		return
	}

	tools.Response(ctx, "ImportUsersFromExcel", true, "Usuarios importados con éxito", "import_users_from_excel", gin.H{
		"imported_users": importedUsers,
		"errors":         errorResponses,
	}, false, "", false)
}

func (c *UserController) ExportUsersToExcel(ctx *gin.Context) {
	excel, response := c.Service.ExportUsersToExcel()
	if response != nil {
		tools.Response(ctx, "ExportUsersToExcel", false, response.Message, "export_users_to_excel", nil, false, "", response.Handled)
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename=users.xlsx")
	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excel)
	tools.Response(ctx, "ExportUsersToExcel", true, "Users exported successfully", "export_users_to_excel", nil, false, "", false)
}

func (c *UserController) UpdateUserPassword(ctx *gin.Context) {
	id := ctx.Param("id")
	password := ctx.PostForm("password")

	response := c.Service.UpdateUserPassword(id, password)
	if response != nil {
		tools.Response(ctx, "ChangePassword", false, response.Message, "change_password", nil, false, "", response.Handled)
		return
	}

	tools.Response(ctx, "ChangePassword", true, "Contraseña cambiada con éxito", "change_password", nil, true, "", false)
}
