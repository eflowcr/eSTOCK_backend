package database

import (
	"encoding/json"
	"time"
)

type Session struct {
	SID    string          `gorm:"column:sid;primaryKey" json:"sid"`
	Sess   json.RawMessage `gorm:"column:sess;type:jsonb" json:"sess"`
	Expire time.Time       `gorm:"column:expire;index:idx_session_expire" json:"expire"`
}

func (Session) TableName() string {
	return "sessions"
}
