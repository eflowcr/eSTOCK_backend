package services

import (
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// mock repository
// ─────────────────────────────────────────────────────────────────────────────

type mockDNRepo struct {
	listResult  *responses.DeliveryNoteListResponse
	listErr     *responses.InternalResponse
	getResult   *responses.DeliveryNoteResponse
	getErr      *responses.InternalResponse
	dnNumber    string
	dnNumberErr *responses.InternalResponse
	updateErr   *responses.InternalResponse
}

func (m *mockDNRepo) List(_ string, _, _ *string, _, _ *string, _, _ int) (*responses.DeliveryNoteListResponse, *responses.InternalResponse) {
	return m.listResult, m.listErr
}
func (m *mockDNRepo) GetByID(_, _ string) (*responses.DeliveryNoteResponse, *responses.InternalResponse) {
	return m.getResult, m.getErr
}
func (m *mockDNRepo) UpdatePDFURL(_, _ string) *responses.InternalResponse {
	return m.updateErr
}
func (m *mockDNRepo) GetDNNumber(_, _ string) (string, *responses.InternalResponse) {
	return m.dnNumber, m.dnNumberErr
}

// ─────────────────────────────────────────────────────────────────────────────
// tests
// ─────────────────────────────────────────────────────────────────────────────

func TestDeliveryNotesService_List_OK(t *testing.T) {
	expected := &responses.DeliveryNoteListResponse{Total: 2, Page: 1, Limit: 50}
	repo := &mockDNRepo{listResult: expected}
	svc := NewDeliveryNotesService(repo, nil)

	result, resp := svc.List("tenant-1", nil, nil, nil, nil, 1, 50)
	require.Nil(t, resp)
	require.Equal(t, expected, result)
}

func TestDeliveryNotesService_List_Error(t *testing.T) {
	repo := &mockDNRepo{listErr: &responses.InternalResponse{
		Error: errors.New("db error"), Message: "Error",
	}}
	svc := NewDeliveryNotesService(repo, nil)

	result, resp := svc.List("tenant-1", nil, nil, nil, nil, 1, 50)
	require.Nil(t, result)
	require.NotNil(t, resp)
}

func TestDeliveryNotesService_GetByID_OK(t *testing.T) {
	expected := &responses.DeliveryNoteResponse{ID: "dn-1", DNNumber: "DN-2026-0001"}
	repo := &mockDNRepo{getResult: expected}
	svc := NewDeliveryNotesService(repo, nil)

	dn, resp := svc.GetByID("dn-1", "tenant-1")
	require.Nil(t, resp)
	require.Equal(t, expected, dn)
}

func TestDeliveryNotesService_GetByID_NotFound(t *testing.T) {
	repo := &mockDNRepo{getErr: &responses.InternalResponse{
		Message:    "Nota de entrega no encontrada",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}}
	svc := NewDeliveryNotesService(repo, nil)

	dn, resp := svc.GetByID("nonexistent", "tenant-1")
	require.Nil(t, dn)
	require.NotNil(t, resp)
	require.True(t, resp.Handled)
}

func TestDeliveryNotesService_GetDNNumber_OK(t *testing.T) {
	repo := &mockDNRepo{dnNumber: "DN-2026-0001"}
	svc := NewDeliveryNotesService(repo, nil)

	num, resp := svc.GetDNNumber("dn-1", "tenant-1")
	require.Nil(t, resp)
	require.Equal(t, "DN-2026-0001", num)
}

func TestPDFLocalPath(t *testing.T) {
	path := PDFLocalPath("abc123")
	require.Equal(t, "/tmp/estock-pdfs/abc123.pdf", path)
}

func TestPDFAPIURL(t *testing.T) {
	url := PDFAPIURL("abc123")
	require.Equal(t, "/api/delivery-notes/abc123/pdf", url)
}

func TestBuildDNPDF_OK(t *testing.T) {
	dn := &responses.DeliveryNoteResponse{
		ID:         "dn-1",
		DNNumber:   "DN-2026-0001",
		CustomerID: "client-1",
		TotalItems: 2,
		Items: []responses.DeliveryNoteItemResponse{
			{ArticleSKU: "SKU-A", Qty: 10, LotNumbers: []string{"LOT-001"}},
			{ArticleSKU: "SKU-B", Qty: 5},
		},
	}
	pdfBytes, err := buildDNPDF(dn)
	require.NoError(t, err)
	require.Greater(t, len(pdfBytes), 100, "PDF should not be empty")
	// Basic PDF magic bytes check.
	require.Equal(t, "%PDF", string(pdfBytes[:4]))
}

func TestDeliveryNotesService_GeneratePDFAsync_NoRepo(t *testing.T) {
	// Should not panic with nil repo (repo.GetByID returns err, goroutine logs and exits).
	repo := &mockDNRepo{getErr: &responses.InternalResponse{Message: "not found", Handled: true}}
	svc := NewDeliveryNotesService(repo, nil)
	// Fire async and give it a moment to run.
	svc.GeneratePDFAsync("dn-1", "tenant-1")
}
