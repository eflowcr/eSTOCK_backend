package database

import "time"

type StockAlert struct {
	ID                    int        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SKU                   string     `gorm:"column:sku" json:"sku"`
	AlertType             string     `gorm:"column:alert_type" json:"alert_type"`
	CurrentStock          int        `gorm:"column:current_stock" json:"current_stock"`
	RecommendedStock      int        `gorm:"column:recommended_stock" json:"recommended_stock"`
	AlertLevel            string     `gorm:"column:alert_level" json:"alert_level"`
	PredictedStockOutDays *int       `gorm:"column:predicted_stock_out_days" json:"predicted_stock_out_days"`
	Message               string     `gorm:"column:message" json:"message"`
	IsResolved            bool       `gorm:"column:is_resolved" json:"is_resolved"`
	LotNumber             *string    `gorm:"column:lot_number" json:"lot_number"`
	ExpirationDate        *time.Time `gorm:"column:expiration_date" json:"expiration_date"`
	DaysToExpiration      *int       `gorm:"column:days_to_expiration" json:"days_to_expiration"`
	CreatedAt             time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	ResolvedAt            *time.Time `gorm:"column:resolved_at" json:"resolved_at"`
}

func (StockAlert) TableName() string {
	return "stock_alerts"
}
