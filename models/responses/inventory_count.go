package responses

import "github.com/eflowcr/eSTOCK_backend/models/database"

// InventoryCountDetail is returned by GET /api/mobile/counts/:id and includes
// the count header plus its locations and lines.
type InventoryCountDetail struct {
	Count     database.InventoryCount           `json:"count"`
	Locations []database.InventoryCountLocation `json:"locations"`
	Lines     []database.InventoryCountLine     `json:"lines"`
}
