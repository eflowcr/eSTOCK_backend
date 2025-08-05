package tools

import (
	"net/http"

	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/gin-gonic/gin"
)

func Response(c *gin.Context, transactionType string, success bool, message string, endpointCode string, data interface{}, encrypted bool, encryptionType string) {
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
		status = http.StatusBadRequest
	}

	c.JSON(status, response)
}
