package services

import (
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTenantA = "00000000-0000-0000-0000-000000000001"
	testTenantB = "00000000-0000-0000-0000-000000000002"
)

// ── mock ─────────────────────────────────────────────────────────────────────
//
// S3.5 W2-A: mock now records the tenantID it was called with so isolation
// tests can assert the controller passed the right tenant.
type mockLocationsRepo struct {
	locations    []database.Location
	byID         map[string]*database.Location
	createErr    *responses.InternalResponse
	deleteErr    *responses.InternalResponse
	gotTenantIDs []string // captures every tenantID passed to any method
}

func (m *mockLocationsRepo) recordTenant(t string) {
	m.gotTenantIDs = append(m.gotTenantIDs, t)
}

func (m *mockLocationsRepo) GetAllLocations(tenantID string) ([]database.Location, *responses.InternalResponse) {
	m.recordTenant(tenantID)
	out := make([]database.Location, 0, len(m.locations))
	for _, l := range m.locations {
		if l.TenantID == "" || l.TenantID == tenantID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (m *mockLocationsRepo) GetLocationByID(tenantID, id string) (*database.Location, *responses.InternalResponse) {
	m.recordTenant(tenantID)
	if m.byID != nil {
		if l, ok := m.byID[id]; ok {
			if l.TenantID == "" || l.TenantID == tenantID {
				return l, nil
			}
		}
	}
	return nil, &responses.InternalResponse{Message: "Ubicación no encontrada", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockLocationsRepo) CreateLocation(tenantID string, loc *requests.Location) *responses.InternalResponse {
	m.recordTenant(tenantID)
	return m.createErr
}

func (m *mockLocationsRepo) UpdateLocation(tenantID, id string, data map[string]interface{}) *responses.InternalResponse {
	m.recordTenant(tenantID)
	return nil
}

func (m *mockLocationsRepo) DeleteLocation(tenantID, id string) *responses.InternalResponse {
	m.recordTenant(tenantID)
	return m.deleteErr
}

func (m *mockLocationsRepo) ImportLocationsFromExcel(tenantID string, fileBytes []byte) ([]string, []string, *responses.InternalResponse) {
	m.recordTenant(tenantID)
	return nil, nil, nil
}

func (m *mockLocationsRepo) ImportLocationsFromJSON(tenantID string, rows []requests.LocationImportRow) ([]string, []string, *responses.InternalResponse) {
	m.recordTenant(tenantID)
	return nil, nil, nil
}

func (m *mockLocationsRepo) ValidateImportRows(tenantID string, rows []requests.LocationImportRow) ([]responses.LocationValidationResult, *responses.InternalResponse) {
	m.recordTenant(tenantID)
	return nil, nil
}

func (m *mockLocationsRepo) ExportLocationsToExcel(tenantID string) ([]byte, *responses.InternalResponse) {
	m.recordTenant(tenantID)
	return nil, nil
}

func (m *mockLocationsRepo) GenerateImportTemplate(language string) ([]byte, error) {
	return nil, nil
}

// ── GetAllLocations ───────────────────────────────────────────────────────────

func TestLocationsService_GetAll_Empty(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{locations: []database.Location{}})
	list, err := svc.GetAllLocations(testTenantA)
	assert.Nil(t, err)
	assert.Empty(t, list)
}

func TestLocationsService_GetAll_WithData(t *testing.T) {
	repo := &mockLocationsRepo{
		locations: []database.Location{
			{ID: "id1", TenantID: testTenantA, LocationCode: "LOC-A01", Type: "SHELF"},
			{ID: "id2", TenantID: testTenantA, LocationCode: "LOC-B01", Type: "PALLET"},
		},
	}
	svc := NewLocationsService(repo)
	list, err := svc.GetAllLocations(testTenantA)
	require.Nil(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "LOC-A01", list[0].LocationCode)
	assert.Equal(t, "LOC-B01", list[1].LocationCode)
}

// TestLocationsService_GetAll_TenantIsolation_returnsOnlyOwnTenant verifies
// that the service forwards the tenantID and the repository (mocked here as a
// real per-tenant filter) honours it. Real tenant guards live in the repo
// layer; this test asserts the plumbing is intact end-to-end.
func TestLocationsService_GetAll_TenantIsolation_returnsOnlyOwnTenant(t *testing.T) {
	repo := &mockLocationsRepo{
		locations: []database.Location{
			{ID: "a1", TenantID: testTenantA, LocationCode: "LOC-A01", Type: "SHELF"},
			{ID: "b1", TenantID: testTenantB, LocationCode: "LOC-B01", Type: "SHELF"},
		},
	}
	svc := NewLocationsService(repo)
	listA, err := svc.GetAllLocations(testTenantA)
	require.Nil(t, err)
	require.Len(t, listA, 1)
	assert.Equal(t, "a1", listA[0].ID)

	listB, err := svc.GetAllLocations(testTenantB)
	require.Nil(t, err)
	require.Len(t, listB, 1)
	assert.Equal(t, "b1", listB[0].ID)

	// Service must have forwarded both tenant IDs verbatim.
	assert.Contains(t, repo.gotTenantIDs, testTenantA)
	assert.Contains(t, repo.gotTenantIDs, testTenantB)
}

// ── GetLocationByID ───────────────────────────────────────────────────────────

func TestLocationsService_GetByID_Found(t *testing.T) {
	repo := &mockLocationsRepo{
		byID: map[string]*database.Location{
			"id1": {ID: "id1", TenantID: testTenantA, LocationCode: "LOC-A01", Type: "SHELF"},
		},
	}
	svc := NewLocationsService(repo)
	loc, err := svc.GetLocationByID(testTenantA, "id1")
	require.Nil(t, err)
	assert.Equal(t, "LOC-A01", loc.LocationCode)
}

func TestLocationsService_GetByID_NotFound(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{byID: map[string]*database.Location{}})
	loc, err := svc.GetLocationByID(testTenantA, "missing")
	assert.Nil(t, loc)
	require.NotNil(t, err)
	assert.Equal(t, responses.StatusNotFound, err.StatusCode)
}

// TestLocationsService_GetByID_TenantIsolation verifies a row owned by tenant A
// is not visible to tenant B even when the id is known.
func TestLocationsService_GetByID_TenantIsolation_blocksOtherTenant(t *testing.T) {
	repo := &mockLocationsRepo{
		byID: map[string]*database.Location{
			"id1": {ID: "id1", TenantID: testTenantA, LocationCode: "LOC-A01", Type: "SHELF"},
		},
	}
	svc := NewLocationsService(repo)

	// tenant A can read it
	loc, err := svc.GetLocationByID(testTenantA, "id1")
	require.Nil(t, err)
	require.NotNil(t, loc)

	// tenant B cannot — receives 404, no leak.
	loc2, err2 := svc.GetLocationByID(testTenantB, "id1")
	assert.Nil(t, loc2)
	require.NotNil(t, err2)
	assert.Equal(t, responses.StatusNotFound, err2.StatusCode)
}

// ── CreateLocation ────────────────────────────────────────────────────────────

func TestLocationsService_Create_Success(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{})
	err := svc.CreateLocation(testTenantA, &requests.Location{LocationCode: "LOC-NEW", Type: "BIN"})
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
	err := svc.CreateLocation(testTenantA, &requests.Location{LocationCode: "LOC-A01", Type: "SHELF"})
	require.NotNil(t, err)
	assert.True(t, err.Handled)
	assert.Contains(t, err.Message, "ya existe")
}

// ── DeleteLocation ────────────────────────────────────────────────────────────

func TestLocationsService_Delete_Success(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{})
	err := svc.DeleteLocation(testTenantA, "id1")
	assert.Nil(t, err)
}

func TestLocationsService_Delete_Error(t *testing.T) {
	repo := &mockLocationsRepo{
		deleteErr: &responses.InternalResponse{Message: "Error al eliminar", Handled: false},
	}
	svc := NewLocationsService(repo)
	err := svc.DeleteLocation(testTenantA, "id1")
	require.NotNil(t, err)
	assert.False(t, err.Handled)
}

// ── ImportLocationsFromJSON ───────────────────────────────────────────────────

func TestLocationsService_ImportJSON_Delegates(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{})
	imported, skipped, err := svc.ImportLocationsFromJSON(testTenantA, []requests.LocationImportRow{
		{LocationCode: "LOC-X01", Type: "SHELF"},
	})
	assert.Nil(t, err)
	assert.Nil(t, imported)
	assert.Nil(t, skipped)
}

func TestLocationsService_ImportJSON_Empty(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{})
	imported, skipped, err := svc.ImportLocationsFromJSON(testTenantA, []requests.LocationImportRow{})
	assert.Nil(t, err)
	assert.Empty(t, imported)
	assert.Empty(t, skipped)
}

// ── ValidateImportRows ────────────────────────────────────────────────────────

func TestLocationsService_ValidateImportRows_Delegates(t *testing.T) {
	svc := NewLocationsService(&mockLocationsRepo{})
	results, err := svc.ValidateImportRows(testTenantA, []requests.LocationImportRow{
		{LocationCode: "LOC-X01", Type: "PALLET"},
	})
	assert.Nil(t, err)
	assert.Nil(t, results)
}
