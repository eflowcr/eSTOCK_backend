package tools

import (
	"net/http"

	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/gin-gonic/gin"
)

// Response sends a JSON response using the shared envelope. It uses 200 for success or "handled"
// errors, and 400 for unhandled errors. Prefer the status helpers below for explicit status codes.
func Response(c *gin.Context, transactionType string, success bool, message string, endpointCode string, data interface{}, encrypted bool, encryptionType string, handled bool) {
	response := responses.APIResponse{
		Envelope: responses.Envelope{
			TransactionType: transactionType,
			Encrypted:       encrypted,
			EncryptionType:  encryptionType,
		},
		Result: responses.Result{
			Success:      success,
			Message:      message,
			EndpointCode: endpointCode,
		},
		Data: data,
	}

	status := http.StatusOK
	if !success {
		if !handled {
			status = http.StatusBadRequest
		}
	}

	c.JSON(status, response)
}

// writeResponse builds the envelope and writes the given status. success is true for 2xx, false otherwise.
func writeResponse(c *gin.Context, status int, transactionType, message, endpointCode string, data interface{}, encrypted bool, encryptionType string, success bool) {
	resp := responses.APIResponse{
		Envelope: responses.Envelope{
			TransactionType: transactionType,
			Encrypted:       encrypted,
			EncryptionType:  encryptionType,
		},
		Result: responses.Result{
			Success:      success,
			Message:      message,
			EndpointCode: endpointCode,
		},
		Data: data,
	}
	c.JSON(status, resp)
}

// Status helpers — use these for explicit HTTP status codes. Envelope shape unchanged for frontend compatibility.
// 200 OK (e.g. GET success, update success)
func ResponseOK(c *gin.Context, transactionType, message, endpointCode string, data interface{}, encrypted bool, encryptionType string) {
	writeResponse(c, http.StatusOK, transactionType, message, endpointCode, data, encrypted, encryptionType, true)
}

// 201 Created (POST create success)
func ResponseCreated(c *gin.Context, transactionType, message, endpointCode string, data interface{}, encrypted bool, encryptionType string) {
	writeResponse(c, http.StatusCreated, transactionType, message, endpointCode, data, encrypted, encryptionType, true)
}

// 400 Bad Request (validation, invalid input)
func ResponseBadRequest(c *gin.Context, transactionType, message, endpointCode string) {
	writeResponse(c, http.StatusBadRequest, transactionType, message, endpointCode, nil, false, "", false)
}

// 401 Unauthorized (missing or invalid auth)
func ResponseUnauthorized(c *gin.Context, transactionType, message, endpointCode string) {
	writeResponse(c, http.StatusUnauthorized, transactionType, message, endpointCode, nil, false, "", false)
}

// 403 Forbidden (auth OK but not allowed)
func ResponseForbidden(c *gin.Context, transactionType, message, endpointCode string) {
	writeResponse(c, http.StatusForbidden, transactionType, message, endpointCode, nil, false, "", false)
}

// 404 Not Found (resource does not exist)
func ResponseNotFound(c *gin.Context, transactionType, message, endpointCode string) {
	writeResponse(c, http.StatusNotFound, transactionType, message, endpointCode, nil, false, "", false)
}

// 409 Conflict (e.g. duplicate SKU, unique constraint violation)
func ResponseConflict(c *gin.Context, transactionType, message, endpointCode string) {
	writeResponse(c, http.StatusConflict, transactionType, message, endpointCode, nil, false, "", false)
}

// 500 Internal Server Error (unexpected server error)
func ResponseInternal(c *gin.Context, transactionType, message, endpointCode string) {
	writeResponse(c, http.StatusInternalServerError, transactionType, message, endpointCode, nil, false, "", false)
}
