package database

type PickingTaskItem struct {
	SKU              string           `json:"sku"`
	ExpectedQuantity int              `json:"expectedQty"`
	Location         string           `json:"location"`
	LotNumbers       StringSliceOrCSV `json:"lotNumbers"`
	SerialNumbers    StringSliceOrCSV `json:"serialNumbers"`
}
