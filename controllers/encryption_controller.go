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
		tools.Response(ctx, "EncryptData", false, response.Message, "encrypt", nil, false, "")
		return
	}

	tools.Response(ctx, "EncryptData", true, "Data encrypted successfully", "encrypt", gin.H{"encrypted_data": encryptedData}, false, "")
}

func (c *EncryptionController) DecryptData(ctx *gin.Context) {
	data := ctx.Param("data")

	decryptedData, response := c.Service.DecryptData(data)

	if response != nil {
		tools.Response(ctx, "DecryptData", false, response.Message, "decrypt", nil, false, "")
		return
	}

	tools.Response(ctx, "DecryptData", true, "Data decrypted successfully", "decrypt", gin.H{"decrypted_data": decryptedData}, false, "")
}
