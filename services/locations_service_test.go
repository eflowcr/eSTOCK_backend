package services

import (
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── mock ─────────────────────────────────────────────────────────────────────

type mockLocationsRepo struct {
	locations []database.Location
	byID      map[string]*database.Location
	createErr *responses.InternalResponse
	deleteErr *responses.InternalResponse
}

func (m *mockLocationsRepo) GetAllLocations() ([]database.Location, *responses.InternalResponse) {
	return m.locations, nil
}

func (m *mockLocationsRepo) GetLocationByID(id string) (*database.Location, *responses.InternalResponse) {
	if m.byID != nil {
		if l, ok := m.byID[id]; ok {
			return l, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "Ubicación no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockLocationsRepo) CreateLocation(loc *requests.Location) *responses.InternalResponse {
	return m.createErr
}

func (m *mockLocationsRepo) UpdateLocation(id string, data map[string]interface{}) *responses.InternalResponse {
	return nil
}

func (m *mockLocationsRepo) DeleteLocation(id string) *responses.InternalResponse {
	return m.deleteErr
}

func (m *mockLocationsRepo) ImportLocationsFromExcel(fileBytes []byte) ([]string, []string, *responses.InternalResponse) {
	return nil, nil, nil
}

func (m *mockLocationsRepo) ImportLocationsFromJSON(rows []requests.LocationImportRow) ([]string, []string, *responses.InternalResponse) {
	return nil, nil, nil
}

func (m *mockLocationsRepo) ValidateImportRows(rows []requests.LocationImportRow) ([]responses.LocationValidationResult, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockLocationsRepo) ExportLocationsToExcel() ([]byte, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockLocationsRepo) GenerateImportTemplate(language string) ([]byte, error) {
	return nil, nil
}

// ── GetAllLocations ───────────────────────────────────────────────────────────

func TestLocationsService_GetAll_Empty(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{locations: []database.Location{}})
	list, err := svc.GetAllLocations()
	assert.Nil(t, err)
	assert.Empty(t, list)
}

func TestLocationsService_GetAll_WithData(t *testing.T) {
	repo := &mockLocationsRepo{
		locations: []database.Location{
			{ID: "id1", LocationCode: "LOC-A01", Type: "SHELF"},
			{ID: "id2", LocationCode: "LOC-B01", Type: "PALLET"},
		},
	}
	svc := NewLocationsService(repo)
	list, err := svc.GetAllLocations()
	require.Nil(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "LOC-A01", list[0].LocationCode)
	assert.Equal(t, "LOC-B01", list[1].LocationCode)
}

// ── GetLocationByID ───────────────────────────────────────────────────────────

func TestLocationsService_GetByID_Found(t *testing.T) {
	repo := &mockLocationsRepo{
		byID: map[string]*database.Location{
			"id1": {ID: "id1", LocationCode: "LOC-A01", Type: "SHELF"},
		},
	}
	svc := NewLocationsService(repo)
	loc, err := svc.GetLocationByID("id1")
	require.Nil(t, err)
	assert.Equal(t, "LOC-A01", loc.LocationCode)
}

func TestLocationsService_GetByID_NotFound(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{byID: map[string]*database.Location{}})
	loc, err := svc.GetLocationByID("missing")
	assert.Nil(t, loc)
	require.NotNil(t, err)
	assert.Equal(t, responses.StatusNotFound, err.StatusCode)
}

// ── CreateLocation ────────────────────────────────────────────────────────────

func TestLocationsService_Create_Success(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{})
	err := svc.CreateLocation(&requests.Location{LocationCode: "LOC-NEW", Type: "BIN"})
	assert.Nil(t, err)
}

func TestLocationsService_Create_Conflict(t *testing.T) {
	repo := &mockLocationsRepo{
		createErr: &responses.InternalResponse{
			Message: "El código de ubicación ya existe",
			Handled: true,
		},
	}
	svc := NewLocationsService(repo)
	err := svc.CreateLocation(&requests.Location{LocationCode: "LOC-A01", Type: "SHELF"})
	require.NotNil(t, err)
	assert.True(t, err.Handled)
	assert.Contains(t, err.Message, "ya existe")
}

// ── DeleteLocation ────────────────────────────────────────────────────────────

func TestLocationsService_Delete_Success(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{})
	err := svc.DeleteLocation("id1")
	assert.Nil(t, err)
}

func TestLocationsService_Delete_Error(t *testing.T) {
	repo := &mockLocationsRepo{
		deleteErr: &responses.InternalResponse{Message: "Error al eliminar", Handled: false},
	}
	svc := NewLocationsService(repo)
	err := svc.DeleteLocation("id1")
	require.NotNil(t, err)
	assert.False(t, err.Handled)
}

// ── ImportLocationsFromJSON ───────────────────────────────────────────────────

func TestLocationsService_ImportJSON_Delegates(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{})
	imported, skipped, err := svc.ImportLocationsFromJSON([]requests.LocationImportRow{
		{LocationCode: "LOC-X01", Type: "SHELF"},
	})
	assert.Nil(t, err)
	assert.Nil(t, imported)
	assert.Nil(t, skipped)
}

func TestLocationsService_ImportJSON_Empty(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{})
	imported, skipped, err := svc.ImportLocationsFromJSON([]requests.LocationImportRow{})
	assert.Nil(t, err)
	assert.Empty(t, imported)
	assert.Empty(t, skipped)
}

// ── ValidateImportRows ────────────────────────────────────────────────────────

func TestLocationsService_ValidateImportRows_Delegates(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{})
	results, err := svc.ValidateImportRows([]requests.LocationImportRow{
		{LocationCode: "LOC-X01", Type: "PALLET"},
	})
	assert.Nil(t, err)
	assert.Nil(t, results)
}
