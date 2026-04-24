package database

import (
	"time"

	"github.com/lib/pq"
)

// DeliveryNoteItem represents a snapshot line in a delivery note.
// LotNumbers is a TEXT[] snapshot of lot numbers picked at delivery time.
type DeliveryNoteItem struct {
	ID             string         `gorm:"column:id;primaryKey" json:"id"`
	DeliveryNoteID string         `gorm:"column:delivery_note_id" json:"delivery_note_id"`
	ArticleSKU     string         `gorm:"column:article_sku" json:"article_sku"`
	Qty            float64        `gorm:"column:qty" json:"qty"`
	LotNumbers     pq.StringArray `gorm:"column:lot_numbers;type:text[]" json:"lot_numbers,omitempty"`
	Notes          *string        `gorm:"column:notes" json:"notes,omitempty"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (DeliveryNoteItem) TableName() string {
	return "delivery_note_items"
}
