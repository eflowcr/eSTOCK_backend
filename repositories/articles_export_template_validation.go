package repositories

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	articleTemplateDataStartRow = 7
	articleTemplateDataEndRow   = 2000
)

func applyArticleTemplateValidations(f *excelize.File, dataSheet string, presentationOptions []string) error {
	options := uniqueNonEmptyStrings(presentationOptions)
	if len(options) == 0 {
		options = []string{"unidad"}
	}

	// Keep helper lists visible in the same sheet so users can confirm
	// which values are valid for dropdowns.
	if err := f.SetCellValue(dataSheet, "L1", "Presentaciones"); err != nil {
		return err
	}
	for i, opt := range options {
		cell, _ := excelize.CoordinatesToCellName(12, i+2)
		if err := f.SetCellValue(dataSheet, cell, opt); err != nil {
			return err
		}
	}

	if err := f.SetCellValue(dataSheet, "M1", "Opciones"); err != nil {
		return err
	}
	yesNo := []string{"Si", "No"}
	for i, opt := range yesNo {
		cell, _ := excelize.CoordinatesToCellName(13, i+2)
		if err := f.SetCellValue(dataSheet, cell, opt); err != nil {
			return err
		}
	}

	if err := f.SetCellValue(dataSheet, "N1", "Rotacion"); err != nil {
		return err
	}
	rotation := []string{"fifo", "fefo"}
	for i, opt := range rotation {
		cell, _ := excelize.CoordinatesToCellName(14, i+2)
		if err := f.SetCellValue(dataSheet, cell, opt); err != nil {
			return err
		}
	}

	if err := addDropListValidation(
		f,
		dataSheet,
		fmt.Sprintf("E%d:E%d", articleTemplateDataStartRow, articleTemplateDataEndRow),
		fmt.Sprintf("$L$2:$L$%d", len(options)+1),
		"Presentacion invalida",
		"Seleccione una presentacion valida de la lista",
	); err != nil {
		return err
	}

	if err := addDropListValidation(
		f,
		dataSheet,
		fmt.Sprintf("F%d:H%d", articleTemplateDataStartRow, articleTemplateDataEndRow),
		"$M$2:$M$3",
		"Valor invalido",
		"Use solo Si o No",
	); err != nil {
		return err
	}

	if err := addDropListValidation(
		f,
		dataSheet,
		fmt.Sprintf("K%d:K%d", articleTemplateDataStartRow, articleTemplateDataEndRow),
		"$N$2:$N$3",
		"Rotacion invalida",
		"Use fifo o fefo",
	); err != nil {
		return err
	}

	if err := addNumericMinValidation(
		f,
		dataSheet,
		fmt.Sprintf("D%d:D%d", articleTemplateDataStartRow, articleTemplateDataEndRow),
		excelize.DataValidationTypeDecimal,
		"Precio invalido",
		"Ingrese un precio numerico mayor o igual a 0",
	); err != nil {
		return err
	}

	if err := addNumericMinValidation(
		f,
		dataSheet,
		fmt.Sprintf("I%d:J%d", articleTemplateDataStartRow, articleTemplateDataEndRow),
		excelize.DataValidationTypeWhole,
		"Cantidad invalida",
		"Ingrese un numero entero mayor o igual a 0",
	); err != nil {
		return err
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
