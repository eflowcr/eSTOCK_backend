package repositories

import (
	"fmt"
	_ "image/png" // register PNG decoder for excelize.AddPictureFromBytes
	"strings"

	"github.com/eflowcr/eSTOCK_backend/assets"
	"github.com/xuri/excelize/v2"
)

const (
	articleTemplateDataStartRow = 9 // Data starts at row 9 (row 8 is example) — shifted up after removing spacer row 5
	articleTemplateDataEndRow   = 2000
)

// ePRAC Brand Colors - Matching the Excel design
const (
	epracBlue       = "0066CC" // Primary brand blue
	epracBlueDark   = "0052A3" // Darker blue for required fields (SKU, Name)
	epracBlueLight  = "3399FF" // Lighter blue for highlights
	epracNavy       = "0B1F3A" // Dark navy (logo text color)
	epracGray       = "6B7280" // Medium gray for subtitle text
	epracGrayLight  = "F3F4F6" // Very light gray for alternating rows
	epracGrayBorder = "E5E7EB" // Border gray
	epracWhite      = "FFFFFF" // White
	epracAccent     = "EEF2FF" // Very light blue tint for logo area & instructions
	epracExample    = "FFF9E6" // Light cream for example row
)

// articleTemplateLang holds all user-facing strings for the import template.
// Supported languages: "es" (default), "en". Others fall back to "es".
var articleTemplateLang = map[string]map[string]string{
	"es": {
		"title":         "Importar Artículos",
		"subtitle":      "Plantilla de importación de artículos — eSTOCK",
		"instructions":  "📋 Instrucciones",
		"inst_content":  "1. Complete los datos en las filas desde la 9 en adelante  •  2. Use las listas desplegables para campos con opciones predefinidas  •  3. Los campos en negrita son obligatorios (SKU, Nombre)  •  4. Puede importar hasta 2,000 artículos a la vez",
		"sheet_data":    "Artículos",
		"sheet_opts":    "Opciones",
		"col_sku":       "SKU *",
		"col_name":      "Nombre *",
		"col_desc":      "Descripción",
		"col_price":     "Precio",
		"col_pres":      "Presentación",
		"col_lot":       "Rastrear por lote",
		"col_serial":    "Rastrear por serie",
		"col_exp":       "Rastrear por expiración",
		"col_max":       "Cantidad Máxima",
		"col_min":       "Cantidad Mínima",
		"col_rotation":  "Estrategia de Rotación",
		"yes":           "Si",
		"no":            "No",
		"example_sku":   "ART-001",
		"example_name":  "Laptop Dell Inspiron 15",
		"example_desc":  "Laptop empresarial, 16GB RAM, 512GB SSD",
		"example_price": "899.99",
		"example_pres":  "unidad",
		"example_max":   "100",
		"example_min":   "10",
		"err_pres":      "Presentación inválida",
		"err_pres_msg":  "Seleccione una presentación de la lista",
		"err_bool":      "Valor inválido",
		"err_bool_msg":  "Use Si o No",
		"err_rot":       "Rotación inválida",
		"err_rot_msg":   "Use fifo o fefo",
		"err_price":     "Precio inválido",
		"err_price_msg": "Ingrese un precio mayor o igual a 0",
		"err_qty":       "Cantidad inválida",
		"err_qty_msg":   "Ingrese un número entero mayor o igual a 0",
	},
	"en": {
		"title":         "Import Articles",
		"subtitle":      "Article import template — eSTOCK",
		"instructions":  "📋 Instructions",
		"inst_content":  "1. Fill in data starting from row 9 onwards  •  2. Use dropdown lists for fields with predefined options  •  3. Fields in bold are required (SKU, Name)  •  4. You can import up to 2,000 articles at once",
		"sheet_data":    "Articles",
		"sheet_opts":    "Options",
		"col_sku":       "SKU *",
		"col_name":      "Name *",
		"col_desc":      "Description",
		"col_price":     "Unit Price",
		"col_pres":      "Presentation",
		"col_lot":       "Track by Lot",
		"col_serial":    "Track by Serial",
		"col_exp":       "Track Expiration",
		"col_max":       "Max Quantity",
		"col_min":       "Min Quantity",
		"col_rotation":  "Rotation Strategy",
		"yes":           "Yes",
		"no":            "No",
		"example_sku":   "ART-001",
		"example_name":  "Dell Inspiron 15 Laptop",
		"example_desc":  "Business laptop, 16GB RAM, 512GB SSD",
		"example_price": "899.99",
		"example_pres":  "unit",
		"example_max":   "100",
		"example_min":   "10",
		"err_pres":      "Invalid presentation",
		"err_pres_msg":  "Select a presentation from the list",
		"err_bool":      "Invalid value",
		"err_bool_msg":  "Use Yes or No",
		"err_rot":       "Invalid rotation",
		"err_rot_msg":   "Use fifo or fefo",
		"err_price":     "Invalid price",
		"err_price_msg": "Enter a price greater than or equal to 0",
		"err_qty":       "Invalid quantity",
		"err_qty_msg":   "Enter an integer greater than or equal to 0",
	},
}

func getLang(language string) map[string]string {
	if l, ok := articleTemplateLang[language]; ok {
		return l
	}
	return articleTemplateLang["es"]
}

// applyArticleTemplateValidations sets up:
//   - A hidden "Options/Opciones" sheet with all dropdown source lists
//   - Data validations on the main data sheet referencing the hidden sheet
func applyArticleTemplateValidations(f *excelize.File, dataSheet string, presentationOptions []string, language string) error {
	l := getLang(language)
	opts := uniqueNonEmptyStrings(presentationOptions)
	if len(opts) == 0 {
		opts = []string{"unidad", "caja", "pallet", "paquete"}
	}
	yesNo := []string{l["yes"], l["no"]}

	// --- Hidden options sheet ---
	optSheet := l["sheet_opts"]
	f.NewSheet(optSheet)

	// Col A: presentations
	for i, opt := range opts {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetCellValue(optSheet, cell, opt); err != nil {
			return err
		}
	}
	// Col B: yes / no
	for i, v := range yesNo {
		cell, _ := excelize.CoordinatesToCellName(2, i+1)
		if err := f.SetCellValue(optSheet, cell, v); err != nil {
			return err
		}
	}
	// Col C: rotation strategies
	for i, v := range []string{"fifo", "fefo"} {
		cell, _ := excelize.CoordinatesToCellName(3, i+1)
		if err := f.SetCellValue(optSheet, cell, v); err != nil {
			return err
		}
	}

	// Hide the options sheet
	if err := f.SetSheetVisible(optSheet, false); err != nil {
		return err
	}

	presRef := fmt.Sprintf("'%s'!$A$1:$A$%d", optSheet, len(opts))
	yesNoRef := fmt.Sprintf("'%s'!$B$1:$B$2", optSheet)
	rotRef := fmt.Sprintf("'%s'!$C$1:$C$2", optSheet)

	// Presentation dropdown (col E)
	if err := addDropListValidation(f, dataSheet,
		fmt.Sprintf("E%d:E%d", articleTemplateDataStartRow, articleTemplateDataEndRow),
		presRef, l["err_pres"], l["err_pres_msg"],
	); err != nil {
		return err
	}

	// Yes/No dropdown (cols F-H)
	if err := addDropListValidation(f, dataSheet,
		fmt.Sprintf("F%d:H%d", articleTemplateDataStartRow, articleTemplateDataEndRow),
		yesNoRef, l["err_bool"], l["err_bool_msg"],
	); err != nil {
		return err
	}

	// Rotation dropdown (col K)
	if err := addDropListValidation(f, dataSheet,
		fmt.Sprintf("K%d:K%d", articleTemplateDataStartRow, articleTemplateDataEndRow),
		rotRef, l["err_rot"], l["err_rot_msg"],
	); err != nil {
		return err
	}

	// Numeric: price (col D)
	if err := addNumericMinValidation(f, dataSheet,
		fmt.Sprintf("D%d:D%d", articleTemplateDataStartRow, articleTemplateDataEndRow),
		excelize.DataValidationTypeDecimal,
		l["err_price"], l["err_price_msg"],
	); err != nil {
		return err
	}

	// Numeric: quantities (cols I-J)
	if err := addNumericMinValidation(f, dataSheet,
		fmt.Sprintf("I%d:J%d", articleTemplateDataStartRow, articleTemplateDataEndRow),
		excelize.DataValidationTypeWhole,
		l["err_qty"], l["err_qty_msg"],
	); err != nil {
		return err
	}

	return nil
}

// applyArticleTemplateHeader writes the styled header block with logo and title
func applyArticleTemplateHeader(f *excelize.File, dataSheet string, language string) error {
	l := getLang(language)

	// --- ROWS 1-4: Logo and Title Section ---

	// Logo area style (light blue accent) — A1:B4
	logoStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{epracAccent}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return err
	}

	// Title style (bold, large, navy) — C1:K2
	titleStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{epracWhite}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Size: 22, Color: epracNavy, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return err
	}

	// Subtitle style (italic, gray) with dotted top border — C3:K4
	subStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{epracWhite}, Pattern: 1},
		Font:      &excelize.Font{Size: 11, Color: epracGray, Italic: true, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "top", Color: epracGrayBorder, Style: 4}, // Style 4 = dotted (dashed/punteado)
		},
	})
	if err != nil {
		return err
	}

	// Merge cells — new layout
	if err := f.MergeCell(dataSheet, "A1", "B4"); err != nil {
		return err
	}
	if err := f.MergeCell(dataSheet, "C1", "K2"); err != nil {
		return err
	}
	if err := f.MergeCell(dataSheet, "C3", "K4"); err != nil {
		return err
	}

	// Apply styles
	if err := f.SetCellStyle(dataSheet, "A1", "B4", logoStyle); err != nil {
		return err
	}
	if err := f.SetCellStyle(dataSheet, "C1", "K2", titleStyle); err != nil {
		return err
	}
	if err := f.SetCellStyle(dataSheet, "C3", "K4", subStyle); err != nil {
		return err
	}

	// Set text
	if err := f.SetCellValue(dataSheet, "C1", l["title"]); err != nil {
		return err
	}
	if err := f.SetCellValue(dataSheet, "C3", l["subtitle"]); err != nil {
		return err
	}

	// Set row heights (row 5 removed)
	heights := map[int]float64{1: 26, 2: 26, 3: 16, 4: 16}
	for row, h := range heights {
		if err := f.SetRowHeight(dataSheet, row, h); err != nil {
			return err
		}
	}

	// Add logo — now in A1:B4
	// Target size: 196 pt W × 53 pt H, positioned at offset (14, 20)
	if len(assets.LogoEPRAC) > 0 {
		if err := f.AddPictureFromBytes(dataSheet, "A1", &excelize.Picture{
			Extension:  ".png",
			File:       assets.LogoEPRAC,
			InsertType: excelize.PictureInsertTypePlaceOverCells,
			Format: &excelize.GraphicOptions{
				OffsetX:         14,
				OffsetY:         20,
				ScaleX:          0.108,
				ScaleY:          0.246,
				LockAspectRatio: false,
			},
		}); err != nil {
			_ = err
		}
	}

	// --- ROWS 5-6: Instructions Section (shifted up from 6-7) ---
	if err := applyInstructionsSection(f, dataSheet, language); err != nil {
		return err
	}

	return nil
}

// applyInstructionsSection adds the instructions box (rows 5-6, shifted up from 6-7)
func applyInstructionsSection(f *excelize.File, dataSheet string, language string) error {
	l := getLang(language)

	// Instructions title style (blue background, white bold text)
	titleStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{epracBlue}, Pattern: 1},
		Font: &excelize.Font{Bold: true, Size: 10, Color: epracWhite, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	})
	if err != nil {
		return err
	}

	// Instructions content style (light blue background)
	contentStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{epracAccent}, Pattern: 1},
		Font: &excelize.Font{Size: 9, Color: epracNavy, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "top",
			WrapText:   true,
		},
	})
	if err != nil {
		return err
	}

	// Merge cells (shifted up by 1 row)
	if err := f.MergeCell(dataSheet, "A5", "K5"); err != nil {
		return err
	}
	if err := f.MergeCell(dataSheet, "A6", "K6"); err != nil {
		return err
	}

	// Apply styles
	if err := f.SetCellStyle(dataSheet, "A5", "K5", titleStyle); err != nil {
		return err
	}
	if err := f.SetCellStyle(dataSheet, "A6", "K6", contentStyle); err != nil {
		return err
	}

	// Set content
	if err := f.SetCellValue(dataSheet, "A5", l["instructions"]); err != nil {
		return err
	}
	if err := f.SetCellValue(dataSheet, "A6", l["inst_content"]); err != nil {
		return err
	}

	// Set row heights
	if err := f.SetRowHeight(dataSheet, 5, 20); err != nil {
		return err
	}
	if err := f.SetRowHeight(dataSheet, 6, 32); err != nil {
		return err
	}

	return nil
}

// applyArticleTemplateColumnHeaders writes column headers (row 7, shifted up from 8)
func applyArticleTemplateColumnHeaders(f *excelize.File, dataSheet string, language string) error {
	l := getLang(language)

	// Required field style (darker blue for SKU and Name)
	requiredStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{epracBlueDark}, Pattern: 1},
		Font: &excelize.Font{Bold: true, Size: 11, Color: epracWhite, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
	})
	if err != nil {
		return err
	}

	// Regular header style (blue)
	headerStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{epracBlue}, Pattern: 1},
		Font: &excelize.Font{Bold: true, Size: 11, Color: epracWhite, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
	})
	if err != nil {
		return err
	}

	headers := []struct {
		text     string
		required bool
	}{
		{l["col_sku"], true},       // A - Required
		{l["col_name"], true},      // B - Required
		{l["col_desc"], false},     // C
		{l["col_price"], false},    // D
		{l["col_pres"], false},     // E
		{l["col_lot"], false},      // F
		{l["col_serial"], false},   // G
		{l["col_exp"], false},      // H
		{l["col_max"], false},      // I
		{l["col_min"], false},      // J
		{l["col_rotation"], false}, // K
	}

	// Column widths (matching Excel design)
	colWidths := []float64{14, 46.67, 32, 12, 16, 13, 13, 18, 14, 13, 18}
	colNames := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K"}
	for i, w := range colWidths {
		if err := f.SetColWidth(dataSheet, colNames[i], colNames[i], w); err != nil {
			return err
		}
	}

	// Set header row height (row 7 now)
	if err := f.SetRowHeight(dataSheet, 7, 36); err != nil {
		return err
	}

	// Set headers (row 7 now)
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 7)
		if err := f.SetCellValue(dataSheet, cell, h.text); err != nil {
			return err
		}
		if h.required {
			if err := f.SetCellStyle(dataSheet, cell, cell, requiredStyle); err != nil {
				return err
			}
		} else {
			if err := f.SetCellStyle(dataSheet, cell, cell, headerStyle); err != nil {
				return err
			}
		}
	}

	// Add example row (row 8 now)
	if err := applyExampleRow(f, dataSheet, language); err != nil {
		return err
	}

	// Add alternating row colors (rows 9+)
	if err := applyAlternatingRowColors(f, dataSheet); err != nil {
		return err
	}

	return nil
}

// applyExampleRow adds example data in row 8 (shifted up from 9) with cream background
func applyExampleRow(f *excelize.File, dataSheet string, language string) error {
	l := getLang(language)

	// Example row style (light cream background)
	exampleStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{epracExample}, Pattern: 1},
		Font:      &excelize.Font{Size: 10, Color: epracGray, Italic: true, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return err
	}

	exampleData := []string{
		l["example_sku"],
		l["example_name"],
		l["example_desc"],
		l["example_price"],
		l["example_pres"],
		l["no"],
		l["no"],
		l["no"],
		l["example_max"],
		l["example_min"],
		"fifo",
	}

	for i, val := range exampleData {
		cell, _ := excelize.CoordinatesToCellName(i+1, 8) // Row 8 now
		if err := f.SetCellValue(dataSheet, cell, val); err != nil {
			return err
		}
		if err := f.SetCellStyle(dataSheet, cell, cell, exampleStyle); err != nil {
			return err
		}
	}

	// Set example row height
	if err := f.SetRowHeight(dataSheet, 8, 22); err != nil {
		return err
	}

	return nil
}

// applyAlternatingRowColors adds alternating colors to data rows (9+, shifted up from 10+)
func applyAlternatingRowColors(f *excelize.File, dataSheet string) error {
	// Light gray style
	grayStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{epracGrayLight}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	// Apply to first 50 rows as sample (starting from row 9 now)
	for row := 9; row <= 59; row++ {
		if row%2 != 0 { // Row 9 = gray, 10 = white, 11 = gray, etc.
			for col := 1; col <= 11; col++ {
				cell, _ := excelize.CoordinatesToCellName(col, row)
				if err := f.SetCellStyle(dataSheet, cell, cell, grayStyle); err != nil {
					return err
				}
			}
		}
		// Set standard row height
		if err := f.SetRowHeight(dataSheet, row, 20); err != nil {
			return err
		}
	}

	return nil
}

func addDropListValidation(f *excelize.File, sheet, sqref, sourceRef, errorTitle, errorBody string) error {
	dv := excelize.NewDataValidation(true)
	dv.Sqref = sqref
	dv.SetSqrefDropList(sourceRef)
	dv.SetError(excelize.DataValidationErrorStyleStop, errorTitle, errorBody)
	return f.AddDataValidation(sheet, dv)
}

func addNumericMinValidation(
	f *excelize.File,
	sheet, sqref string,
	validationType excelize.DataValidationType,
	errorTitle, errorBody string,
) error {
	dv := excelize.NewDataValidation(true)
	dv.Sqref = sqref
	if err := dv.SetRange(0, 0, validationType, excelize.DataValidationOperatorGreaterThanOrEqual); err != nil {
		return err
	}
	dv.SetError(excelize.DataValidationErrorStyleStop, errorTitle, errorBody)
	return f.AddDataValidation(sheet, dv)
}

func uniqueNonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v == "" {
			continue
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	return out
}
