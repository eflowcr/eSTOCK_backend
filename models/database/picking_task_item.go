package database

type PickingTaskItem struct {
	SKU              string           `json:"sku"`
	ExpectedQuantity int              `json:"required_qty"`
	Location         string           `json:"location"`
	LotNumbers       StringSliceOrCSV `json:"lotNumbers"`
	SerialNumbers    StringSliceOrCSV `json:"serialNumbers"`
}
