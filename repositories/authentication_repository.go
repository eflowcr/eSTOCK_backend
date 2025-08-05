package repositories

import (
	"errors"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthenticationRepository struct {
	DB *gorm.DB
}

func (a *AuthenticationRepository) Login(email, password string) (*string, *responses.InternalResponse) {
	var user database.User

	err := a.DB.Where("email = ?", email).First(&user).Error
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

	if user.Password == nil || bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)) != nil {
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
