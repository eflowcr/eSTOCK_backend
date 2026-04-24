package repositories

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

// DeliveryNotesRepository implements ports.DeliveryNotesRepository using GORM.
type DeliveryNotesRepository struct {
	DB *gorm.DB
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

// nextDNNumber generates "DN-YYYY-NNNN" unique per tenant per year inside tx.
// Uses pg_advisory_xact_lock to serialize number generation per (tenant, year),
// which correctly handles the empty-table case where SELECT MAX ... FOR UPDATE
// would lock no rows and allow duplicate numbers under concurrent inserts.
func nextDNNumber(tx *gorm.DB, tenantID string) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("DN-%d-", year)

	// Acquire a per-(tenant, year) advisory lock for the duration of this transaction.
	// hashtext() produces a stable int4 from the string key; combining with year
	// ensures locks don't cross years. The lock is automatically released on tx commit/rollback.
	lockKey := fmt.Sprintf("dn-number-%s-%d", tenantID, year)
	if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey).Error; err != nil {
		return "", fmt.Errorf("acquire DN number lock: %w", err)
	}

	var maxNum int
	if err := tx.Raw(`
		SELECT COALESCE(MAX(
			CAST(SUBSTRING(dn_number FROM LENGTH($1)+1) AS INTEGER)
		), 0)
		FROM delivery_notes
		WHERE tenant_id = $2
		  AND dn_number LIKE $3
	`, prefix, tenantID, prefix+"%").Scan(&maxNum).Error; err != nil {
		return "", fmt.Errorf("generate DN number: %w", err)
	}

	return fmt.Sprintf("%s%04d", prefix, maxNum+1), nil
}

// toDNItemResponse converts a database.DeliveryNoteItem to the API response shape.
func toDNItemResponse(item *database.DeliveryNoteItem) responses.DeliveryNoteItemResponse {
	lots := make([]string, len(item.LotNumbers))
	copy(lots, item.LotNumbers)
	return responses.DeliveryNoteItemResponse{
		ID:             item.ID,
		DeliveryNoteID: item.DeliveryNoteID,
		ArticleSKU:     item.ArticleSKU,
		Qty:            item.Qty,
		LotNumbers:     lots,
		Notes:          item.Notes,
		CreatedAt:      item.CreatedAt,
	}
}

// toDNResponse builds the full response from header + items.
func toDNResponse(dn *database.DeliveryNote, items []database.DeliveryNoteItem, customerName *string) *responses.DeliveryNoteResponse {
	itemsResp := make([]responses.DeliveryNoteItemResponse, 0, len(items))
	for i := range items {
		itemsResp = append(itemsResp, toDNItemResponse(&items[i]))
	}
	return &responses.DeliveryNoteResponse{
		ID:             dn.ID,
		DNNumber:       dn.DNNumber,
		SalesOrderID:   dn.SalesOrderID,
		PickingTaskID:  dn.PickingTaskID,
		CustomerID:     dn.CustomerID,
		CustomerName:   customerName,
		TotalItems:     dn.TotalItems,
		PdfURL:         dn.PdfURL,
		PdfGeneratedAt: dn.PdfGeneratedAt,
		DeliveredAt:    dn.DeliveredAt,
		SignedBy:       dn.SignedBy,
		Items:          itemsResp,
		CreatedAt:      dn.CreatedAt,
		UpdatedAt:      dn.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DN-Create (called from picking_task_repository — not a port method)
// ─────────────────────────────────────────────────────────────────────────────

// DNCreationParams groups everything needed to create a delivery note inside a tx.
type DNCreationParams struct {
	TenantID      string
	SalesOrderID  string
	PickingTaskID string
	CustomerID    string
	Items         []DNItemCreationParam
}

// DNItemCreationParam is one line item for DN creation.
type DNItemCreationParam struct {
	ArticleSKU string
	Qty        float64
	LotNumbers []string
}

// CreateDeliveryNote inserts a delivery_note header + items in a transaction.
// Called from within the picking completion flow (already in a tx).
// Returns the created DeliveryNote ID.
func CreateDeliveryNote(tx *gorm.DB, params DNCreationParams) (string, error) {
	dnNumber, err := nextDNNumber(tx, params.TenantID)
	if err != nil {
		return "", fmt.Errorf("nextDNNumber: %w", err)
	}

	var dnID string
	if err := tx.Raw("SELECT nanoid(16)").Scan(&dnID).Error; err != nil {
		return "", fmt.Errorf("generate dn id: %w", err)
	}

	ptID := params.PickingTaskID
	dn := &database.DeliveryNote{
		ID:            dnID,
		TenantID:      params.TenantID,
		DNNumber:      dnNumber,
		SalesOrderID:  params.SalesOrderID,
		PickingTaskID: &ptID,
		CustomerID:    params.CustomerID,
		TotalItems:    len(params.Items),
	}
	if err := tx.Create(dn).Error; err != nil {
		return "", fmt.Errorf("create delivery_note: %w", err)
	}

	for _, it := range params.Items {
		var itemID string
		if err := tx.Raw("SELECT nanoid()").Scan(&itemID).Error; err != nil {
			return "", fmt.Errorf("generate dn item id: %w", err)
		}
		lots := make([]string, len(it.LotNumbers))
		copy(lots, it.LotNumbers)
		dni := &database.DeliveryNoteItem{
			ID:             itemID,
			DeliveryNoteID: dnID,
			ArticleSKU:     it.ArticleSKU,
			Qty:            it.Qty,
			LotNumbers:     lots,
		}
		if err := tx.Create(dni).Error; err != nil {
			return "", fmt.Errorf("create delivery_note_item %s: %w", it.ArticleSKU, err)
		}
	}

	return dnID, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ports.DeliveryNotesRepository implementation
// ─────────────────────────────────────────────────────────────────────────────

// List returns paginated delivery notes for a tenant.
func (r *DeliveryNotesRepository) List(tenantID string, customerID, soNumber *string, from, to *string, page, limit int) (*responses.DeliveryNoteListResponse, *responses.InternalResponse) {
	type rawRow struct {
		database.DeliveryNote
		CustomerName string `gorm:"column:customer_name"`
	}

	q := r.DB.Table("delivery_notes dn").
		Select("dn.*, c.name AS customer_name").
		Joins("LEFT JOIN clients c ON c.id = dn.customer_id").
		Where("dn.tenant_id = ?", tenantID)

	if customerID != nil && *customerID != "" {
		q = q.Where("dn.customer_id = ?", *customerID)
	}
	if soNumber != nil && *soNumber != "" {
		like := "%" + strings.ToLower(*soNumber) + "%"
		q = q.Where("EXISTS (SELECT 1 FROM sales_orders so WHERE so.id = dn.sales_order_id AND LOWER(so.so_number) LIKE ?)", like)
	}
	if from != nil && *from != "" {
		q = q.Where("dn.created_at >= ?", *from)
	}
	if to != nil && *to != "" {
		q = q.Where("dn.created_at <= ?", *to)
	}

	var total int64
	countQ := r.DB.Table("delivery_notes dn").
		Joins("LEFT JOIN clients c ON c.id = dn.customer_id").
		Where("dn.tenant_id = ?", tenantID)
	if customerID != nil && *customerID != "" {
		countQ = countQ.Where("dn.customer_id = ?", *customerID)
	}
	if soNumber != nil && *soNumber != "" {
		like := "%" + strings.ToLower(*soNumber) + "%"
		countQ = countQ.Where("EXISTS (SELECT 1 FROM sales_orders so WHERE so.id = dn.sales_order_id AND LOWER(so.so_number) LIKE ?)", like)
	}
	if from != nil && *from != "" {
		countQ = countQ.Where("dn.created_at >= ?", *from)
	}
	if to != nil && *to != "" {
		countQ = countQ.Where("dn.created_at <= ?", *to)
	}
	if err := countQ.Count(&total).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al contar notas de entrega"}
	}

	offset := (page - 1) * limit
	var rows []rawRow
	if err := q.Order("dn.created_at DESC").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al listar notas de entrega"}
	}

	items := make([]responses.DeliveryNoteListItem, 0, len(rows))
	for _, row := range rows {
		name := row.CustomerName
		var namePtr *string
		if name != "" {
			namePtr = &name
		}
		items = append(items, responses.DeliveryNoteListItem{
			ID:             row.ID,
			DNNumber:       row.DNNumber,
			SalesOrderID:   row.SalesOrderID,
			CustomerID:     row.CustomerID,
			CustomerName:   namePtr,
			TotalItems:     row.TotalItems,
			PdfURL:         row.PdfURL,
			PdfGeneratedAt: row.PdfGeneratedAt,
			DeliveredAt:    row.DeliveredAt,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		})
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &responses.DeliveryNoteListResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// GetByID returns the full delivery note with embedded items.
func (r *DeliveryNotesRepository) GetByID(id, tenantID string) (*responses.DeliveryNoteResponse, *responses.InternalResponse) {
	var dn database.DeliveryNote
	if err := r.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&dn).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &responses.InternalResponse{
				Message:    "Nota de entrega no encontrada",
				Handled:    true,
				StatusCode: responses.StatusNotFound,
			}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener nota de entrega"}
	}

	var items []database.DeliveryNoteItem
	if err := r.DB.Where("delivery_note_id = ?", id).Find(&items).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al cargar items de nota de entrega"}
	}

	// Load customer name.
	var customerName *string
	var client struct {
		Name string `gorm:"column:name"`
	}
	if err := r.DB.Table("clients").Select("name").Where("id = ?", dn.CustomerID).Scan(&client).Error; err == nil && client.Name != "" {
		customerName = &client.Name
	}

	return toDNResponse(&dn, items, customerName), nil
}

// UpdatePDFURL sets pdf_url and pdf_generated_at on a delivery note.
func (r *DeliveryNotesRepository) UpdatePDFURL(id, pdfURL string) *responses.InternalResponse {
	now := time.Now()
	if err := r.DB.Table("delivery_notes").
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"pdf_url":          pdfURL,
			"pdf_generated_at": now,
			"updated_at":       now,
		}).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al actualizar PDF URL"}
	}
	return nil
}

// GetDNNumber returns the dn_number for a delivery note (used for PDF filename).
func (r *DeliveryNotesRepository) GetDNNumber(id, tenantID string) (string, *responses.InternalResponse) {
	var result struct {
		DNNumber string `gorm:"column:dn_number"`
	}
	if err := r.DB.Table("delivery_notes").
		Select("dn_number").
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Scan(&result).Error; err != nil {
		return "", &responses.InternalResponse{Error: err, Message: "Error al obtener número de nota de entrega"}
	}
	if result.DNNumber == "" {
		return "", &responses.InternalResponse{
			Message:    "Nota de entrega no encontrada",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	}
	return result.DNNumber, nil
}
