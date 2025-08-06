package database

import (
	"encoding/json"
	"time"
)

type Badge struct {
	ID          int             `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string          `gorm:"column:name" json:"name"`
	Description string          `gorm:"column:description" json:"description"`
	Emoji       string          `gorm:"column:emoji" json:"emoji"`
	RuleType    string          `gorm:"column:rule_type" json:"rule_type"`
	Criteria    json.RawMessage `gorm:"column:criteria;type:jsonb" json:"criteria"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Badge) TableName() string {
	return "badges"
}
