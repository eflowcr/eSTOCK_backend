package repositories

import (
	"testing"
)

func TestIsValidReceivingTransition(t *testing.T) {
	tests := []struct {
		name    string
		current string
		next    string
		want    bool
	}{
		// ── Transiciones válidas desde open ───────────────────────────────────
		{"open → in_progress", "open", "in_progress", true},
		{"open → cancelled", "open", "cancelled", true},

		// ── Transiciones válidas desde in_progress ────────────────────────────
		{"in_progress → completed", "in_progress", "completed", true},
		{"in_progress → completed_with_differences", "in_progress", "completed_with_differences", true},
		{"in_progress → cancelled", "in_progress", "cancelled", true},

		// ── No-op: mismo → mismo (siempre true) ──────────────────────────────
		{"no-op open", "open", "open", true},
		{"no-op in_progress", "in_progress", "in_progress", true},
		{"no-op completed", "completed", "completed", true},
		{"no-op completed_with_differences", "completed_with_differences", "completed_with_differences", true},
		{"no-op cancelled", "cancelled", "cancelled", true},

		// ── Estados finales: sin transición saliente ──────────────────────────
		{"completed → open (final)", "completed", "open", false},
		{"completed → in_progress (final)", "completed", "in_progress", false},
		{"completed_with_differences → open (final)", "completed_with_differences", "open", false},
		{"cancelled → open (final)", "cancelled", "open", false},
		{"cancelled → in_progress (final)", "cancelled", "in_progress", false},

		// ── Transiciones inválidas entre estados no-finales ───────────────────
		{"open → completed (salta in_progress)", "open", "completed", false},
		{"open → completed_with_differences (salta)", "open", "completed_with_differences", false},
		{"in_progress → open (no retroactivo)", "in_progress", "open", false},

		// ── Receiving no tiene abandoned ──────────────────────────────────────
		{"open → abandoned (no existe en receiving)", "open", "abandoned", false},
		{"in_progress → abandoned (no existe en receiving)", "in_progress", "abandoned", false},

		// ── Estado desconocido como origen ────────────────────────────────────
		{"unknown origin", "foo", "open", false},
		{"unknown origin and next", "foo", "bar", false},
		{"empty strings", "", "", true}, // no-op: "" == "" → true
		{"empty origin, non-empty next", "", "open", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidReceivingTransition(tt.current, tt.next)
			if got != tt.want {
				t.Errorf("isValidReceivingTransition(%q, %q) = %v, want %v", tt.current, tt.next, got, tt.want)
			}
		})
	}
}
