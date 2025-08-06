package repositories

import (
	"errors"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

type AuthenticationRepository struct {
	DB *gorm.DB
}

func (a *AuthenticationRepository) Login(login requests.Login) (*string, *responses.InternalResponse) {
	var user database.User

	err := a.DB.Where("email = ?", login.Email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &responses.InternalResponse{
			Error:   errors.New("user not found"),
			Message: "Invalid credentials",
			Handled: true,
		}
	}

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch user",
			Handled: false,
		}
	}

	if !user.IsActive {
		return nil, &responses.InternalResponse{
			Error:   errors.New("inactive account"),
			Message: "Your account is inactive",
			Handled: true,
		}
	}

	if user.Password == nil || !tools.ComparePasswords(*user.Password, login.Password) {
		return nil, &responses.InternalResponse{
			Error:   errors.New("invalid password"),
			Message: "Invalid credentials",
			Handled: true,
		}
	}

	token, err := tools.GenerateToken(user.ID, user.FirstName+" "+user.LastName, user.Email, user.Role)
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to generate token",
			Handled: false,
		}
	}

	return &token, nil
}
