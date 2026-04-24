package ports

import "github.com/eflowcr/eSTOCK_backend/models/responses"

// DeliveryNotesRepository defines persistence operations for delivery notes (DN1–DN3).
type DeliveryNotesRepository interface {
	// List returns paginated delivery notes filtered by tenant and optional query params.
	List(tenantID string, customerID, soNumber *string, from, to *string, page, limit int) (*responses.DeliveryNoteListResponse, *responses.InternalResponse)

	// GetByID returns the full delivery note (header + items), scoped to tenantID.
	GetByID(id, tenantID string) (*responses.DeliveryNoteResponse, *responses.InternalResponse)

	// UpdatePDFURL sets pdf_url and pdf_generated_at after async PDF generation.
	UpdatePDFURL(id, pdfURL string) *responses.InternalResponse

	// GetDNNumber returns the dn_number for a given delivery note ID (used for PDF filename).
	GetDNNumber(id, tenantID string) (string, *responses.InternalResponse)
}
