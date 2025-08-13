package database

type ReceivingTaskItem struct {
	SKU              string           `json:"sku"`
	ExpectedQuantity int              `json:"expectedQty"`
	Location         string           `json:"location"`
	LotNumbers       StringSliceOrCSV `json:"lotNumbers" gorm:"type:jsonb"`
	SerialNumbers    StringSliceOrCSV `json:"serialNumbers" gorm:"type:jsonb"`
}
