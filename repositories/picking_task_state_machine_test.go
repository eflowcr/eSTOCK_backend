package repositories

import (
	"testing"
)

func TestIsValidPickingTransition(t *testing.T) {
	tests := []struct {
		name    string
		current string
		next    string
		want    bool
	}{
		// ── Transiciones válidas desde open ───────────────────────────────────
		{"open → assigned", "open", "assigned", true},
		{"open → in_progress", "open", "in_progress", true},
		{"open → cancelled", "open", "cancelled", true},
		{"open → abandoned", "open", "abandoned", true},

		// ── Transiciones válidas desde assigned ───────────────────────────────
		{"assigned → open (des-asignar)", "assigned", "open", true},
		{"assigned → in_progress", "assigned", "in_progress", true},
		{"assigned → cancelled", "assigned", "cancelled", true},
		{"assigned → abandoned", "assigned", "abandoned", true},

		// ── Transiciones válidas desde in_progress ────────────────────────────
		{"in_progress → completed", "in_progress", "completed", true},
		{"in_progress → completed_with_differences", "in_progress", "completed_with_differences", true},
		{"in_progress → cancelled", "in_progress", "cancelled", true},
		{"in_progress → abandoned", "in_progress", "abandoned", true},

		// ── No-op: mismo → mismo (siempre true) ──────────────────────────────
		{"no-op open", "open", "open", true},
		{"no-op assigned", "assigned", "assigned", true},
		{"no-op in_progress", "in_progress", "in_progress", true},
		{"no-op completed", "completed", "completed", true},
		{"no-op completed_with_differences", "completed_with_differences", "completed_with_differences", true},
		{"no-op cancelled", "cancelled", "cancelled", true},
		{"no-op abandoned", "abandoned", "abandoned", true},

		// ── Estados finales: sin transición saliente ──────────────────────────
		{"completed → open (final)", "completed", "open", false},
		{"completed → in_progress (final)", "completed", "in_progress", false},
		{"completed_with_differences → open (final)", "completed_with_differences", "open", false},
		{"cancelled → in_progress (final)", "cancelled", "in_progress", false},
		{"cancelled → open (final)", "cancelled", "open", false},
		{"abandoned → open (final)", "abandoned", "open", false},
		{"abandoned → in_progress (final)", "abandoned", "in_progress", false},

		// ── Transiciones inválidas entre estados no-finales ───────────────────
		{"open → completed (salta in_progress)", "open", "completed", false},
		{"open → completed_with_differences (salta)", "open", "completed_with_differences", false},
		{"in_progress → open (no retroactivo)", "in_progress", "open", false},
		{"in_progress → assigned (no retroactivo)", "in_progress", "assigned", false},
		{"assigned → completed (salta in_progress)", "assigned", "completed", false},

		// ── Estado desconocido como origen ────────────────────────────────────
		{"unknown origin", "foo", "open", false},
		{"unknown origin and next", "foo", "bar", false},
		{"empty strings", "", "", true}, // no-op: "" == "" → true
		{"empty origin, non-empty next", "", "open", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidPickingTransition(tt.current, tt.next)
			if got != tt.want {
				t.Errorf("isValidPickingTransition(%q, %q) = %v, want %v", tt.current, tt.next, got, tt.want)
			}
		})
	}
}
