package repositories

import (
	"errors"
	"strings"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

type UsersRepository struct {
	DB *gorm.DB
}

func (u *UsersRepository) GetAllUsers() ([]database.User, *responses.InternalResponse) {
	var users []database.User

	err := u.DB.
		Table(database.User{}.TableName()).
		Order("created_at DESC").
		Find(&users).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch users",
			Handled: false,
		}
	}

	if len(users) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No users found",
			Handled: true,
		}
	}

	return users, nil
}

func (u *UsersRepository) GetUserByID(id string) (*database.User, *responses.InternalResponse) {
	var user database.User

	err := u.DB.First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "User not found",
			Handled: true,
		}
	}
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to find user",
			Handled: false,
		}
	}

	return &user, nil
}

func (u *UsersRepository) CreateUser(user *requests.User) *responses.InternalResponse {
	var existingByID database.User
	err := u.DB.First(&existingByID, "id = ?", user.ID).Error

	if err == nil {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "User ID already exists",
			Handled: true,
		}
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to check user ID",
			Handled: false,
		}
	}

	var count int64
	err = u.DB.Model(&database.User{}).Where("email = ?", user.Email).Count(&count).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to check email",
			Handled: false,
		}
	}
	if count > 0 {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Email address already exists",
			Handled: true,
		}
	}

	encryptedPassword, err := tools.Encrypt(*user.Password)
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to encrypt password",
			Handled: false,
		}
	}

	user.Password = &encryptedPassword

	var newUser database.User
	tools.CopyStructFields(user, &newUser)

	newUser.IsActive = true
	newUser.AuthProvider = "email"
	newUser.ResetToken = nil
	newUser.ResetTokenExpires = nil
	newUser.CreatedAt = tools.GetCurrentTime()
	newUser.UpdatedAt = tools.GetCurrentTime()

	err = u.DB.Table(database.User{}.TableName()).Create(&newUser).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to create user",
			Handled: false,
		}
	}

	return nil
}

func (u *UsersRepository) UpdateUser(id string, data map[string]interface{}) *responses.InternalResponse {
	var user database.User
	err := u.DB.First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "User not found",
			Handled: true,
		}
	}
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to find user",
			Handled: false,
		}
	}

	protectedFields := map[string]bool{
		"id":                  true,
		"password":            true,
		"auth_provider":       true,
		"reset_token":         true,
		"reset_token_expires": true,
		"created_at":          true,
	}

	for k := range protectedFields {
		delete(data, k)
	}

	data["updated_at"] = tools.GetCurrentTime()

	err = u.DB.Model(&user).Updates(data).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to update user",
			Handled: false,
		}
	}

	return nil
}

func (u *UsersRepository) DeleteUser(id string) *responses.InternalResponse {
	var user database.User

	err := u.DB.First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "User not found",
			Handled: true,
		}
	}
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to find user",
			Handled: false,
		}
	}

	err = u.DB.Delete(&user).Error
	if err != nil {
		if strings.Contains(err.Error(), "foreign key") {
			return &responses.InternalResponse{
				Error:   err,
				Message: "Cannot delete user due to existing relationships",
				Handled: true,
			}
		}

		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to delete user",
			Handled: false,
		}
	}

	return nil
}
