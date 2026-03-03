// Package tools provides structured logging for service and repository errors.
// Log only domain, operation, and error context; no PII (no user IDs, emails, or request bodies).

package tools

import (
	"github.com/rs/zerolog/log"
)

// LogRepoError logs a repository-layer error with structured fields.
// domain: e.g. "articles", "lots", "locations"
// operation: e.g. "GetByID", "Create", "Update"
// Do not pass user identifiers or request payloads.
func LogRepoError(domain, operation string, err error, message string) {
	if err == nil {
		return
	}
	log.Error().
		Str("layer", "repo").
		Str("domain", domain).
		Str("operation", operation).
		Err(err).
		Msg(message)
}

// LogServiceError logs a service-layer error with structured fields.
// domain: e.g. "articles", "lots"
// operation: e.g. "CreateArticle", "GetAllLots"
// Do not pass user identifiers or request payloads.
func LogServiceError(domain, operation string, err error, message string) {
	if err == nil {
		return
	}
	log.Error().
		Str("layer", "service").
		Str("domain", domain).
		Str("operation", operation).
		Err(err).
		Msg(message)
}
