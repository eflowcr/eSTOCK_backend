package responses

import "github.com/eflowcr/eSTOCK_backend/models/database"

// InventoryCountDetail is returned by GET /api/mobile/counts/:id and includes
// the count header plus its locations and lines.
type InventoryCountDetail struct {
	Count     database.InventoryCount           `json:"count"`
	Locations []database.InventoryCountLocation `json:"locations"`
	Lines     []database.InventoryCountLine     `json:"lines"`
}

// MobileInventoryCountLine wraps database.InventoryCountLine and adds the
// resolved product `name` (Halo detail-card fidelity). Embedding keeps the
// existing JSON shape (id/sku/lot/qty/…) byte-for-byte compatible while adding
// the new `name` field.
type MobileInventoryCountLine struct {
	database.InventoryCountLine
	Name string `json:"name"`
}

// MobileInventoryCountDetail mirrors InventoryCountDetail but its lines carry
// the resolved product name. Returned by GET /api/mobile/counts/:id.
type MobileInventoryCountDetail struct {
	Count     database.InventoryCount           `json:"count"`
	Locations []database.InventoryCountLocation `json:"locations"`
	Lines     []MobileInventoryCountLine        `json:"lines"`
}
