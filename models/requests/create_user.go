package requests

type User struct {
	ID              string  `gorm:"column:id;primaryKey" json:"id"`
	Email           string  `gorm:"column:email;unique" json:"email" validate:"required,email,max=255"`
	FirstName       string  `gorm:"column:first_name" json:"first_name" validate:"required,max=100"`
	LastName        string  `gorm:"column:last_name" json:"last_name" validate:"required,max=100"`
	ProfileImageURL *string `gorm:"column:profile_image_url" json:"profile_image_url"`
	Password        *string `gorm:"column:password" json:"password" validate:"required,min=6"`
	RoleID          string  `gorm:"column:role_id" json:"role_id" validate:"required,max=50"`
}
