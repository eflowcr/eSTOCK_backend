package repositories

import (
	_ "image/png"

	"github.com/eflowcr/eSTOCK_backend/assets"
	"github.com/xuri/excelize/v2"
)

// ColumnDef describes a single import template column.
type ColumnDef struct {
	Header   string
	Required bool
	Width    float64
}

// ModuleTemplateConfig holds everything needed to build a module's import template.
type ModuleTemplateConfig struct {
	DataSheetName string
	OptSheetName  string
	Title         string
	Subtitle      string
	InstrTitle    string
	InstrContent  string
	Columns       []ColumnDef
	ExampleRow    []string
	// ApplyValidations adds module-specific dropdown / numeric validations.
	// dataStartRow is the first data row (9 in the current layout).
	ApplyValidations func(f *excelize.File, dataSheet, optSheet string, dataStartRow, dataEndRow int) error
}

const (
	sharedDataStartRow = 9
	sharedDataEndRow   = 2000
)

// BuildModuleImportTemplate creates a full-featured import Excel for any module.
func BuildModuleImportTemplate(cfg ModuleTemplateConfig) ([]byte, error) {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", cfg.DataSheetName)

	if err := applySharedLogoHeader(f, cfg.DataSheetName, cfg.Title, cfg.Subtitle); err != nil {
		return nil, err
	}
	if err := applySharedInstructions(f, cfg.DataSheetName, cfg.InstrTitle, cfg.InstrContent); err != nil {
		return nil, err
	}
	if err := applySharedColumnHeaders(f, cfg.DataSheetName, cfg.Columns); err != nil {
		return nil, err
	}
	if err := applySharedExampleRow(f, cfg.DataSheetName, cfg.ExampleRow); err != nil {
		return nil, err
	}
	if err := applySharedAlternatingColors(f, cfg.DataSheetName, len(cfg.Columns)); err != nil {
		return nil, err
	}
	if cfg.ApplyValidations != nil {
		if err := cfg.ApplyValidations(f, cfg.DataSheetName, cfg.OptSheetName, sharedDataStartRow, sharedDataEndRow); err != nil {
			return nil, err
		}
	}

	var buf excelize.File
	_ = buf
	data, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}

func applySharedLogoHeader(f *excelize.File, dataSheet, title, subtitle string) error {
	logoStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{epracAccent}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return err
	}
	titleStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{epracWhite}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Size: 22, Color: epracNavy, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return err
	}
	subStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{epracWhite}, Pattern: 1},
		Font:      &excelize.Font{Size: 11, Color: epracGray, Italic: true, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    []excelize.Border{{Type: "top", Color: epracGrayBorder, Style: 4}},
	})
	if err != nil {
		return err
	}

	// Determine column count from later set widths; use K (11) as max for the merges
	if err := f.MergeCell(dataSheet, "A1", "B4"); err != nil {
		return err
	}
	if err := f.MergeCell(dataSheet, "C1", "K2"); err != nil {
		return err
	}
	if err := f.MergeCell(dataSheet, "C3", "K4"); err != nil {
		return err
	}

	// Fill A1:B4 with accent
	for row := 1; row <= 4; row++ {
		for col := 1; col <= 2; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			if err := f.SetCellStyle(dataSheet, cell, cell, logoStyle); err != nil {
				return err
			}
		}
	}
	if err := f.SetCellStyle(dataSheet, "C1", "K2", titleStyle); err != nil {
		return err
	}
	if err := f.SetCellStyle(dataSheet, "C3", "K4", subStyle); err != nil {
		return err
	}
	if err := f.SetCellValue(dataSheet, "C1", title); err != nil {
		return err
	}
	if err := f.SetCellValue(dataSheet, "C3", subtitle); err != nil {
		return err
	}

	heights := map[int]float64{1: 26, 2: 26, 3: 16, 4: 16}
	for row, h := range heights {
		if err := f.SetRowHeight(dataSheet, row, h); err != nil {
			return err
		}
	}

	if len(assets.LogoEPRAC) > 0 {
		_ = f.AddPictureFromBytes(dataSheet, "A1", &excelize.Picture{
			Extension:  ".png",
			File:       assets.LogoEPRAC,
			InsertType: excelize.PictureInsertTypePlaceOverCells,
			Format: &excelize.GraphicOptions{
				OffsetX: 8, OffsetY: 6,
				ScaleX: 0.247, ScaleY: 0.236,
			},
		})
	}
	return nil
}

func applySharedInstructions(f *excelize.File, dataSheet, instrTitle, instrContent string) error {
	titleStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{epracBlue}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Size: 10, Color: epracWhite, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return err
	}
	contentStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{epracAccent}, Pattern: 1},
		Font:      &excelize.Font{Size: 9, Color: epracNavy, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "top", WrapText: true},
	})
	if err != nil {
		return err
	}

	if err := f.MergeCell(dataSheet, "A5", "K5"); err != nil {
		return err
	}
	if err := f.MergeCell(dataSheet, "A6", "K6"); err != nil {
		return err
	}
	if err := f.SetCellStyle(dataSheet, "A5", "K5", titleStyle); err != nil {
		return err
	}
	if err := f.SetCellStyle(dataSheet, "A6", "K6", contentStyle); err != nil {
		return err
	}
	if err := f.SetCellValue(dataSheet, "A5", instrTitle); err != nil {
		return err
	}
	if err := f.SetCellValue(dataSheet, "A6", instrContent); err != nil {
		return err
	}
	if err := f.SetRowHeight(dataSheet, 5, 20); err != nil {
		return err
	}
	return f.SetRowHeight(dataSheet, 6, 32)
}

func applySharedColumnHeaders(f *excelize.File, dataSheet string, cols []ColumnDef) error {
	requiredStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{epracBlueDark}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Size: 11, Color: epracWhite, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return err
	}
	headerStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{epracBlue}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Size: 11, Color: epracWhite, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return err
	}

	colNames := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N"}
	for i, col := range cols {
		if i >= len(colNames) {
			break
		}
		if err := f.SetColWidth(dataSheet, colNames[i], colNames[i], col.Width); err != nil {
			return err
		}
		cell, _ := excelize.CoordinatesToCellName(i+1, 7)
		if err := f.SetCellValue(dataSheet, cell, col.Header); err != nil {
			return err
		}
		style := headerStyle
		if col.Required {
			style = requiredStyle
		}
		if err := f.SetCellStyle(dataSheet, cell, cell, style); err != nil {
			return err
		}
	}
	return f.SetRowHeight(dataSheet, 7, 36)
}

func applySharedExampleRow(f *excelize.File, dataSheet string, example []string) error {
	exampleStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{epracExample}, Pattern: 1},
		Font:      &excelize.Font{Size: 10, Color: epracGray, Italic: true, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return err
	}
	for i, val := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 8)
		if err := f.SetCellValue(dataSheet, cell, val); err != nil {
			return err
		}
		if err := f.SetCellStyle(dataSheet, cell, cell, exampleStyle); err != nil {
			return err
		}
	}
	return f.SetRowHeight(dataSheet, 8, 22)
}

func applySharedAlternatingColors(f *excelize.File, dataSheet string, numCols int) error {
	grayStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{epracGrayLight}, Pattern: 1},
	})
	if err != nil {
		return err
	}
	colNames := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N"}
	if numCols > len(colNames) {
		numCols = len(colNames)
	}
	for row := sharedDataStartRow; row <= sharedDataStartRow+49; row++ {
		if row%2 != 0 {
			for col := 1; col <= numCols; col++ {
				cell, _ := excelize.CoordinatesToCellName(col, row)
				if err := f.SetCellStyle(dataSheet, cell, cell, grayStyle); err != nil {
					return err
				}
			}
		}
		if err := f.SetRowHeight(dataSheet, row, 20); err != nil {
			return err
		}
	}
	return nil
}

// SharedDropListValidation adds a dropdown list validation referencing the options sheet.
func SharedDropListValidation(f *excelize.File, dataSheet, optSheet, colRange, optRange, errTitle, errMsg string) error {
	ref := "'" + optSheet + "'!" + optRange
	return addDropListValidation(f, dataSheet, colRange, ref, errTitle, errMsg)
}

// SharedNumericValidation adds a numeric >= 0 validation.
func SharedNumericValidation(f *excelize.File, dataSheet, colRange string, vType excelize.DataValidationType, errTitle, errMsg string) error {
	return addNumericMinValidation(f, dataSheet, colRange, vType, errTitle, errMsg)
}
