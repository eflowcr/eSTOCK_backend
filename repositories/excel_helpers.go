package repositories

import "strings"

func get(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func ptr[T any](v T) *T { return &v }

func safeDeref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
