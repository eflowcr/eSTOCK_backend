package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── parseBoolCell ─────────────────────────────────────────────────────────────

func TestParseBoolCell_Spanish(t *testing.T) {
	assert.True(t, parseBoolCell("Si"))
	assert.True(t, parseBoolCell("si"))
	assert.True(t, parseBoolCell("SI"))
	assert.True(t, parseBoolCell("Sí"))
	assert.True(t, parseBoolCell("sí"))
}

func TestParseBoolCell_English(t *testing.T) {
	assert.True(t, parseBoolCell("Yes"))
	assert.True(t, parseBoolCell("yes"))
	assert.True(t, parseBoolCell("YES"))
	assert.True(t, parseBoolCell("true"))
	assert.True(t, parseBoolCell("True"))
	assert.True(t, parseBoolCell("TRUE"))
	assert.True(t, parseBoolCell("1"))
}

func TestParseBoolCell_Falsy(t *testing.T) {
	assert.False(t, parseBoolCell("No"))
	assert.False(t, parseBoolCell("no"))
	assert.False(t, parseBoolCell("false"))
	assert.False(t, parseBoolCell("0"))
	assert.False(t, parseBoolCell(""))
	assert.False(t, parseBoolCell("nope"))
}

// ── uniqueNonEmptyStrings ─────────────────────────────────────────────────────

func TestUniqueNonEmptyStrings_Basic(t *testing.T) {
	result := uniqueNonEmptyStrings([]string{"unit", "box", "unit", "", "BOX", "pallet"})
	// "unit", "box", "pallet" — case-insensitive dedup, preserves first occurrence
	assert.Len(t, result, 3)
	assert.Equal(t, "unit", result[0])
	assert.Equal(t, "box", result[1])
	assert.Equal(t, "pallet", result[2])
}

func TestUniqueNonEmptyStrings_Empty(t *testing.T) {
	assert.Empty(t, uniqueNonEmptyStrings(nil))
	assert.Empty(t, uniqueNonEmptyStrings([]string{}))
	assert.Empty(t, uniqueNonEmptyStrings([]string{"", "  "}))
}

// ── buildImportTemplate smoke test ────────────────────────────────────────────

func TestBuildImportTemplate_ReturnsBytes(t *testing.T) {
	data, errResp := buildImportTemplate([]string{"unit", "box"}, "es")
	assert.Nil(t, errResp)
	assert.NotEmpty(t, data)
}

func TestBuildImportTemplate_English(t *testing.T) {
	data, errResp := buildImportTemplate([]string{"unit"}, "en")
	assert.Nil(t, errResp)
	assert.NotEmpty(t, data)
}

func TestBuildImportTemplate_FallbackLanguage(t *testing.T) {
	// "fr" falls back to "es"
	data, errResp := buildImportTemplate([]string{"unit"}, "fr")
	assert.Nil(t, errResp)
	assert.NotEmpty(t, data)
}

// ── getLang fallback ──────────────────────────────────────────────────────────

func TestGetLang_KnownLanguages(t *testing.T) {
	es := getLang("es")
	assert.Equal(t, "Importar Artículos", es["title"])

	en := getLang("en")
	assert.Equal(t, "Import Articles", en["title"])
}

func TestGetLang_UnknownFallsBackToSpanish(t *testing.T) {
	de := getLang("de")
	assert.Equal(t, "Importar Artículos", de["title"])
}
