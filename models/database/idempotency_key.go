package database

import "time"

// IdempotencyKey is a cached response for a mobile mutation client UUID v4.
// See S7.2 W0 design: mobile generates a key per mutation, sends it on
// every retry via `Idempotency-Key` header; backend dedupes within a 7-day
// window so retries return the original response without re-applying.
//
// PK is composite (key, user_id) — keys are UUID v4 so cross-user collision
// is astronomical, but the user_id scope guarantees one user can never
// observe another user's cached response even in the worst case.
type IdempotencyKey struct {
	Key            string    `gorm:"column:key;primaryKey" json:"key"`
	UserID         string    `gorm:"column:user_id;primaryKey" json:"user_id"`
	Endpoint       string    `gorm:"column:endpoint" json:"endpoint"`
	Method         string    `gorm:"column:method" json:"method"`
	ResponseStatus int       `gorm:"column:response_status" json:"response_status"`
	ResponseBody   []byte    `gorm:"column:response_body;type:jsonb" json:"response_body"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	ExpiresAt      time.Time `gorm:"column:expires_at" json:"expires_at"`
}

func (IdempotencyKey) TableName() string {
	return "idempotency_keys"
}
