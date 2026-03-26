package repositories

import (
	"fmt"
	"strings"

	"github.com/eflowcr/eSTOCK_backend/assets"
	"github.com/xuri/excelize/v2"
)

const (
	articleTemplateDataStartRow = 7
	articleTemplateDataEndRow   = 2000
)

// articleTemplateLang holds all user-facing strings for the import template.
// Supported languages: "es" (default), "en". Others fall back to "es".
var articleTemplateLang = map[string]map[string]string{
	"es": {
		"title":        "Importar Artículos",
		"subtitle":     "Plantilla de importación de artículos — eSTOCK",
		"sheet_data":   "Artículos",
		"sheet_opts":   "Opciones",
		"col_sku":      "SKU",
		"col_name":     "Nombre",
		"col_desc":     "Descripción",
		"col_price":    "Precio",
		"col_pres":     "Presentación",
		"col_lot":      "Rastrear por lote",
		"col_serial":   "Rastrear por serie",
		"col_exp":      "Rastrear por expiración",
		"col_max":      "Cantidad Máxima",
		"col_min":      "Cantidad Mínima",
		"col_rotation": "Estrategia de Rotación",
		"yes":          "Si",
		"no":           "No",
		"err_pres":     "Presentación inválida",
		"err_pres_msg": "Seleccione una presentación de la lista",
		"err_bool":     "Valor inválido",
		"err_bool_msg": "Use Si o No",
		"err_rot":      "Rotación inválida",
		"err_rot_msg":  "Use fifo o fefo",
		"err_price":    "Precio inválido",
		"err_price_msg": "Ingrese un precio mayor o igual a 0",
		"err_qty":      "Cantidad inválida",
		"err_qty_msg":  "Ingrese un número entero mayor o igual a 0",
	},
	"en": {
		"title":        "Import Articles",
		"subtitle":     "Article import template — eSTOCK",
		"sheet_data":   "Articles",
		"sheet_opts":   "Options",
		"col_sku":      "SKU",
		"col_name":     "Name",
		"col_desc":     "Description",
		"col_price":    "Unit Price",
		"col_pres":     "Presentation",
		"col_lot":      "Track by Lot",
		"col_serial":   "Track by Serial",
		"col_exp":      "Track Expiration",
		"col_max":      "Max Quantity",
		"col_min":      "Min Quantity",
		"col_rotation": "Rotation Strategy",
		"yes":          "Yes",
		"no":           "No",
		"err_pres":     "Invalid presentation",
		"err_pres_msg": "Select a presentation from the list",
		"err_bool":     "Invalid value",
		"err_bool_msg": "Use Yes or No",
		"err_rot":      "Invalid rotation",
		"err_rot_msg":  "Use fifo or fefo",
		"err_price":    "Invalid price",
		"err_price_msg": "Enter a price greater than or equal to 0",
		"err_qty":      "Invalid quantity",
		"err_qty_msg":  "Enter an integer greater than or equal to 0",
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
		opts = []string{"unidad"}
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

// applyArticleTemplateHeader writes the styled header block (rows 1–5) with
// the ePRAC logo and localized title on the data sheet.
func applyArticleTemplateHeader(f *excelize.File, dataSheet string, language string) error {
	l := getLang(language)

	const headerBg = "1E3A5F"   // ePRAC dark blue
	const headerFg = "FFFFFF"   // white text
	const subFg = "B0C4DE"     // light steel blue for subtitle

	headerStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{headerBg}, Pattern: 1},
		Font: &excelize.Font{Bold: true, Size: 20, Color: headerFg, Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return err
	}

	subStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{headerBg}, Pattern: 1},
		Font: &excelize.Font{Size: 11, Color: subFg, Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return err
	}

	// Logo area background (A1:D5)
	logoStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{headerBg}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return err
	}

	// Fill all header cells with the dark blue background first
	for row := 1; row <= 5; row++ {
		for col := 1; col <= 11; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			if err := f.SetCellStyle(dataSheet, cell, cell, logoStyle); err != nil {
				return err
			}
		}
	}

	// Merge A1:D4 → logo area
	if err := f.MergeCell(dataSheet, "A1", "D4"); err != nil {
		return err
	}
	// Merge A5:D5 → empty bottom of logo area
	if err := f.MergeCell(dataSheet, "A5", "D5"); err != nil {
		return err
	}
	// Merge E1:K3 → title
	if err := f.MergeCell(dataSheet, "E1", "K3"); err != nil {
		return err
	}
	// Merge E4:K5 → subtitle
	if err := f.MergeCell(dataSheet, "E4", "K5"); err != nil {
		return err
	}

	if err := f.SetCellStyle(dataSheet, "E1", "K3", headerStyle); err != nil {
		return err
	}
	if err := f.SetCellStyle(dataSheet, "E4", "K5", subStyle); err != nil {
		return err
	}

	if err := f.SetCellValue(dataSheet, "E1", l["title"]); err != nil {
		return err
	}
	if err := f.SetCellValue(dataSheet, "E4", l["subtitle"]); err != nil {
		return err
	}

	// Set row heights for the header block
	for row := 1; row <= 5; row++ {
		if err := f.SetRowHeight(dataSheet, row, 22); err != nil {
			return err
		}
	}

	// Embed logo
	if len(assets.LogoEPRAC) > 0 {
		_ = f.AddPictureFromBytes(dataSheet, "A1", &excelize.Picture{
			Extension: ".png",
			File:      assets.LogoEPRAC,
			Format: &excelize.GraphicOptions{
				OffsetX:       5,
				OffsetY:       5,
				ScaleX:        0.45,
				ScaleY:        0.45,
				Positioning:   "oneCell",
			},
		})
	}

	return nil
}

// applyArticleTemplateColumnHeaders writes styled column header row (row 6).
func applyArticleTemplateColumnHeaders(f *excelize.File, dataSheet string, language string) error {
	l := getLang(language)

	colStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"D6E4F0"}, Pattern: 1},
		Font: &excelize.Font{Bold: true, Size: 11, Color: "1E3A5F", Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "1E3A5F", Style: 2},
		},
	})
	if err != nil {
		return err
	}

	headers := []string{
		l["col_sku"], l["col_name"], l["col_desc"], l["col_price"], l["col_pres"],
		l["col_lot"], l["col_serial"], l["col_exp"],
		l["col_max"], l["col_min"], l["col_rotation"],
	}

	if err := f.SetRowHeight(dataSheet, 6, 18); err != nil {
		return err
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 6)
		if err := f.SetCellValue(dataSheet, cell, h); err != nil {
			return err
		}
		if err := f.SetCellStyle(dataSheet, cell, cell, colStyle); err != nil {
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
