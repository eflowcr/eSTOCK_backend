package tools

import (
	"gorm.io/gorm"
)

// GenerateNanoid returns a new nanoid from the database.
// Uses PostgreSQL nanoid() function. Safe for use within a transaction.
func GenerateNanoid(db *gorm.DB) (string, error) {
	var id string
	err := db.Raw("SELECT nanoid()").Scan(&id).Error
	return id, err
}
