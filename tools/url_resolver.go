package tools

// devFallbackURL is the last-resort base URL used when neither an allowed
// Origin header nor Config.AppURL is available. Keeps local dev working without
// any env configuration.
const devFallbackURL = "http://localhost:4200"

// ResolveFrontendURL returns the best frontend base URL to embed in outbound emails.
//
// Priority:
//  1. Request Origin header, if present AND in the ALLOWED_ORIGINS allowlist.
//  2. fallbackURL (typically Config.AppURL).
//  3. "http://localhost:4200" as last-resort dev default.
//
// The ALLOWED_ORIGINS allowlist check prevents an attacker from sending a
// forgot-password request with a spoofed Origin header to redirect the email
// link to a malicious site.
func ResolveFrontendURL(origin string, fallbackURL string) string {
	if IsAllowedOrigin(origin) {
		return origin
	}
	if fallbackURL != "" {
		return fallbackURL
	}
	return devFallbackURL
}
