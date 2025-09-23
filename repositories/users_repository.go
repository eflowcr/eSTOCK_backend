package repositories

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/xuri/excelize/v2"
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
			Message: "Error al obtener usuarios",
			Handled: false,
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
			Message: "Usuario no encontrado",
			Handled: true,
		}
	}
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al buscar usuario",
			Handled: false,
		}
	}

	return &user, nil
}

func (u *UsersRepository) CreateUser(user *requests.User) *responses.InternalResponse {
	user.ID = tools.GenerateGUID()

	var count int64
	err := u.DB.Model(&database.User{}).Where("email = ?", user.Email).Count(&count).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al verificar el correo electrónico",
			Handled: false,
		}
	}
	if count > 0 {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "El correo electrónico ya existe",
			Handled: true,
		}
	}

	encryptedPassword, err := tools.Encrypt(*user.Password)
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al encriptar la contraseña",
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
			Message: "Error al crear usuario",
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
			Message: "Usuario no encontrado",
			Handled: true,
		}
	}
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al buscar usuario",
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
			Message: "Error al actualizar usuario",
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
			Message: "Error al buscar usuario",
			Handled: false,
		}
	}

	err = u.DB.Delete(&user).Error
	if err != nil {
		if strings.Contains(err.Error(), "foreign key") {
			return &responses.InternalResponse{
				Error:   err,
				Message: "No se puede eliminar el usuario debido a relaciones existentes",
				Handled: true,
			}
		}

		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al eliminar usuario",
			Handled: false,
		}
	}

	return nil
}

func (u *UsersRepository) ImportUsersFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse) {
	imported := []string{}
	errorsList := []*responses.InternalResponse{}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		errorsList = append(errorsList, &responses.InternalResponse{
			Error:   err,
			Message: "Error al abrir el archivo de Excel",
			Handled: false,
		})
		return imported, errorsList
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		errorsList = append(errorsList, &responses.InternalResponse{
			Error:   err,
			Message: "Error al leer las filas",
			Handled: false,
		})
		return imported, errorsList
	}

	for i, row := range rows {
		if i < 6 {
			continue
		}

		if len(row) < 6 {
			continue
		}

		id := strings.TrimSpace(row[0])
		email := strings.TrimSpace(row[1])
		firstName := strings.TrimSpace(row[2])
		lastName := strings.TrimSpace(row[3])
		password := strings.TrimSpace(row[4])
		role := strings.TrimSpace(row[5])

		if id == "" || email == "" || password == "" || role == "" {
			continue
		}

		user := &requests.User{
			ID:        id,
			Email:     email,
			FirstName: firstName,
			LastName:  lastName,
			Password:  &password,
			Role:      role,
		}

		resp := u.CreateUser(user)
		if resp != nil {
			errorsList = append(errorsList, &responses.InternalResponse{
				Error:   resp.Error,
				Message: fmt.Sprintf("Row %d: %s", i+1, resp.Message),
				Handled: resp.Handled,
			})
			continue
		}

		imported = append(imported, id)
	}

	return imported, errorsList
}

func (u *UsersRepository) ExportUsersToExcel() ([]byte, *responses.InternalResponse) {
	users, errResp := u.GetAllUsers()
	if errResp != nil {
		return nil, errResp
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ID Usuario", "Email", "Nombre", "Apellido", "Rol"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 6)
		f.SetCellValue(sheet, cell, h)
	}

	for idx, user := range users {
		row := idx + 7
		values := []interface{}{
			user.ID,
			user.Email,
			user.FirstName,
			user.LastName,
			user.Role,
		}
		for col, val := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellValue(sheet, cell, val)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al generar el archivo de Excel",
			Handled: false,
		}
	}

	return buf.Bytes(), nil
}

func (u *UsersRepository) UpdateUserPassword(id string, plainPassword string) *responses.InternalResponse {
	var user database.User

	err := u.DB.First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Usuario no encontrado",
			Handled: true,
		}
	}
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al buscar usuario",
			Handled: false,
		}
	}

	hashedPassword, err := tools.Encrypt(plainPassword)
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al encriptar la contraseña",
			Handled: false,
		}
	}

	updateData := map[string]interface{}{
		"password":   hashedPassword,
		"updated_at": tools.GetCurrentTime(),
	}

	err = u.DB.Model(&user).Updates(updateData).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al actualizar la contraseña",
			Handled: false,
		}
	}

	return nil
}
