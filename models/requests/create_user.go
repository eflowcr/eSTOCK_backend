package requests

type User struct {
	ID              string  `gorm:"column:id;primaryKey" json:"id"`
	Email           string  `gorm:"column:email;unique" json:"email"`
	FirstName       string  `gorm:"column:first_name" json:"first_name"`
	LastName        string  `gorm:"column:last_name" json:"last_name"`
	ProfileImageURL *string `gorm:"column:profile_image_url" json:"profile_image_url"`
	Password        *string `gorm:"column:password" json:"password"`
	Role            string  `gorm:"column:role" json:"role"`
}
