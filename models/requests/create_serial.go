package requests

type CreateSerialRequest struct {
	SerialNumber string `gorm:"column:serial_number" json:"serial_number" validate:"required"`
	SKU          string `gorm:"column:sku" json:"sku" validate:"required"`
}
