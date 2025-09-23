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

func (a *AuthenticationRepository) Login(login requests.Login) (*responses.LoginResponse, *responses.InternalResponse) {
	var user database.User

	err := a.DB.Where("email = ?", login.Email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &responses.InternalResponse{
			Error:   errors.New("usuario no encontrado"),
			Message: "Credenciales inválidas",
			Handled: true,
		}
	}

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener el usuario",
			Handled: false,
		}
	}

	if !user.IsActive {
		return nil, &responses.InternalResponse{
			Error:   errors.New("cuenta inactiva"),
			Message: "Su cuenta está inactiva. Por favor, contacte al administrador.",
			Handled: true,
		}
	}

	if user.Password == nil || !tools.ComparePasswords(*user.Password, login.Password) {
		return nil, &responses.InternalResponse{
			Error:   errors.New("contraseña inválida"),
			Message: "Credenciales inválidas",
			Handled: true,
		}
	}

	token, err := tools.GenerateToken(user.ID, user.FirstName+" "+user.LastName, user.Email, user.Role)
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al generar el token",
			Handled: false,
		}
	}

	return &responses.LoginResponse{
		Name:     user.FirstName,
		LastName: user.LastName,
		Email:    user.Email,
		Token:    token,
		Role:     user.Role,
	}, nil
}
