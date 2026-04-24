package responses

import "time"

// DeliveryNoteItemResponse is an embedded line item in a delivery note.
type DeliveryNoteItemResponse struct {
	ID             string     `json:"id"`
	DeliveryNoteID string     `json:"delivery_note_id"`
	ArticleSKU     string     `json:"article_sku"`
	Qty            float64    `json:"qty"`
	LotNumbers     []string   `json:"lot_numbers,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// DeliveryNoteResponse is the full response for a single delivery note (header + items).
type DeliveryNoteResponse struct {
	ID             string                     `json:"id"`
	DNNumber       string                     `json:"dn_number"`
	SalesOrderID   string                     `json:"sales_order_id"`
	PickingTaskID  *string                    `json:"picking_task_id,omitempty"`
	CustomerID     string                     `json:"customer_id"`
	CustomerName   *string                    `json:"customer_name,omitempty"`
	TotalItems     int                        `json:"total_items"`
	PdfURL         *string                    `json:"pdf_url,omitempty"`
	PdfGeneratedAt *time.Time                 `json:"pdf_generated_at,omitempty"`
	DeliveredAt    *time.Time                 `json:"delivered_at,omitempty"`
	SignedBy       *string                    `json:"signed_by,omitempty"`
	Items          []DeliveryNoteItemResponse `json:"items"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

// DeliveryNoteListItem is the lightweight row used in list responses.
type DeliveryNoteListItem struct {
	ID             string     `json:"id"`
	DNNumber       string     `json:"dn_number"`
	SalesOrderID   string     `json:"sales_order_id"`
	CustomerID     string     `json:"customer_id"`
	CustomerName   *string    `json:"customer_name,omitempty"`
	TotalItems     int        `json:"total_items"`
	PdfURL         *string    `json:"pdf_url,omitempty"`
	PdfGeneratedAt *time.Time `json:"pdf_generated_at,omitempty"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// DeliveryNoteListResponse wraps list + pagination.
type DeliveryNoteListResponse struct {
	Items      []DeliveryNoteListItem `json:"items"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
	TotalPages int                    `json:"total_pages"`
}
