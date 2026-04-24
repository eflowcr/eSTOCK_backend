package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// stub repo
// ─────────────────────────────────────────────────────────────────────────────

type stubDNRepo struct {
	listResult  *responses.DeliveryNoteListResponse
	listErr     *responses.InternalResponse
	getResult   *responses.DeliveryNoteResponse
	getErr      *responses.InternalResponse
	dnNumber    string
	dnNumberErr *responses.InternalResponse
}

func (s *stubDNRepo) List(_ string, _, _ *string, _, _ *string, _, _ int) (*responses.DeliveryNoteListResponse, *responses.InternalResponse) {
	return s.listResult, s.listErr
}
func (s *stubDNRepo) GetByID(_, _ string) (*responses.DeliveryNoteResponse, *responses.InternalResponse) {
	return s.getResult, s.getErr
}
func (s *stubDNRepo) UpdatePDFURL(_, _ string) *responses.InternalResponse { return nil }
func (s *stubDNRepo) GetDNNumber(_, _ string) (string, *responses.InternalResponse) {
	return s.dnNumber, s.dnNumberErr
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func dnController(repo *stubDNRepo) *DeliveryNotesController {
	svc := services.NewDeliveryNotesService(repo, nil)
	return NewDeliveryNotesController(svc, "tenant-test")
}

func dnGin(ctrl *DeliveryNotesController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/delivery-notes", ctrl.List)
	r.GET("/delivery-notes/:id", ctrl.GetByID)
	r.GET("/delivery-notes/:id/pdf", ctrl.DownloadPDF)
	return r
}

// ─────────────────────────────────────────────────────────────────────────────
// tests
// ─────────────────────────────────────────────────────────────────────────────

func TestDNController_List_OK(t *testing.T) {
	repo := &stubDNRepo{listResult: &responses.DeliveryNoteListResponse{
		Items: []responses.DeliveryNoteListItem{{ID: "dn-1", DNNumber: "DN-2026-0001"}},
		Total: 1, Page: 1, Limit: 50,
	}}
	r := dnGin(dnController(repo))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/delivery-notes", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body["data"])
}

func TestDNController_GetByID_OK(t *testing.T) {
	repo := &stubDNRepo{getResult: &responses.DeliveryNoteResponse{
		ID:       "dn-1",
		DNNumber: "DN-2026-0001",
		Items:    []responses.DeliveryNoteItemResponse{},
	}}
	r := dnGin(dnController(repo))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/delivery-notes/dn-1", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestDNController_GetByID_NotFound(t *testing.T) {
	repo := &stubDNRepo{getErr: &responses.InternalResponse{
		Message:    "Nota de entrega no encontrada",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}}
	r := dnGin(dnController(repo))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/delivery-notes/nonexistent", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDNController_DownloadPDF_NotFound_DN(t *testing.T) {
	// DN itself not found.
	repo := &stubDNRepo{dnNumberErr: &responses.InternalResponse{
		Message:    "Nota de entrega no encontrada",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}}
	r := dnGin(dnController(repo))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/delivery-notes/dn-1/pdf", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDNController_DownloadPDF_NotYetGenerated(t *testing.T) {
	// DN exists but PDF not yet on disk.
	repo := &stubDNRepo{dnNumber: "DN-2026-0001"}
	r := dnGin(dnController(repo))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/delivery-notes/nonexistent-dn/pdf", nil)
	r.ServeHTTP(w, req)

	// PDF file won't exist → 202 Accepted.
	require.Equal(t, http.StatusAccepted, w.Code)
}
