package requests

type CreateSerialRequest struct {
	SerialNumber string `json:"serial_number" validate:"required,max=100"`
	SKU          string `json:"sku" validate:"required,max=100"`
}
