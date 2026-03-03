package tools

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ParseRequiredParam reads the path parameter named paramName and returns (value, true) if non-empty.
// If the param is missing or empty, it writes ResponseBadRequest with invalidMessage and returns ("", false).
// Use for string IDs (e.g. location id or location_code). Caller should return immediately when ok is false.
func ParseRequiredParam(c *gin.Context, paramName, transactionType, endpointCode, invalidMessage string) (string, bool) {
	raw := strings.TrimSpace(c.Param(paramName))
	if raw == "" {
		ResponseBadRequest(c, transactionType, invalidMessage, endpointCode)
		return "", false
	}
	return raw, true
}

// ParseIntParam reads the path parameter named paramName, parses it as an int, and returns (value, true).
// If the param is missing or not a valid integer, it writes ResponseBadRequest with invalidMessage and returns (0, false).
// Use for :id, :idParam, etc. Caller should return immediately when ok is false.
func ParseIntParam(c *gin.Context, paramName, transactionType, endpointCode, invalidMessage string) (int, bool) {
	raw := c.Param(paramName)
	if raw == "" {
		ResponseBadRequest(c, transactionType, invalidMessage, endpointCode)
		return 0, false
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		ResponseBadRequest(c, transactionType, invalidMessage, endpointCode)
		return 0, false
	}
	return val, true
}

// ParseQueryInt reads the query parameter named name, parses it as an int, and validates it is within [min, max].
// If the param is missing, returns defaultVal. If invalid or out of range, writes ResponseBadRequest and returns (0, false).
// Caller should return immediately when ok is false.
func ParseQueryInt(c *gin.Context, name string, defaultVal, min, max int, transactionType, endpointCode, invalidMessage string) (int, bool) {
	raw := c.DefaultQuery(name, strconv.Itoa(defaultVal))
	val, err := strconv.Atoi(raw)
	if err != nil {
		ResponseBadRequest(c, transactionType, invalidMessage, endpointCode)
		return 0, false
	}
	if val < min || val > max {
		ResponseBadRequest(c, transactionType, invalidMessage, endpointCode)
		return 0, false
	}
	return val, true
}

// ParseQueryPageLimit parses page and limit query params with validation.
// Returns (page, limit, true) or (0, 0, false) and writes 400 on failure.
// Defaults: page=1, limit=20. Max limit=100. page and limit must be >= 1.
func ParseQueryPageLimit(c *gin.Context, transactionType, endpointCode string) (page, limit int, ok bool) {
	page, ok = ParseQueryInt(c, "page", 1, 1, 10000, transactionType, endpointCode, "page debe ser un número entre 1 y 10000")
	if !ok {
		return 0, 0, false
	}
	limit, ok = ParseQueryInt(c, "limit", 20, 1, 100, transactionType, endpointCode, "limit debe ser un número entre 1 y 100")
	if !ok {
		return 0, 0, false
	}
	return page, limit, true
}

// ParseQueryEnum validates that the query param value is one of the allowed values.
// If missing, returns defaultVal (or "" if not in allowed). If invalid, writes 400 and returns ("", false).
func ParseQueryEnum(c *gin.Context, name string, defaultVal string, allowed []string, transactionType, endpointCode string) (string, bool) {
	raw := c.DefaultQuery(name, defaultVal)
	for _, a := range allowed {
		if raw == a {
			return raw, true
		}
	}
	if len(allowed) > 0 {
		ResponseBadRequest(c, transactionType, "parámetro "+name+" inválido; valores permitidos: "+strings.Join(allowed, ", "), endpointCode)
	} else {
		ResponseBadRequest(c, transactionType, "parámetro "+name+" inválido", endpointCode)
	}
	return "", false
}
