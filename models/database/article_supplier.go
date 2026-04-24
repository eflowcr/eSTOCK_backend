package database

import "time"

// ArticleSupplier represents the M-N relationship between articles and supplier clients.
// A single article can have multiple suppliers with preferred flag, lead time, and unit cost.
type ArticleSupplier struct {
	ID            string     `gorm:"column:id;primaryKey" json:"id"`
	TenantID      string     `gorm:"column:tenant_id" json:"tenant_id"`
	ArticleSKU    string     `gorm:"column:article_sku" json:"article_sku"`
	SupplierID    string     `gorm:"column:supplier_id" json:"supplier_id"`
	IsPreferred   bool       `gorm:"column:is_preferred" json:"is_preferred"`
	LeadTimeDays  *int       `gorm:"column:lead_time_days" json:"lead_time_days,omitempty"`
	UnitCost      *float64   `gorm:"column:unit_cost" json:"unit_cost,omitempty"`
	SupplierSKU   *string    `gorm:"column:supplier_sku" json:"supplier_sku,omitempty"`
	Notes         *string    `gorm:"column:notes" json:"notes,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (ArticleSupplier) TableName() string {
	return "article_suppliers"
}
