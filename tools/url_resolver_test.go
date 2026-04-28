package tools

import (
	"testing"
)

// Each test case resets the allowlist singleton so t.Setenv changes take effect.

func TestResolveFrontendURL_OriginInAllowlist_ReturnsOrigin(t *testing.T) {
	resetAllowedOriginsForTest()
	t.Setenv("ALLOWED_ORIGINS", "https://estock.eflowsuite.com,http://localhost:4200")

	got := ResolveFrontendURL("https://estock.eflowsuite.com", "http://fallback.example.com")
	if got != "https://estock.eflowsuite.com" {
		t.Fatalf("expected origin to be returned when in allowlist, got %q", got)
	}
}

func TestResolveFrontendURL_OriginNotInAllowlist_ReturnsFallback(t *testing.T) {
	resetAllowedOriginsForTest()
	t.Setenv("ALLOWED_ORIGINS", "https://estock.eflowsuite.com")

	got := ResolveFrontendURL("https://attacker.example.com", "http://fallback.example.com")
	if got != "http://fallback.example.com" {
		t.Fatalf("expected fallback when origin not in allowlist, got %q", got)
	}
}

func TestResolveFrontendURL_EmptyOrigin_ReturnsFallback(t *testing.T) {
	resetAllowedOriginsForTest()
	t.Setenv("ALLOWED_ORIGINS", "https://estock.eflowsuite.com")

	got := ResolveFrontendURL("", "http://fallback.example.com")
	if got != "http://fallback.example.com" {
		t.Fatalf("expected fallback for empty origin, got %q", got)
	}
}

func TestResolveFrontendURL_EmptyOriginAndFallback_ReturnsLocalhost(t *testing.T) {
	resetAllowedOriginsForTest()
	t.Setenv("ALLOWED_ORIGINS", "https://estock.eflowsuite.com")

	got := ResolveFrontendURL("", "")
	if got != devFallbackURL {
		t.Fatalf("expected localhost dev fallback when origin and fallbackURL are empty, got %q", got)
	}
}

func TestResolveFrontendURL_EmptyAllowedOrigins_FallsBack(t *testing.T) {
	// When ALLOWED_ORIGINS is empty, the default allowlist is used (which includes
	// prod/qa/dev hostnames and localhost). A non-allowlisted origin must not pass.
	resetAllowedOriginsForTest()
	t.Setenv("ALLOWED_ORIGINS", "")

	// A random unknown origin must not be trusted even with empty env var.
	got := ResolveFrontendURL("https://unknown.example.com", "http://config.example.com")
	if got != "http://config.example.com" {
		t.Fatalf("expected fallback when origin not in default allowlist, got %q", got)
	}
}
