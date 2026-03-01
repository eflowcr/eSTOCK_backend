package tools

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var defaultValidator = validator.New()

// ValidationError describes a single field validation failure for the API response.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidateStruct runs validation on v and returns a slice of field errors suitable for the API.
// Returns nil if valid. Messages are sanitized (no stack traces or internal paths).
func ValidateStruct(v interface{}) []ValidationError {
	err := defaultValidator.Struct(v)
	if err == nil {
		return nil
	}
	verrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return []ValidationError{{Field: "_", Message: "invalid request"}}
	}
	var out []ValidationError
	for _, e := range verrs {
		msg := e.Tag()
		if e.Param() != "" {
			msg += ":" + e.Param()
		}
		out = append(out, ValidationError{
			Field:   e.Field(),
			Message: msg,
		})
	}
	return out
}

// ResponseValidationError sends 400 with result.message "Validation failed" and data.errors set to errs.
// Use after ValidateStruct when the slice is non-nil. Same envelope as other responses.
func ResponseValidationError(c *gin.Context, transactionType, endpointCode string, errs []ValidationError) {
	writeResponse(c, http.StatusBadRequest, transactionType, "Validation failed", endpointCode, gin.H{"errors": errs}, false, "", false)
}
