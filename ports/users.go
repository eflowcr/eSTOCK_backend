package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// UsersRepository defines persistence operations for users.
//
// S3.5 W5.5 (HR-S3.5 C2): CreateUser and ImportUsersFromExcel now require tenantID
// because the users table has a NOT NULL tenant_id column. Controllers source it from
// the JWT (TenantIDFromContext) so admins only create users inside their own tenant.
type UsersRepository interface {
	GetAllUsers() ([]database.User, *responses.InternalResponse)
	GetUserByID(id string) (*database.User, *responses.InternalResponse)
	CreateUser(tenantID string, user *requests.User) *responses.InternalResponse
	UpdateUser(id string, data map[string]interface{}) *responses.InternalResponse
	DeleteUser(id string) *responses.InternalResponse
	ImportUsersFromExcel(tenantID string, fileBytes []byte) ([]string, []*responses.InternalResponse)
	ExportUsersToExcel() ([]byte, *responses.InternalResponse)
	UpdateUserPassword(id string, newPassword string) *responses.InternalResponse
	GenerateImportTemplate(language string) ([]byte, error)
}
