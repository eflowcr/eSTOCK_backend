package database

import "time"

type UserBadge struct {
	ID        int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    string    `gorm:"column:user_id;index:idx_user_badges_user_id" json:"user_id"`
	BadgeID   int       `gorm:"column:badge_id;index:idx_user_badges_badge_id" json:"badge_id"`
	AwardedAt time.Time `gorm:"column:awarded_at;autoCreateTime" json:"awarded_at"`
}

func (UserBadge) TableName() string {
	return "user_badges"
}
