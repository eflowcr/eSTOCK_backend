package database

import "time"

type Location struct {
	ID           int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	LocationCode string    `gorm:"column:location_code;unique" json:"location_code"`
	Description  *string   `gorm:"column:description" json:"description"`
	Zone         *string   `gorm:"column:zone" json:"zone"`
	Type         string    `gorm:"column:type" json:"type"`
	IsActive     bool      `gorm:"column:is_active" json:"is_active"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Location) TableName() string {
	return "locations"
}
