package database

import "time"

// InventoryCount represents a cycle / physical count sheet.
// Status enum: draft, scheduled, in_progress, in_review, submitted, cancelled.
type InventoryCount struct {
	ID           string     `gorm:"column:id;primaryKey" json:"id"`
	Code         string     `gorm:"column:code;unique" json:"code"`
	Name         string     `gorm:"column:name" json:"name"`
	Description  *string    `gorm:"column:description" json:"description,omitempty"`
	Status       string     `gorm:"column:status" json:"status"`
	ScheduledFor *time.Time `gorm:"column:scheduled_for" json:"scheduled_for,omitempty"`
	StartedAt    *time.Time `gorm:"column:started_at" json:"started_at,omitempty"`
	CompletedAt  *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	SubmittedAt  *time.Time `gorm:"column:submitted_at" json:"submitted_at,omitempty"`
	SubmittedBy  *string    `gorm:"column:submitted_by" json:"submitted_by,omitempty"`
	AdjustmentID *string    `gorm:"column:adjustment_id" json:"adjustment_id,omitempty"`
	CreatedBy    string     `gorm:"column:created_by" json:"created_by"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (InventoryCount) TableName() string {
	return "inventory_counts"
}

// InventoryCountLocation links a count sheet to a specific location (each row is a "zone" to count).
type InventoryCountLocation struct {
	ID         string  `gorm:"column:id;primaryKey" json:"id"`
	CountID    string  `gorm:"column:count_id" json:"count_id"`
	LocationID string  `gorm:"column:location_id" json:"location_id"`
	Status     string  `gorm:"column:status" json:"status"`
	AssignedTo *string `gorm:"column:assigned_to" json:"assigned_to,omitempty"`
}

func (InventoryCountLocation) TableName() string {
	return "inventory_count_locations"
}

// InventoryCountLine is a single scan event against a count sheet.
// variance_qty = scanned_qty - expected_qty (computed by service at scan time).
type InventoryCountLine struct {
	ID          string    `gorm:"column:id;primaryKey" json:"id"`
	CountID     string    `gorm:"column:count_id" json:"count_id"`
	LocationID  string    `gorm:"column:location_id" json:"location_id"`
	SKU         string    `gorm:"column:sku" json:"sku"`
	Lot         *string   `gorm:"column:lot" json:"lot,omitempty"`
	Serial      *string   `gorm:"column:serial" json:"serial,omitempty"`
	ExpectedQty float64   `gorm:"column:expected_qty" json:"expected_qty"`
	ScannedQty  float64   `gorm:"column:scanned_qty" json:"scanned_qty"`
	VarianceQty float64   `gorm:"column:variance_qty" json:"variance_qty"`
	Note        *string   `gorm:"column:note" json:"note,omitempty"`
	ScannedBy   string    `gorm:"column:scanned_by" json:"scanned_by"`
	ScannedAt   time.Time `gorm:"column:scanned_at;autoCreateTime" json:"scanned_at"`
}

func (InventoryCountLine) TableName() string {
	return "inventory_count_lines"
}
