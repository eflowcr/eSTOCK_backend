package database

type ReceivingTaskItem struct {
	SKU              string           `json:"sku"`
	ExpectedQuantity int              `json:"expected_qty"`
	Location         string           `json:"location"`
	LotNumbers       StringSliceOrCSV `json:"lot_numbers" gorm:"type:jsonb"`
	SerialNumbers    StringSliceOrCSV `json:"serial_numbers" gorm:"type:jsonb"`
}
