package controllers

import (
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

type EncryptionController struct {
	Service services.EncryptionService
}

func NewEncryptionController(service services.EncryptionService) *EncryptionController {
	return &EncryptionController{
		Service: service,
	}
}

func (c *EncryptionController) EncryptData(ctx *gin.Context) {
	data := ctx.Param("data")

	encryptedData, response := c.Service.EncryptData(data)

	if response != nil {
		writeErrorResponse(ctx, "EncryptData", "encrypt", response)
		return
	}

	tools.ResponseOK(ctx, "EncryptData", "Data encrypted successfully", "encrypt", encryptedData, false, "")
}

func (c *EncryptionController) DecryptData(ctx *gin.Context) {
	data := ctx.Param("data")

	decryptedData, response := c.Service.DecryptData(data)

	if response != nil {
		writeErrorResponse(ctx, "DecryptData", "decrypt", response)
		return
	}

	tools.ResponseOK(ctx, "DecryptData", "Data decrypted successfully", "decrypt", decryptedData, false, "")
}
