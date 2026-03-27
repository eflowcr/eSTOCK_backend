package repositories

import (
	"bytes"
	"fmt"
	_ "image/png" // register PNG decoder for AddPictureFromBytes

	"github.com/eflowcr/eSTOCK_backend/assets"
	"github.com/xuri/excelize/v2"
)

// ColumnDef describes a single column in the import template.
type ColumnDef struct {
	Header   string
	Required bool
	Width    float64
}

// ModuleTemplateConfig holds every piece of data needed to build a
// professional-looking import template that mirrors the design.
type ModuleTemplateConfig struct {
	DataSheetName string
	LogoOffsetX   int
	LogoOffsetY   int
	LogoScaleX    float64 // 0 = default 0.247
	LogoScaleY    float64 // 0 = default 0.236
	LogoAnchor    string  // "" = default "A1"
	OptSheetName  string
	Title         string // e.g. "Importar Ubicaciones"
	Subtitle      string // e.g. "Plantilla de importación — eSTOCK"
	InstrTitle    string // e.g. "📋 Instrucciones"
	InstrContent  string // instruction body text
	Columns       []ColumnDef
	ExampleRow    []string
	// ApplyValidations adds module-specific dropdown / numeric validations.
	ApplyValidations func(f *excelize.File, dataSheet, optSheet string, start, end int) error
}

// ── Colour palette ──────────────────────────────────────────────────────────
const (
	colBrandBlue     = "1B3A6B" // dark navy for title
	colSubtitleGray  = "6B7280" // muted gray for subtitle
	colInstrBg       = "1F2937" // dark header for instructions badge
	colInstrBodyBg   = "F3F4F6" // light gray bg for instructions body
	colInstrBodyText = "374151" // dark gray text
	colHeaderBg      = "1D4ED8" // vivid blue column header bg
	colHeaderText    = "FFFFFF" // white
	colExampleText   = "9CA3AF" // light gray for example row
	colStripeBg      = "F9FAFB" // subtle stripe
	colBorderLight   = "E5E7EB" // thin border colour
	colWhite         = "FFFFFF"
)

// BuildModuleImportTemplate creates the .xlsx bytes for an import template.
func BuildModuleImportTemplate(cfg ModuleTemplateConfig) ([]byte, error) {
	f := excelize.NewFile()
	sheet := cfg.DataSheetName
	f.SetSheetName("Sheet1", sheet)

	numCols := len(cfg.Columns)
	lastCol, _ := excelize.ColumnNumberToName(numCols)

	// Page setup
	f.SetPageLayout(sheet, &excelize.PageLayoutOptions{
		Orientation: ptrStr("portrait"),
	})

	// Column widths
	for i, col := range cfg.Columns {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, colName, colName, col.Width)
	}

	// ── Rows 1-2: Logo area ─────────────────────────────────────────────────
	f.SetRowHeight(sheet, 1, 10)
	f.SetRowHeight(sheet, 2, 50)
	f.MergeCell(sheet, "A1", fmt.Sprintf("%s1", lastCol))
	f.MergeCell(sheet, "A2", fmt.Sprintf("%s2", lastCol))

	if len(assets.LogoEPRAC) > 0 {
		anchor := cfg.LogoAnchor
		if anchor == "" {
			anchor = "A1"
		}
		scaleX := cfg.LogoScaleX
		if scaleX == 0 {
			scaleX = 0.108
		}
		scaleY := cfg.LogoScaleY
		if scaleY == 0 {
			scaleY = 0.246
		}
		_ = f.AddPictureFromBytes(sheet, anchor, &excelize.Picture{
			Extension:  ".png",
			File:       assets.LogoEPRAC,
			InsertType: excelize.PictureInsertTypePlaceOverCells,
			Format: &excelize.GraphicOptions{
				OffsetX:         cfg.LogoOffsetX,
				OffsetY:         cfg.LogoOffsetY,
				ScaleX:          scaleX,
				ScaleY:          scaleY,
				LockAspectRatio: false,
			},
		})
	}

	// ── Row 3: Title ────────────────────────────────────────────────────────
	f.SetRowHeight(sheet, 3, 36)
	f.MergeCell(sheet, "A3", fmt.Sprintf("%s3", lastCol))
	f.SetCellValue(sheet, "A3", cfg.Title)

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 20, Bold: true, Color: colBrandBlue},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheet, "A3", fmt.Sprintf("%s3", lastCol), titleStyle)

	// ── Row 4: Subtitle ─────────────────────────────────────────────────────
	f.SetRowHeight(sheet, 4, 20)
	f.MergeCell(sheet, "A4", fmt.Sprintf("%s4", lastCol))
	f.SetCellValue(sheet, "A4", cfg.Subtitle)

	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Italic: true, Color: colSubtitleGray},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheet, "A4", fmt.Sprintf("%s4", lastCol), subtitleStyle)

	// ── Row 5: Spacer ────────────────────────────────────────────────────────
	f.SetRowHeight(sheet, 5, 10)

	// ── Row 6: Instructions badge ────────────────────────────────────────────
	f.SetRowHeight(sheet, 6, 22)
	f.MergeCell(sheet, "A6", fmt.Sprintf("%s6", lastCol))
	f.SetCellValue(sheet, "A6", cfg.InstrTitle)

	instrBadgeStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Bold: true, Color: colWhite},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colInstrBg}},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", Indent: 1},
		Border: []excelize.Border{
			{Type: "top", Color: colInstrBg, Style: 1},
			{Type: "left", Color: colInstrBg, Style: 1},
			{Type: "right", Color: colInstrBg, Style: 1},
		},
	})
	f.SetCellStyle(sheet, "A6", fmt.Sprintf("%s6", lastCol), instrBadgeStyle)

	// ── Row 7: Instructions body ─────────────────────────────────────────────
	f.SetRowHeight(sheet, 7, 36)
	f.MergeCell(sheet, "A7", fmt.Sprintf("%s7", lastCol))
	f.SetCellValue(sheet, "A7", cfg.InstrContent)

	instrBodyStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9, Color: colInstrBodyText},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colInstrBodyBg}},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true, Indent: 1},
		Border: []excelize.Border{
			{Type: "bottom", Color: colBorderLight, Style: 1},
			{Type: "left", Color: colInstrBodyBg, Style: 1},
			{Type: "right", Color: colInstrBodyBg, Style: 1},
		},
	})
	f.SetCellStyle(sheet, "A7", fmt.Sprintf("%s7", lastCol), instrBodyStyle)

	// ── Row 8: Column headers ─────────────────────────────────────────────────
	f.SetRowHeight(sheet, 8, 28)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 11, Bold: true, Color: colHeaderText},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colHeaderBg}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "top", Color: colHeaderBg, Style: 1},
			{Type: "bottom", Color: colHeaderBg, Style: 1},
			{Type: "left", Color: colHeaderBg, Style: 1},
			{Type: "right", Color: colHeaderBg, Style: 1},
		},
	})

	for i, col := range cfg.Columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 8)
		f.SetCellValue(sheet, cell, col.Header)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// ── Row 9: Example row ────────────────────────────────────────────────────
	f.SetRowHeight(sheet, 9, 22)

	exampleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Italic: true, Color: colExampleText},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", Indent: 1},
		Border:    []excelize.Border{{Type: "bottom", Color: colBorderLight, Style: 1}},
	})

	for i, val := range cfg.ExampleRow {
		if i >= numCols {
			break
		}
		cell, _ := excelize.CoordinatesToCellName(i+1, 9)
		f.SetCellValue(sheet, cell, val)
		f.SetCellStyle(sheet, cell, cell, exampleStyle)
	}

	// ── Rows 10-59: Pre-formatted data rows with alternating stripes ──────────
	whiteRowStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", Indent: 1},
		Border:    []excelize.Border{{Type: "bottom", Color: colBorderLight, Style: 1}},
	})
	stripeRowStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colStripeBg}},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", Indent: 1},
		Border:    []excelize.Border{{Type: "bottom", Color: colBorderLight, Style: 1}},
	})

	for row := 10; row <= 59; row++ {
		f.SetRowHeight(sheet, row, 22)
		style := whiteRowStyle
		if (row-10)%2 == 1 {
			style = stripeRowStyle
		}
		for col := 1; col <= numCols; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellStyle(sheet, cell, cell, style)
		}
	}

	// ── Freeze panes: keep column headers visible while scrolling ─────────────
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		XSplit:      0,
		YSplit:      8,
		TopLeftCell: "A9",
		ActivePane:  "bottomLeft",
	})

	// ── Hide gridlines for a cleaner look ─────────────────────────────────────
	f.SetSheetView(sheet, 0, &excelize.ViewOptions{
		ShowGridLines: ptrBool(false),
	})

	// ── Apply module-specific validations (dropdowns, etc.) ───────────────────
	if cfg.ApplyValidations != nil {
		if err := cfg.ApplyValidations(f, sheet, cfg.OptSheetName, 9, 2000); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("error writing xlsx: %w", err)
	}
	return buf.Bytes(), nil
}

// SharedDropListValidation adds a dropdown referencing a range on the options sheet.
func SharedDropListValidation(f *excelize.File, dataSheet, optSheet, dataRange, optRange, errTitle, errMsg string) error {
	ref := fmt.Sprintf("'%s'!%s", optSheet, optRange)
	dv := excelize.NewDataValidation(true)
	dv.Sqref = dataRange
	dv.SetSqrefDropList(ref)
	dv.SetError(excelize.DataValidationErrorStyleStop, errTitle, errMsg)
	return f.AddDataValidation(dataSheet, dv)
}

// SharedNumericValidation adds a numeric >= 0 validation.
func SharedNumericValidation(f *excelize.File, dataSheet, colRange string, vType excelize.DataValidationType, errTitle, errMsg string) error {
	dv := excelize.NewDataValidation(true)
	dv.Sqref = colRange
	if err := dv.SetRange(0, 0, vType, excelize.DataValidationOperatorGreaterThanOrEqual); err != nil {
		return err
	}
	dv.SetError(excelize.DataValidationErrorStyleStop, errTitle, errMsg)
	return f.AddDataValidation(dataSheet, dv)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func ptrStr(s string) *string { return &s }
func ptrBool(b bool) *bool    { return &b }
func ifStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
