package tools_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayEmailSender_Send_201Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/emails/send", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"send":{"id":"123"}}`))
	}))
	defer srv.Close()
	sender := tools.NewGatewayEmailSender(srv.URL+"/api/v1", "test-key", "from@test.com", "TestApp")
	err := sender.Send(context.Background(), "to@test.com", "Subject", "<b>html</b>", "text")
	require.NoError(t, err)
}

func TestGatewayEmailSender_Send_202QueuedIsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"message":"queued"}`))
	}))
	defer srv.Close()
	sender := tools.NewGatewayEmailSender(srv.URL+"/api/v1", "key", "from@test.com", "App")
	err := sender.Send(context.Background(), "to@test.com", "Subject", "", "text only")
	require.NoError(t, err)
}

func TestGatewayEmailSender_Send_400ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"body_html or body_text is required"}`))
	}))
	defer srv.Close()
	sender := tools.NewGatewayEmailSender(srv.URL+"/api/v1", "key", "from@test.com", "App")
	err := sender.Send(context.Background(), "to@test.com", "Subject", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 400")
}

func TestGatewayEmailSender_Send_403Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()
	sender := tools.NewGatewayEmailSender(srv.URL+"/api/v1", "wrong-key", "from@test.com", "App")
	err := sender.Send(context.Background(), "to@test.com", "Subject", "html", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 403")
}

func TestGatewayEmailSender_Send_PayloadFields(t *testing.T) {
	var captured map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	sender := tools.NewGatewayEmailSender(srv.URL+"/api/v1", "key", "from@test.com", "App")
	err := sender.Send(context.Background(), "to@test.com", "My Subject", "<b>html</b>", "plain text")
	require.NoError(t, err)
	assert.Equal(t, "to@test.com", captured["to"])
	assert.Equal(t, "My Subject", captured["subject"])
	assert.Equal(t, "<b>html</b>", captured["body_html"])
	assert.Equal(t, "plain text", captured["body_text"])
}

func TestGatewayEmailSender_Send_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sender := tools.NewGatewayEmailSender(srv.URL+"/api/v1", "key", "from@test.com", "App")
	err := sender.Send(ctx, "to@test.com", "Subject", "html", "text")
	require.Error(t, err)
}
