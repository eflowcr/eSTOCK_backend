package controllers

import (
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
		tools.Response(ctx, "GetAllUsers", false, response.Message, "get_all_users", nil, false, "")
		return
	}

	tools.Response(ctx, "GetAllUsers", true, "Users retrieved successfully", "get_all_users", gin.H{"users": users}, false, "")
}

func (c *UserController) GetUserByID(ctx *gin.Context) {
	id := ctx.Param("id")
	user, response := c.Service.GetUserByID(id)

	if response != nil {
		tools.Response(ctx, "GetUserByID", false, response.Message, "get_user_by_id", nil, false, "")
		return
	}

	if user == nil {
		tools.Response(ctx, "GetUserByID", false, "User not found", "get_user_by_id", nil, false, "")
		return
	}

	tools.Response(ctx, "GetUserByID", true, "User retrieved successfully", "get_user_by_id", gin.H{"user": user}, false, "")
}

func (c *UserController) CreateUser(ctx *gin.Context) {
	var user requests.User

	if err := ctx.ShouldBindJSON(&user); err != nil {
		tools.Response(ctx, "CreateUser", false, "Invalid request body", "create_user", nil, false, "")
		return
	}

	response := c.Service.CreateUser(&user)

	if response != nil {
		tools.Response(ctx, "CreateUser", false, response.Message, "create_user", nil, false, "")
		return
	}

	tools.Response(ctx, "CreateUser", true, "User created successfully", "create_user", gin.H{"user": user}, false, "")
}

func (c *UserController) UpdateUser(ctx *gin.Context) {
	var data map[string]interface{}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		tools.Response(ctx, "UpdateUser", false, "Invalid request body", "update_user", nil, false, "")
		return
	}

	id := ctx.Param("id")
	response := c.Service.UpdateUser(id, data)

	if response != nil {
		tools.Response(ctx, "UpdateUser", false, response.Message, "update_user", nil, false, "")
		return
	}

	tools.Response(ctx, "UpdateUser", true, "User updated successfully", "update_user", nil, false, "")
}

func (c *UserController) DeleteUser(ctx *gin.Context) {
	id := ctx.Param("id")
	response := c.Service.DeleteUser(id)

	if response != nil {
		tools.Response(ctx, "DeleteUser", false, response.Message, "delete_user", nil, false, "")
		return
	}

	tools.Response(ctx, "DeleteUser", true, "User deleted successfully", "delete_user", nil, false, "")
}
