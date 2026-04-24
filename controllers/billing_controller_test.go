package controllers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripe "github.com/stripe/stripe-go/v79"
)

// ─── mock billing repository ─────────────────────────────────────────────────

type mockBillingRepo struct {
	sub              *database.Subscription
	getSubErr        *responses.InternalResponse
	upsertErr        *responses.InternalResponse
	updateStatusErr  *responses.InternalResponse
	updateCustomerErr *responses.InternalResponse
	updateTenantErr  *responses.InternalResponse
	processedEvents  map[string]bool
	adminUserID      string
}

func (m *mockBillingRepo) GetSubscriptionByTenant(_ string) (*database.Subscription, *responses.InternalResponse) {
	return m.sub, m.getSubErr
}

func (m *mockBillingRepo) UpsertSubscription(sub *database.Subscription) *responses.InternalResponse {
	m.sub = sub
	return m.upsertErr
}

func (m *mockBillingRepo) UpdateSubscriptionStatus(_, status string, cancelAtPeriodEnd bool, update *database.Subscription) *responses.InternalResponse {
	if m.sub != nil {
		m.sub.Status = status
		m.sub.CancelAtPeriodEnd = cancelAtPeriodEnd
	}
	return m.updateStatusErr
}

func (m *mockBillingRepo) UpdateStripeCustomerID(_ string, customerID string) *responses.InternalResponse {
	if m.sub != nil {
		m.sub.StripeCustomerID = &customerID
	}
	return m.updateCustomerErr
}

func (m *mockBillingRepo) UpdateTenantStatus(_ string, _ string) *responses.InternalResponse {
	return m.updateTenantErr
}

func (m *mockBillingRepo) IsWebhookEventProcessed(eventID string) (bool, *responses.InternalResponse) {
	if m.processedEvents == nil {
		return false, nil
	}
	return m.processedEvents[eventID], nil
}

func (m *mockBillingRepo) MarkWebhookEventProcessed(eventID string) *responses.InternalResponse {
	if m.processedEvents == nil {
		m.processedEvents = make(map[string]bool)
	}
	m.processedEvents[eventID] = true
	return nil
}

func (m *mockBillingRepo) GetTenantAdminUserID(_ string) (string, *responses.InternalResponse) {
	return m.adminUserID, nil
}

var _ ports.BillingRepository = (*mockBillingRepo)(nil)

// ─── helpers ─────────────────────────────────────────────────────────────────

const (
	testWebhookSecret = "whsec_test_secret_for_unit_tests"
	testTenantID      = "00000000-0000-0000-0000-000000000001"
	testPriceStarter  = "price_starter_test"
)

func newTestBillingConfig() configuration.Config {
	return configuration.Config{
		StripeSecretKey:      "sk_test_dummy",
		StripeWebhookSecret:  testWebhookSecret,
		StripePriceStarter:   testPriceStarter,
		StripePricePro:       "price_pro_test",
		StripePriceEnterprise: "price_enterprise_test",
		AppURL:               "http://localhost:4200",
		TenantID:             testTenantID,
	}
}

func newBillingController(repo *mockBillingRepo) *BillingController {
	cfg := newTestBillingConfig()
	// NOTE: NewBillingService sets stripe.Key globally. In tests this is a dummy
	// value — Stripe API calls will fail, which is expected (we test non-Stripe flows).
	svc := services.NewBillingService(repo, nil, testTenantID, cfg)
	return NewBillingController(svc, testTenantID, testWebhookSecret)
}

func newBillingRouter(ctrl *BillingController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/billing/checkout", ctrl.Checkout)
	r.GET("/billing/subscription", ctrl.GetSubscription)
	r.POST("/billing/portal-session", ctrl.PortalSession)
	r.POST("/billing/stripe-webhook", ctrl.StripeWebhook)
	return r
}

// stripeWebhookSignature builds a test Stripe-Signature header using HMAC-SHA256.
// This mirrors what the Stripe library uses to verify webhook signatures.
func stripeWebhookSignature(t *testing.T, payload []byte, secret string, ts time.Time) string {
	t.Helper()
	timestamp := fmt.Sprintf("%d", ts.Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	sig := fmt.Sprintf("%x", mac.Sum(nil))
	return fmt.Sprintf("t=%s,v1=%s", timestamp, sig)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestBillingController_Checkout_InvalidPlan(t *testing.T) {
	repo := &mockBillingRepo{}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	body, _ := json.Marshal(map[string]string{"plan": "invalid_plan"})
	req := httptest.NewRequest(http.MethodPost, "/billing/checkout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBillingController_Checkout_MissingPlan(t *testing.T) {
	repo := &mockBillingRepo{}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/billing/checkout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBillingController_GetSubscription_NoSub(t *testing.T) {
	repo := &mockBillingRepo{sub: nil}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billing/subscription", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	assert.Nil(t, data["subscription"])
}

func TestBillingController_GetSubscription_WithSub(t *testing.T) {
	now := time.Now()
	end := now.Add(30 * 24 * time.Hour)
	custID := "cus_test123"
	subID := "sub_test123"
	repo := &mockBillingRepo{
		sub: &database.Subscription{
			ID:                   "sub-local-1",
			TenantID:             testTenantID,
			Plan:                 "starter",
			Status:               "active",
			StripeCustomerID:     &custID,
			StripeSubscriptionID: &subID,
			CurrentPeriodStart:   &now,
			CurrentPeriodEnd:     &end,
			CancelAtPeriodEnd:    false,
		},
	}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billing/subscription", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	sub := data["subscription"].(map[string]interface{})
	assert.Equal(t, "starter", sub["plan"])
	assert.Equal(t, "active", sub["status"])
}

func TestBillingController_GetSubscription_RepoError(t *testing.T) {
	repo := &mockBillingRepo{
		getSubErr: &responses.InternalResponse{
			Error:      fmt.Errorf("db error"),
			Message:    "db error",
			Handled:    false,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billing/subscription", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestBillingController_PortalSession_NoSubscription(t *testing.T) {
	repo := &mockBillingRepo{sub: nil}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	req := httptest.NewRequest(http.MethodPost, "/billing/portal-session", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBillingController_PortalSession_NoStripeCustomer(t *testing.T) {
	repo := &mockBillingRepo{
		sub: &database.Subscription{
			ID:       "sub-1",
			TenantID: testTenantID,
			Plan:     "starter",
			Status:   "active",
			// StripeCustomerID intentionally nil
		},
	}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	req := httptest.NewRequest(http.MethodPost, "/billing/portal-session", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── webhook tests ───────────────────────────────────────────────────────────

func TestBillingController_Webhook_MissingSignature(t *testing.T) {
	repo := &mockBillingRepo{}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	payload := []byte(`{"type":"checkout.session.completed","id":"evt_test"}`)
	req := httptest.NewRequest(http.MethodPost, "/billing/stripe-webhook", bytes.NewReader(payload))
	// No Stripe-Signature header
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBillingController_Webhook_InvalidSignature(t *testing.T) {
	repo := &mockBillingRepo{}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	payload := []byte(`{"type":"checkout.session.completed","id":"evt_test"}`)
	req := httptest.NewRequest(http.MethodPost, "/billing/stripe-webhook", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", "t=1234567890,v1=badhash")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBillingController_Webhook_UnknownEventType_Returns200(t *testing.T) {
	repo := &mockBillingRepo{}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	// Build a minimal valid Stripe event payload for an unknown event type.
	ts := time.Now()
	eventPayload := map[string]interface{}{
		"id":      "evt_unknown_type",
		"type":    "some.unknown.event",
		"object":  "event",
		"created": ts.Unix(),
		"data": map[string]interface{}{
			"object": map[string]interface{}{},
		},
		"livemode":          false,
		"pending_webhooks":  0,
		"request":           nil,
		"api_version":       "2024-06-20",
	}
	payload, err := json.Marshal(eventPayload)
	require.NoError(t, err)

	sig := stripeWebhookSignature(t, payload, testWebhookSecret, ts)
	req := httptest.NewRequest(http.MethodPost, "/billing/stripe-webhook", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sig)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBillingController_Webhook_DuplicateEvent_Skipped(t *testing.T) {
	repo := &mockBillingRepo{
		processedEvents: map[string]bool{"evt_already_done": true},
	}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	ts := time.Now()
	eventPayload := map[string]interface{}{
		"id":      "evt_already_done",
		"type":    "checkout.session.completed",
		"object":  "event",
		"created": ts.Unix(),
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":       "cs_test",
				"object":   "checkout.session",
				"metadata": map[string]interface{}{},
			},
		},
		"livemode":         false,
		"pending_webhooks": 0,
		"request":          nil,
		"api_version":      "2024-06-20",
	}
	payload, _ := json.Marshal(eventPayload)
	sig := stripeWebhookSignature(t, payload, testWebhookSecret, ts)

	req := httptest.NewRequest(http.MethodPost, "/billing/stripe-webhook", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sig)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBillingController_Webhook_CheckoutCompleted_SetsActive(t *testing.T) {
	repo := &mockBillingRepo{}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	ts := time.Now()
	eventPayload := map[string]interface{}{
		"id":      "evt_checkout_done",
		"type":    "checkout.session.completed",
		"object":  "event",
		"created": ts.Unix(),
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":     "cs_test123",
				"object": "checkout.session",
				"mode":   "subscription",
				"metadata": map[string]string{
					"tenant_id": testTenantID,
					"plan":      "starter",
				},
				"customer": map[string]interface{}{
					"id": "cus_test456",
				},
				"subscription": map[string]interface{}{
					"id": "sub_test789",
				},
			},
		},
		"livemode":         false,
		"pending_webhooks": 0,
		"request":          nil,
		"api_version":      "2024-06-20",
	}
	payload, _ := json.Marshal(eventPayload)
	sig := stripeWebhookSignature(t, payload, testWebhookSecret, ts)

	req := httptest.NewRequest(http.MethodPost, "/billing/stripe-webhook", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sig)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Subscription should have been upserted.
	require.NotNil(t, repo.sub)
	assert.Equal(t, "active", repo.sub.Status)
	assert.Equal(t, "starter", repo.sub.Plan)
	// Event should be marked processed.
	assert.True(t, repo.processedEvents["evt_checkout_done"])
}

func TestBillingController_Webhook_PaymentFailed_SetsPastDue(t *testing.T) {
	subID := "sub_existing"
	repo := &mockBillingRepo{
		sub: &database.Subscription{
			ID:                   "local-sub-1",
			TenantID:             testTenantID,
			Status:               "active",
			StripeSubscriptionID: &subID,
		},
	}
	ctrl := newBillingController(repo)
	r := newBillingRouter(ctrl)

	ts := time.Now()
	eventPayload := map[string]interface{}{
		"id":      "evt_payment_failed",
		"type":    "invoice.payment_failed",
		"object":  "event",
		"created": ts.Unix(),
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":     "in_test",
				"object": "invoice",
				"subscription": map[string]interface{}{
					"id": subID,
					"metadata": map[string]string{
						"tenant_id": testTenantID,
					},
				},
			},
		},
		"livemode":         false,
		"pending_webhooks": 0,
		"request":          nil,
		"api_version":      "2024-06-20",
	}
	payload, _ := json.Marshal(eventPayload)
	sig := stripeWebhookSignature(t, payload, testWebhookSecret, ts)

	req := httptest.NewRequest(http.MethodPost, "/billing/stripe-webhook", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sig)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Subscription status should have been updated to past_due.
	assert.Equal(t, "past_due", repo.sub.Status)
}

// ─── service-level unit tests (no HTTP) ──────────────────────────────────────

func TestBillingService_PriceIDForPlan(t *testing.T) {
	repo := &mockBillingRepo{}
	cfg := newTestBillingConfig()
	svc := services.NewBillingService(repo, nil, testTenantID, cfg)

	id, err := svc.PriceIDForPlan("starter")
	assert.NoError(t, err)
	assert.Equal(t, testPriceStarter, id)

	_, err = svc.PriceIDForPlan("unknown")
	assert.Error(t, err)
}

func TestBillingService_IsWebhookEventProcessed(t *testing.T) {
	repo := &mockBillingRepo{
		processedEvents: map[string]bool{"evt_done": true},
	}
	cfg := newTestBillingConfig()
	svc := services.NewBillingService(repo, nil, testTenantID, cfg)

	done, resp := svc.IsWebhookEventProcessed("evt_done")
	assert.Nil(t, resp)
	assert.True(t, done)

	notDone, resp := svc.IsWebhookEventProcessed("evt_new")
	assert.Nil(t, resp)
	assert.False(t, notDone)
}

func TestBillingService_HandleSubscriptionDeleted(t *testing.T) {
	subID := "sub_to_delete"
	repo := &mockBillingRepo{
		sub: &database.Subscription{
			ID:                   "local-1",
			TenantID:             testTenantID,
			Status:               "active",
			StripeSubscriptionID: &subID,
		},
	}
	cfg := newTestBillingConfig()
	svc := services.NewBillingService(repo, nil, testTenantID, cfg)

	stripeSub := &stripe.Subscription{
		ID: subID,
		Metadata: map[string]string{
			"tenant_id": testTenantID,
		},
	}
	resp := svc.HandleSubscriptionDeleted(stripeSub)
	assert.Nil(t, resp)
	assert.Equal(t, "cancelled", repo.sub.Status)
}

func TestBillingService_HandleSubscriptionUpdated(t *testing.T) {
	subID := "sub_to_update"
	repo := &mockBillingRepo{
		sub: &database.Subscription{
			ID:                   "local-1",
			TenantID:             testTenantID,
			Status:               "active",
			StripeSubscriptionID: &subID,
		},
	}
	cfg := newTestBillingConfig()
	svc := services.NewBillingService(repo, nil, testTenantID, cfg)

	now := time.Now()
	stripeSub := &stripe.Subscription{
		ID:                subID,
		Status:            stripe.SubscriptionStatus("past_due"),
		CancelAtPeriodEnd: true,
		CurrentPeriodStart: now.Unix(),
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour).Unix(),
		Metadata: map[string]string{
			"tenant_id": testTenantID,
			"plan":      "pro",
		},
	}
	resp := svc.HandleSubscriptionUpdated(stripeSub)
	assert.Nil(t, resp)
	assert.Equal(t, "past_due", repo.sub.Status)
	assert.True(t, repo.sub.CancelAtPeriodEnd)
}
