package services

import (
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── mock ─────────────────────────────────────────────────────────────────────

type mockInventoryRepo struct {
	all        []*dto.EnhancedInventory
	bySkuLoc   *dto.EnhancedInventory
	createErr  *responses.InternalResponse
}

func (m *mockInventoryRepo) GetAllInventory() ([]*dto.EnhancedInventory, *responses.InternalResponse) {
	return m.all, nil
}
func (m *mockInventoryRepo) GetPickSuggestionsBySKU(_ string, _ float64) (*dto.PickSuggestionResponse, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockInventoryRepo) GetInventoryBySkuAndLocation(_, _ string) (*dto.EnhancedInventory, *responses.InternalResponse) {
	return m.bySkuLoc, nil
}
func (m *mockInventoryRepo) CreateInventory(_ string, _ *requests.CreateInventory) *responses.InternalResponse {
	return m.createErr
}
func (m *mockInventoryRepo) UpdateInventory(_ *requests.UpdateInventory) *responses.InternalResponse {
	return nil
}
func (m *mockInventoryRepo) DeleteInventory(_, _ string) *responses.InternalResponse { return nil }
func (m *mockInventoryRepo) Trend(_ string) (*dto.ConsumptionTrend, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockInventoryRepo) ImportInventoryFromExcel(_ string, _ []byte) ([]string, []string, *responses.InternalResponse) {
	return nil, nil, nil
}
func (m *mockInventoryRepo) ImportInventoryFromJSON(_ string, _ []requests.InventoryImportRow) ([]string, []string, *responses.InternalResponse) {
	return nil, nil, nil
}
func (m *mockInventoryRepo) ValidateImportRows(_ []requests.InventoryImportRow) ([]responses.InventoryValidationResult, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockInventoryRepo) ExportInventoryToExcel() ([]byte, *responses.InternalResponse) { return nil, nil }
func (m *mockInventoryRepo) GetInventoryLots(_ string) ([]responses.InventoryLot, *responses.InternalResponse) { return nil, nil }
func (m *mockInventoryRepo) GetInventorySerials(_ string) ([]responses.InventorySerialWithSerial, *responses.InternalResponse) { return nil, nil }
func (m *mockInventoryRepo) CreateInventoryLot(_ string, _ *requests.CreateInventoryLotRequest) *responses.InternalResponse { return nil }
func (m *mockInventoryRepo) DeleteInventoryLot(_ string) *responses.InternalResponse { return nil }
func (m *mockInventoryRepo) CreateInventorySerial(_ string, _ *requests.CreateInventorySerial) *responses.InternalResponse { return nil }
func (m *mockInventoryRepo) DeleteInventorySerial(_ string) *responses.InternalResponse { return nil }
func (m *mockInventoryRepo) GenerateImportTemplate(_ string) ([]byte, error) { return nil, nil }

// ── GetAllInventory ───────────────────────────────────────────────────────────

func TestInventoryService_GetAll_Empty(t *testing.T) {
	svc := NewInventoryService(&mockInventoryRepo{}, nil)
	list, err := svc.GetAllInventory()
	assert.Nil(t, err)
	assert.Empty(t, list)
}

func TestInventoryService_GetAll_WithData(t *testing.T) {
	repo := &mockInventoryRepo{
		all: []*dto.EnhancedInventory{
			{SKU: "SKU-001", Location: "LOC-A01", Quantity: 10},
			{SKU: "SKU-002", Location: "LOC-B01", Quantity: 5},
		},
	}
	svc := NewInventoryService(repo, nil)
	list, err := svc.GetAllInventory()
	require.Nil(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "SKU-001", list[0].SKU)
}

// ── GetInventoryBySkuAndLocation ──────────────────────────────────────────────

func TestInventoryService_GetBySkuAndLocation_Found(t *testing.T) {
	repo := &mockInventoryRepo{
		bySkuLoc: &dto.EnhancedInventory{SKU: "SKU-001", Location: "LOC-A01", Quantity: 10},
	}
	svc := NewInventoryService(repo, nil)
	inv, err := svc.GetInventoryBySkuAndLocation("SKU-001", "LOC-A01")
	require.Nil(t, err)
	assert.Equal(t, "SKU-001", inv.SKU)
}

func TestInventoryService_GetBySkuAndLocation_NotFound(t *testing.T) {
	svc := NewInventoryService(&mockInventoryRepo{}, nil)
	inv, err := svc.GetInventoryBySkuAndLocation("MISSING", "LOC-X")
	assert.Nil(t, err)    // mock returns nil,nil
	assert.Nil(t, inv)
}

// ── ImportInventoryFromJSON ───────────────────────────────────────────────────

func TestInventoryService_ImportJSON_Delegates(t *testing.T) {
	svc := NewInventoryService(&mockInventoryRepo{}, nil)
	imported, skipped, err := svc.ImportInventoryFromJSON("user1", []requests.InventoryImportRow{
		{SKU: "SKU-001", Location: "LOC-A01", Quantity: "10"},
	})
	assert.Nil(t, err)
	assert.Nil(t, imported)
	assert.Nil(t, skipped)
}

func TestInventoryService_ImportJSON_Empty(t *testing.T) {
	svc := NewInventoryService(&mockInventoryRepo{}, nil)
	imported, skipped, err := svc.ImportInventoryFromJSON("user1", []requests.InventoryImportRow{})
	assert.Nil(t, err)
	assert.Empty(t, imported)
	assert.Empty(t, skipped)
}

// ── ValidateImportRows ────────────────────────────────────────────────────────

func TestInventoryService_ValidateImportRows_Delegates(t *testing.T) {
	svc := NewInventoryService(&mockInventoryRepo{}, nil)
	results, err := svc.ValidateImportRows([]requests.InventoryImportRow{
		{SKU: "SKU-X01", Location: "LOC-A01", Quantity: "5"},
	})
	assert.Nil(t, err)
	assert.Nil(t, results)
}
