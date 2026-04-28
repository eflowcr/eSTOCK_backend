package repositories

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

// InventoryCountsRepository persists count sheets, their target locations, and scan lines.
// Uses GORM (single-tenant codebase, no sqlc queries needed for this module).
type InventoryCountsRepository struct {
	DB *gorm.DB
}

func (r *InventoryCountsRepository) List(status, locationID string) ([]database.InventoryCount, *responses.InternalResponse) {
	var counts []database.InventoryCount
	tx := r.DB.Model(&database.InventoryCount{}).Order("created_at DESC")
	if status != "" {
		statuses := strings.Split(status, ",")
		tx = tx.Where("status IN ?", statuses)
	}
	if locationID != "" {
		tx = tx.Where("id IN (?)",
			r.DB.Model(&database.InventoryCountLocation{}).Select("count_id").Where("location_id = ?", locationID))
	}
	if err := tx.Find(&counts).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener conteos", Handled: false}
	}
	return counts, nil
}

func (r *InventoryCountsRepository) GetByID(id string) (*database.InventoryCount, *responses.InternalResponse) {
	var c database.InventoryCount
	if err := r.DB.Where("id = ?", id).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &responses.InternalResponse{Message: "Conteo no encontrado", Handled: true, StatusCode: responses.StatusNotFound}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener el conteo", Handled: false}
	}
	return &c, nil
}

func (r *InventoryCountsRepository) GetDetail(id string) (*responses.InventoryCountDetail, *responses.InternalResponse) {
	c, resp := r.GetByID(id)
	if resp != nil {
		return nil, resp
	}
	locs, resp := r.ListLocations(id)
	if resp != nil {
		return nil, resp
	}
	lines, resp := r.ListLines(id)
	if resp != nil {
		return nil, resp
	}
	return &responses.InventoryCountDetail{Count: *c, Locations: locs, Lines: lines}, nil
}

func (r *InventoryCountsRepository) Create(userID string, req *requests.CreateInventoryCount) (*database.InventoryCount, *responses.InternalResponse) {
	var created *database.InventoryCount

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		// Uniqueness on code is enforced by DB, but check up front for nicer error.
		var n int64
		if err := tx.Model(&database.InventoryCount{}).Where("code = ?", req.Code).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return fmt.Errorf("CODE_TAKEN")
		}

		id, err := tools.GenerateNanoid(tx)
		if err != nil {
			return fmt.Errorf("generate id: %w", err)
		}

		var desc *string
		if strings.TrimSpace(req.Description) != "" {
			d := req.Description
			desc = &d
		}

		now := time.Now()
		c := &database.InventoryCount{
			ID:           id,
			Code:         req.Code,
			Name:         req.Name,
			Description:  desc,
			Status:       "draft",
			ScheduledFor: req.ScheduledFor,
			CreatedBy:    userID,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(c).Error; err != nil {
			return fmt.Errorf("create count: %w", err)
		}

		// Insert locations.
		seen := map[string]bool{}
		for _, locID := range req.LocationIDs {
			if locID == "" || seen[locID] {
				continue
			}
			seen[locID] = true
			locRowID, err := tools.GenerateNanoid(tx)
			if err != nil {
				return fmt.Errorf("generate location row id: %w", err)
			}
			row := &database.InventoryCountLocation{
				ID:         locRowID,
				CountID:    id,
				LocationID: locID,
				Status:     "pending",
			}
			if err := tx.Create(row).Error; err != nil {
				return fmt.Errorf("create count location: %w", err)
			}
		}

		created = c
		return nil
	})

	if err != nil {
		if err.Error() == "CODE_TAKEN" {
			return nil, &responses.InternalResponse{Message: "Ya existe un conteo con ese código", Handled: true, StatusCode: responses.StatusConflict}
		}
		return nil, &responses.InternalResponse{Error: err, Message: err.Error(), Handled: true, StatusCode: responses.StatusBadRequest}
	}
	return created, nil
}

func (r *InventoryCountsRepository) UpdateStatus(id, status string) *responses.InternalResponse {
	res := r.DB.Model(&database.InventoryCount{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"status": status, "updated_at": time.Now()})
	if res.Error != nil {
		return &responses.InternalResponse{Error: res.Error, Message: "Error al actualizar el estado del conteo", Handled: false}
	}
	if res.RowsAffected == 0 {
		return &responses.InternalResponse{Message: "Conteo no encontrado", Handled: true, StatusCode: responses.StatusNotFound}
	}
	return nil
}

func (r *InventoryCountsRepository) MarkStarted(id string) *responses.InternalResponse {
	now := time.Now()
	res := r.DB.Model(&database.InventoryCount{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"status": "in_progress", "started_at": &now, "updated_at": now})
	if res.Error != nil {
		return &responses.InternalResponse{Error: res.Error, Message: "Error al iniciar el conteo", Handled: false}
	}
	if res.RowsAffected == 0 {
		return &responses.InternalResponse{Message: "Conteo no encontrado", Handled: true, StatusCode: responses.StatusNotFound}
	}
	return nil
}

func (r *InventoryCountsRepository) MarkCancelled(id string) *responses.InternalResponse {
	return r.UpdateStatus(id, "cancelled")
}

func (r *InventoryCountsRepository) MarkSubmitted(id, submittedBy, adjustmentID string) *responses.InternalResponse {
	now := time.Now()
	updates := map[string]interface{}{
		"status":       "submitted",
		"submitted_at": &now,
		"submitted_by": submittedBy,
		"updated_at":   now,
	}
	if adjustmentID != "" {
		updates["adjustment_id"] = adjustmentID
	}
	res := r.DB.Model(&database.InventoryCount{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return &responses.InternalResponse{Error: res.Error, Message: "Error al enviar el conteo", Handled: false}
	}
	if res.RowsAffected == 0 {
		return &responses.InternalResponse{Message: "Conteo no encontrado", Handled: true, StatusCode: responses.StatusNotFound}
	}
	return nil
}

func (r *InventoryCountsRepository) ListLines(countID string) ([]database.InventoryCountLine, *responses.InternalResponse) {
	var lines []database.InventoryCountLine
	if err := r.DB.Where("count_id = ?", countID).Order("scanned_at ASC").Find(&lines).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener las líneas del conteo", Handled: false}
	}
	return lines, nil
}

func (r *InventoryCountsRepository) AddLine(line *database.InventoryCountLine) *responses.InternalResponse {
	if line.ID == "" {
		id, err := tools.GenerateNanoid(r.DB)
		if err != nil {
			return &responses.InternalResponse{Error: err, Message: "Error al generar ID", Handled: false}
		}
		line.ID = id
	}
	if err := r.DB.Create(line).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al guardar la línea de conteo", Handled: false}
	}
	return nil
}

func (r *InventoryCountsRepository) ListLocations(countID string) ([]database.InventoryCountLocation, *responses.InternalResponse) {
	var locs []database.InventoryCountLocation
	if err := r.DB.Where("count_id = ?", countID).Find(&locs).Error; err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener ubicaciones del conteo", Handled: false}
	}
	return locs, nil
}

// ResolveSKUByBarcode returns a SKU when the article is keyed by barcode.
// The current articles schema has no dedicated barcode column — we look up by SKU
// equality as a fallback (mobile sends sku in barcode field for now). When the
// multi-barcode column lands, swap in a JOIN against item_barcode.
func (r *InventoryCountsRepository) ResolveSKUByBarcode(barcode string) (string, *responses.InternalResponse) {
	if strings.TrimSpace(barcode) == "" {
		return "", &responses.InternalResponse{Message: "Código de barras vacío", Handled: true, StatusCode: responses.StatusBadRequest}
	}
	var sku string
	err := r.DB.Table("articles").Select("sku").Where("sku = ?", barcode).Limit(1).Scan(&sku).Error
	if err != nil {
		return "", &responses.InternalResponse{Error: err, Message: "Error al resolver código de barras", Handled: false}
	}
	if sku == "" {
		return "", &responses.InternalResponse{Message: "Código de barras no encontrado", Handled: true, StatusCode: responses.StatusNotFound}
	}
	return sku, nil
}

// GetExpectedQty returns the on-hand quantity for sku+locationCode (and optionally lot, ignored at base inventory level).
func (r *InventoryCountsRepository) GetExpectedQty(sku, locationCode, lot string) (float64, *responses.InternalResponse) {
	var inv database.Inventory
	err := r.DB.Where("sku = ? AND location = ?", sku, locationCode).First(&inv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, &responses.InternalResponse{Error: err, Message: "Error al obtener inventario esperado", Handled: false}
	}
	return inv.Quantity, nil
}

func (r *InventoryCountsRepository) GetLocationCodeByID(locationID string) (string, *responses.InternalResponse) {
	var code string
	err := r.DB.Table("locations").Select("location_code").Where("id = ?", locationID).Limit(1).Scan(&code).Error
	if err != nil {
		return "", &responses.InternalResponse{Error: err, Message: "Error al obtener ubicación", Handled: false}
	}
	if code == "" {
		return "", &responses.InternalResponse{Message: "Ubicación no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
	}
	return code, nil
}
