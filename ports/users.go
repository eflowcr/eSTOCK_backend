package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// UsersRepository defines persistence operations for users.
type UsersRepository interface {
	GetAllUsers() ([]database.User, *responses.InternalResponse)
	GetUserByID(id string) (*database.User, *responses.InternalResponse)
	CreateUser(user *requests.User) *responses.InternalResponse
	UpdateUser(id string, data map[string]interface{}) *responses.InternalResponse
	DeleteUser(id string) *responses.InternalResponse
	ImportUsersFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse)
	ExportUsersToExcel() ([]byte, *responses.InternalResponse)
	UpdateUserPassword(id string, newPassword string) *responses.InternalResponse
	GenerateImportTemplate(language string) ([]byte, error)
}
