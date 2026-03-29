package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock encryption repo ─────────────────────────────────────────────────────

type mockEncryptionRepo struct {
	encryptResult string
	encryptErr    *responses.InternalResponse
	decryptResult string
	decryptErr    *responses.InternalResponse
}

func (m *mockEncryptionRepo) Encrypt(data string) (string, *responses.InternalResponse) {
	return m.encryptResult, m.encryptErr
}
func (m *mockEncryptionRepo) Decrypt(data string) (string, *responses.InternalResponse) {
	return m.decryptResult, m.decryptErr
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestEncryptionController_EncryptData_Success(t *testing.T) {
	repo := &mockEncryptionRepo{encryptResult: "encrypted-base64"}
	ctrl := NewEncryptionController(*services.NewEncryptionService(repo))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/encrypt/hello", nil)
	c.Request = req
	c.Params = gin.Params{{Key: "data", Value: "hello"}}
	ctrl.EncryptData(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEncryptionController_EncryptData_Error(t *testing.T) {
	repo := &mockEncryptionRepo{
		encryptErr: &responses.InternalResponse{Message: "encrypt failed", StatusCode: responses.StatusInternalServerError},
	}
	ctrl := NewEncryptionController(*services.NewEncryptionService(repo))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/encrypt/data", nil)
	c.Request = req
	c.Params = gin.Params{{Key: "data", Value: "data"}}
	ctrl.EncryptData(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestEncryptionController_DecryptData_Success(t *testing.T) {
	repo := &mockEncryptionRepo{decryptResult: "plaintext"}
	ctrl := NewEncryptionController(*services.NewEncryptionService(repo))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/decrypt/enc", nil)
	c.Request = req
	c.Params = gin.Params{{Key: "data", Value: "enc"}}
	ctrl.DecryptData(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEncryptionController_DecryptData_Error(t *testing.T) {
	repo := &mockEncryptionRepo{
		decryptErr: &responses.InternalResponse{Message: "decrypt failed", StatusCode: responses.StatusBadRequest},
	}
	ctrl := NewEncryptionController(*services.NewEncryptionService(repo))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/decrypt/bad", nil)
	c.Request = req
	c.Params = gin.Params{{Key: "data", Value: "bad"}}
	ctrl.DecryptData(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
