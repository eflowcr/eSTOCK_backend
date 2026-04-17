package database

import (
	"encoding/json"
	"time"
)

type ReceivingTask struct {
	ID              string          `gorm:"column:id;primaryKey" json:"id"`
	TaskID          string          `gorm:"column:task_id;unique" json:"task_id"`
	InboundNumber   string          `gorm:"column:inbound_number" json:"inbound_number"`
	CreatedBy       string          `gorm:"column:created_by" json:"created_by"`
	AssignedTo      *string         `gorm:"column:assigned_to" json:"assigned_to"`
	Status          string          `gorm:"column:status" json:"status"`
	Priority        string          `gorm:"column:priority" json:"priority"`
	Notes           *string         `gorm:"column:notes" json:"notes"`
	Items           json.RawMessage `gorm:"column:items;type:jsonb" json:"items"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	CompletedAt     *time.Time      `gorm:"column:completed_at" json:"completed_at"`
	// S2 M2 vendor/supplier fields
	SupplierID      *string `gorm:"column:supplier_id" json:"supplier_id,omitempty"`
	VendorRef       *string `gorm:"column:vendor_ref" json:"vendor_ref,omitempty"`
	TrackingNumber  *string `gorm:"column:tracking_number" json:"tracking_number,omitempty"`
	ReceptionMethod *string `gorm:"column:reception_method" json:"reception_method,omitempty"`
	Incoterms       *string `gorm:"column:incoterms" json:"incoterms,omitempty"`
}

func (ReceivingTask) TableName() string {
	return "receiving_tasks"
}
