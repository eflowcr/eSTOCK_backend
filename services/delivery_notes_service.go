package services

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"
)

// DeliveryNotesService provides business logic for delivery notes (DN1-DN3).
type DeliveryNotesService struct {
	Repository ports.DeliveryNotesRepository
	DB         *gorm.DB
}

func NewDeliveryNotesService(repo ports.DeliveryNotesRepository, db *gorm.DB) *DeliveryNotesService {
	return &DeliveryNotesService{Repository: repo, DB: db}
}

// List returns paginated delivery notes for a tenant.
func (s *DeliveryNotesService) List(tenantID string, customerID, soNumber *string, from, to *string, page, limit int) (*responses.DeliveryNoteListResponse, *responses.InternalResponse) {
	return s.Repository.List(tenantID, customerID, soNumber, from, to, page, limit)
}

// GetByID returns a full delivery note by ID.
func (s *DeliveryNotesService) GetByID(id, tenantID string) (*responses.DeliveryNoteResponse, *responses.InternalResponse) {
	return s.Repository.GetByID(id, tenantID)
}

// GetDNNumber returns the dn_number for a given delivery note (used for PDF download filename).
func (s *DeliveryNotesService) GetDNNumber(id, tenantID string) (string, *responses.InternalResponse) {
	return s.Repository.GetDNNumber(id, tenantID)
}

// PDFLocalPath returns the filesystem path for a DN's PDF.
func PDFLocalPath(dnID string) string {
	return filepath.Join("/tmp/estock-pdfs", dnID+".pdf")
}

// PDFAPIURL returns the API URL for downloading a DN's PDF.
func PDFAPIURL(dnID string) string {
	return "/api/delivery-notes/" + dnID + "/pdf"
}

// ─────────────────────────────────────────────────────────────────────────────
// DN2 — PDF generation (async goroutine, implements repositories.DNPDFGenerator)
// ─────────────────────────────────────────────────────────────────────────────

// GeneratePDFAsync implements repositories.DNPDFGenerator.
// Spawns a goroutine that generates the PDF and writes it to local FS.
func (s *DeliveryNotesService) GeneratePDFAsync(dnID, tenantID string) {
	go s.generatePDF(dnID, tenantID)
}

func (s *DeliveryNotesService) generatePDF(dnID, tenantID string) {
	dn, resp := s.Repository.GetByID(dnID, tenantID)
	if resp != nil {
		fmt.Printf("[WARN] GeneratePDF: fetch DN %s: %s\n", dnID, resp.Message)
		return
	}

	pdfBytes, err := buildDNPDF(dn)
	if err != nil {
		fmt.Printf("[WARN] GeneratePDF: build PDF for %s: %v\n", dnID, err)
		return
	}

	// Ensure directory exists.
	dir := "/tmp/estock-pdfs"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("[WARN] GeneratePDF: mkdir %s: %v\n", dir, err)
		return
	}

	outPath := PDFLocalPath(dnID)
	if err := os.WriteFile(outPath, pdfBytes, 0644); err != nil {
		fmt.Printf("[WARN] GeneratePDF: write %s: %v\n", outPath, err)
		return
	}

	pdfURL := PDFAPIURL(dnID)
	if resp := s.Repository.UpdatePDFURL(dnID, pdfURL); resp != nil {
		fmt.Printf("[WARN] GeneratePDF: update pdf_url %s: %s\n", dnID, resp.Message)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// buildDNPDF — PDF layout with gofpdf
// ─────────────────────────────────────────────────────────────────────────────

// buildDNPDF constructs PDF bytes for a delivery note.
func buildDNPDF(dn *responses.DeliveryNoteResponse) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 20, 15)
	pdf.AddPage()

	// ── Header ──────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 18)
	pdf.Cell(0, 10, "eSTOCK - Delivery Note")
	pdf.Ln(12)

	pdf.SetFont("Helvetica", "", 11)
	customerName := dn.CustomerID
	if dn.CustomerName != nil && *dn.CustomerName != "" {
		customerName = *dn.CustomerName
	}
	pdf.Cell(90, 7, fmt.Sprintf("DN Number: %s", dn.DNNumber))
	pdf.Cell(0, 7, fmt.Sprintf("Date: %s", dn.CreatedAt.Format("2006-01-02")))
	pdf.Ln(8)
	pdf.Cell(90, 7, fmt.Sprintf("Sales Order: %s", dn.SalesOrderID))
	pdf.Cell(0, 7, fmt.Sprintf("Customer: %s", customerName))
	pdf.Ln(8)
	if dn.PickingTaskID != nil {
		pdf.Cell(0, 7, fmt.Sprintf("Picking Task: %s", *dn.PickingTaskID))
		pdf.Ln(8)
	}
	pdf.Ln(4)

	// ── Items table ──────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(220, 220, 220)
	pdf.CellFormat(20, 8, "#", "1", 0, "C", true, 0, "")
	pdf.CellFormat(50, 8, "SKU", "1", 0, "L", true, 0, "")
	pdf.CellFormat(30, 8, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(0, 8, "Lot Numbers", "1", 1, "L", true, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetFillColor(255, 255, 255)
	for i, item := range dn.Items {
		lots := strings.Join(item.LotNumbers, ", ")
		pdf.CellFormat(20, 7, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 7, item.ArticleSKU, "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, 7, fmt.Sprintf("%.3f", item.Qty), "1", 0, "C", false, 0, "")
		pdf.CellFormat(0, 7, lots, "1", 1, "L", false, 0, "")
	}
	pdf.Ln(4)

	// ── Totals ───────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(0, 7, fmt.Sprintf("Total Items: %d", dn.TotalItems))
	pdf.Ln(14)

	// ── Signature line ────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "", 10)
	pdf.Cell(80, 7, "_______________________________")
	pdf.Cell(0, 7, "_______________________________")
	pdf.Ln(6)
	pdf.Cell(80, 7, "Received By (Signature)")
	pdf.Cell(0, 7, "Delivered By (Signature)")
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.Cell(0, 6, fmt.Sprintf("Generated: %s", time.Now().Format(time.RFC3339)))

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return buf.Bytes(), nil
}

// compile-time check: DeliveryNotesService satisfies ports.DeliveryNotesRepository indirectly
// and repositories.DNPDFGenerator via GeneratePDFAsync.
var _ interface{ GeneratePDFAsync(dnID, tenantID string) } = (*DeliveryNotesService)(nil)
