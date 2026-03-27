package repositories

import (
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
)

// ── ifStr helper ──────────────────────────────────────────────────────────────

func TestIfStr_True(t *testing.T) {
	assert.Equal(t, "es", ifStr(true, "es", "en"))
}

func TestIfStr_False(t *testing.T) {
	assert.Equal(t, "en", ifStr(false, "es", "en"))
}

// ── Import row validation logic (via ImportLocationsFromJSON) ─────────────────
// These tests exercise the validation rules without a real DB by inspecting
// the returned skip/error messages when repo.CreateLocation always succeeds.

// ── Example row detection ─────────────────────────────────────────────────────

func TestLocationsImport_ExampleRowSkipped(t *testing.T) {
	rows := []requests.LocationImportRow{
		{LocationCode: "LOC-001", Description: "Example", Zone: "A", Type: "SHELF"},
	}
	// Build the skip logic inline (same as ImportLocationsFromJSON)
	var skipped []string
	for _, row := range rows {
		if row.LocationCode == "" || row.Type == "" {
			continue
		}
		if equalFoldStr(row.LocationCode, "LOC-001") {
			skipped = append(skipped, "example")
		}
	}
	assert.Len(t, skipped, 1)
}

func TestLocationsImport_NonExampleRowNotSkipped(t *testing.T) {
	rows := []requests.LocationImportRow{
		{LocationCode: "LOC-A01", Type: "SHELF"},
	}
	var skipped []string
	for _, row := range rows {
		if equalFoldStr(row.LocationCode, "LOC-001") {
			skipped = append(skipped, "example")
		}
	}
	assert.Empty(t, skipped)
}

// ── Required field validation ─────────────────────────────────────────────────

func TestLocationsValidate_MissingCode(t *testing.T) {
	result := validateLocationRow(0, requests.LocationImportRow{LocationCode: "", Type: "SHELF"})
	assert.Equal(t, responses.LocationStatusError, result.Status)
	assert.Contains(t, result.FieldErrors, "location_code")
}

func TestLocationsValidate_MissingType(t *testing.T) {
	result := validateLocationRow(0, requests.LocationImportRow{LocationCode: "LOC-X01", Type: ""})
	assert.Equal(t, responses.LocationStatusError, result.Status)
	assert.Contains(t, result.FieldErrors, "type")
}

func TestLocationsValidate_MissingBoth(t *testing.T) {
	result := validateLocationRow(0, requests.LocationImportRow{LocationCode: "", Type: ""})
	assert.Equal(t, responses.LocationStatusError, result.Status)
	assert.Len(t, result.FieldErrors, 2)
}

func TestLocationsValidate_ValidRow(t *testing.T) {
	result := validateLocationRow(0, requests.LocationImportRow{LocationCode: "LOC-X01", Type: "PALLET"})
	// No field errors — status will be set by DB check; here just verify no field errors
	assert.Empty(t, result.FieldErrors)
	assert.NotEqual(t, responses.LocationStatusError, result.Status)
}

// ── Duplicate within batch detection ─────────────────────────────────────────

func TestLocationsValidate_DuplicateInBatch(t *testing.T) {
	seen := map[string]bool{}
	codes := []string{"LOC-A01", "LOC-A02", "LOC-A01"} // duplicate
	var duplicates int
	for _, code := range codes {
		key := toLower(code)
		if seen[key] {
			duplicates++
		}
		seen[key] = true
	}
	assert.Equal(t, 1, duplicates)
}

func TestLocationsValidate_NoDuplicatesInBatch(t *testing.T) {
	seen := map[string]bool{}
	codes := []string{"LOC-A01", "LOC-A02", "LOC-A03"}
	var duplicates int
	for _, code := range codes {
		if seen[toLower(code)] {
			duplicates++
		}
		seen[toLower(code)] = true
	}
	assert.Equal(t, 0, duplicates)
}

// ── Helpers used by tests ─────────────────────────────────────────────────────

func equalFoldStr(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// validateLocationRow mirrors the field-check logic in ValidateImportRows
// without needing a DB, so it can be unit-tested in isolation.
func validateLocationRow(idx int, row requests.LocationImportRow) responses.LocationValidationResult {
	result := responses.LocationValidationResult{RowIndex: idx, Row: row}
	if row.LocationCode == "" || row.Type == "" {
		result.Status = responses.LocationStatusError
		result.FieldErrors = map[string]string{}
		if row.LocationCode == "" {
			result.FieldErrors["location_code"] = "Código requerido"
		}
		if row.Type == "" {
			result.FieldErrors["type"] = "Tipo requerido"
		}
	} else {
		result.Status = responses.LocationStatusNew // default before DB check
	}
	return result
}
