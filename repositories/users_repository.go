package repositories

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type UsersRepository struct {
	DB               *gorm.DB
	JWTSecret        string
	NotificationsSvc *services.NotificationsService // optional: emit user_welcome on create
}

func (u *UsersRepository) GetAllUsers() ([]database.User, *responses.InternalResponse) {
	var users []database.User

	err := u.DB.
		Table(database.User{}.TableName()).
		Preload("Role").
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

	err := u.DB.Preload("Role").First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &responses.InternalResponse{
			Message:    "Usuario no encontrado",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
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
			Message:    "El correo electrónico ya existe",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		}
	}

	encryptedPassword, err := tools.Encrypt(*user.Password, u.JWTSecret)
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
	if newUser.RoleID == "" {
		newUser.RoleID = "Operator"
	}
	// roles.id is nanoid; API may send name (Admin, Operator, Viewer) or id — resolve to role id
	if resolvedID := resolveRoleIDByName(u.DB, newUser.RoleID); resolvedID != "" {
		newUser.RoleID = resolvedID
	}
	newUser.ID = "" // Let DB generate id via DEFAULT nanoid()
	// name is required; derive from first_name + last_name or fallback to email
	name := strings.TrimSpace(newUser.FirstName + " " + newUser.LastName)
	if name == "" {
		name = newUser.Email
	}
	newUser.Name = name

	newUser.IsActive = true
	newUser.CreatedAt = tools.GetCurrentTime()
	newUser.UpdatedAt = tools.GetCurrentTime()

	err = u.DB.Table(database.User{}.TableName()).Omit("id").Create(&newUser).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al crear usuario",
			Handled: false,
		}
	}

	// Emit user_welcome notification (fire-and-forget via NotificationsService).
	// newUser.ID is populated by GORM after CREATE ... RETURNING id.
	if u.NotificationsSvc != nil && newUser.ID != "" {
		welcomeTitle := "¡Bienvenido a eSTOCK!"
		welcomeBody := fmt.Sprintf("Hola %s, tu cuenta ha sido creada.\nEmail: %s\n\nPor seguridad, solicita un restablecimiento de contraseña para establecer tu acceso.", newUser.Name, newUser.Email)
		_ = u.NotificationsSvc.Send(context.Background(), newUser.ID, "user_welcome",
			welcomeTitle, welcomeBody, "user", newUser.ID)
	}

	return nil
}

func (u *UsersRepository) UpdateUser(id string, data map[string]interface{}) *responses.InternalResponse {
	var user database.User
	err := u.DB.First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Message:    "Usuario no encontrado",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
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
		"id":         true,
		"password":   true,
		"created_at": true,
	}

	for k := range protectedFields {
		delete(data, k)
	}

	// Resolve role_id if sent as name (Admin, Operator, Viewer) or id
	if v, ok := data["role_id"].(string); ok && v != "" {
		if resolvedID := resolveRoleIDByName(u.DB, v); resolvedID != "" {
			data["role_id"] = resolvedID
		}
	}

	// Keep name in sync when first_name or last_name change
	_, hasFirst := data["first_name"]
	_, hasLast := data["last_name"]
	if hasFirst || hasLast {
		firstName := user.FirstName
		if v, ok := data["first_name"].(string); ok {
			firstName = v
		}
		lastName := user.LastName
		if v, ok := data["last_name"].(string); ok {
			lastName = v
		}
		name := strings.TrimSpace(firstName + " " + lastName)
		if name == "" {
			name = user.Email
		}
		data["name"] = name
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
			Message:    "User not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
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
				Message:    "No se puede eliminar el usuario debido a relaciones existentes",
				Handled:    true,
				StatusCode: responses.StatusConflict,
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
		roleID := strings.TrimSpace(row[5])

		if id == "" || email == "" || password == "" || roleID == "" {
			continue
		}

		user := &requests.User{
			ID:        id,
			Email:     email,
			FirstName: firstName,
			LastName:  lastName,
			Password:  &password,
			RoleID:    roleID,
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
		roleDisplay := user.RoleID
		if user.Role != nil {
			roleDisplay = user.Role.Name
		}
		values := []interface{}{
			user.ID,
			user.Email,
			user.FirstName,
			user.LastName,
			roleDisplay,
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
			Message:    "Usuario no encontrado",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al buscar usuario",
			Handled: false,
		}
	}

	hashedPassword, err := tools.Encrypt(plainPassword, u.JWTSecret)
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

// resolveRoleIDByName returns roles.id for the given name (case-insensitive) or id.
func resolveRoleIDByName(db *gorm.DB, roleIDOrName string) string {
	if roleIDOrName == "" {
		return ""
	}
	var r database.Role
	if err := db.Where("id = ? OR LOWER(name) = LOWER(?)", roleIDOrName, roleIDOrName).First(&r).Error; err != nil {
		return ""
	}
	return r.ID
}

func (u *UsersRepository) GenerateImportTemplate(language string) ([]byte, error) {
	isEs := language != "en"
	title := "Importar Usuarios"; subtitle := "Plantilla de importación — eSTOCK"
	instrTitle := "📋 Instrucciones"; instrContent := "1. Complete desde la fila 9  •  2. Email, Nombre, Apellido, Contraseña y Rol son obligatorios (*)  •  3. El Rol debe ser uno de la lista desplegable"
	if !isEs {
		title = "Import Users"; subtitle = "User import template — eSTOCK"
		instrTitle = "📋 Instructions"; instrContent = "1. Fill in data from row 9  •  2. Email, First Name, Last Name, Password and Role are required (*)  •  3. Role must be one from the dropdown list"
	}

	// Fetch role names from DB
	var roleNames []string
	u.DB.Table("roles").Pluck("name", &roleNames)
	if len(roleNames) == 0 {
		roleNames = []string{"Admin", "Operator", "Viewer"}
	}

	cfg := ModuleTemplateConfig{
		DataSheetName: func() string { if isEs { return "Usuarios" }; return "Users" }(),
		OptSheetName:  func() string { if isEs { return "Opciones" }; return "Options" }(),
		Title: title, Subtitle: subtitle, InstrTitle: instrTitle, InstrContent: instrContent,
		Columns: func() []ColumnDef {
			if isEs {
				return []ColumnDef{
					{Header: "Email *", Required: true, Width: 28},
					{Header: "Nombre *", Required: true, Width: 20},
					{Header: "Apellido *", Required: true, Width: 20},
					{Header: "Contraseña *", Required: true, Width: 18},
					{Header: "Rol *", Required: true, Width: 16},
				}
			}
			return []ColumnDef{
				{Header: "Email *", Required: true, Width: 28},
				{Header: "First Name *", Required: true, Width: 20},
				{Header: "Last Name *", Required: true, Width: 20},
				{Header: "Password *", Required: true, Width: 18},
				{Header: "Role *", Required: true, Width: 16},
			}
		}(),
		ExampleRow: []string{"user@company.com", "Ana", "García", "Passw0rd!", roleNames[0]},
		ApplyValidations: func(f *excelize.File, dataSheet, optSheet string, start, end int) error {
			f.NewSheet(optSheet)
			for i, v := range roleNames { cell, _ := excelize.CoordinatesToCellName(1, i+1); f.SetCellValue(optSheet, cell, v) }
			f.SetSheetVisible(optSheet, false)
			roleRef := "'" + optSheet + "'!$A$1:$A$" + fmt.Sprintf("%d", len(roleNames))
			errRole := func() string { if isEs { return "Rol inválido" }; return "Invalid role" }()
			return addDropListValidation(f, dataSheet, "E9:E2000", roleRef, errRole, errRole)
		},
	}
	return BuildModuleImportTemplate(cfg)
}
