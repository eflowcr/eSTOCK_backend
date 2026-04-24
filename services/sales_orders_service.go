package services

import (
	"fmt"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

// SalesOrdersService implements business logic for sales orders.
type SalesOrdersService struct {
	Repository     ports.SalesOrdersRepository
	ClientsService clientLookup // optional: validate customer_id
}

func NewSalesOrdersService(repo ports.SalesOrdersRepository) *SalesOrdersService {
	return &SalesOrdersService{Repository: repo}
}

// WithClientsService attaches an optional ClientsService for customer validation.
func (s *SalesOrdersService) WithClientsService(cs clientLookup) *SalesOrdersService {
	s.ClientsService = cs
	return s
}

// validateCustomer checks that the client exists and is type customer or both.
func (s *SalesOrdersService) validateCustomer(customerID string) *responses.InternalResponse {
	if s.ClientsService == nil {
		return nil // skip validation when not wired
	}
	client, resp := s.ClientsService.GetByID(customerID)
	if resp != nil {
		return &responses.InternalResponse{
			Message:    fmt.Sprintf("customer_id inválido: %s", resp.Message),
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	if client.Type != "customer" && client.Type != "both" {
		return &responses.InternalResponse{
			Message:    fmt.Sprintf("el cliente '%s' es de tipo '%s', se requiere 'customer' o 'both'", customerID, client.Type),
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	return nil
}

// validateItems validates that there are no duplicate SKUs in the request.
func validateSOItems(items []requests.CreateSalesOrderItem) *responses.InternalResponse {
	seen := make(map[string]bool, len(items))
	for _, it := range items {
		if seen[it.ArticleSKU] {
			return &responses.InternalResponse{
				Message:    fmt.Sprintf("SKU duplicado en los items: %s", it.ArticleSKU),
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		seen[it.ArticleSKU] = true
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SO1 — CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (s *SalesOrdersService) Create(tenantID, userID string, req *requests.CreateSalesOrderRequest) (*responses.SalesOrderResponse, *responses.InternalResponse) {
	if resp := s.validateCustomer(req.CustomerID); resp != nil {
		return nil, resp
	}
	if resp := validateSOItems(req.Items); resp != nil {
		return nil, resp
	}
	return s.Repository.Create(tenantID, userID, req)
}

func (s *SalesOrdersService) List(tenantID string, status, customerID, search *string, dateFrom, dateTo *string, page, limit int) (*responses.SalesOrderListResponse, *responses.InternalResponse) {
	return s.Repository.List(tenantID, status, customerID, search, dateFrom, dateTo, page, limit)
}

func (s *SalesOrdersService) GetByID(id, tenantID string) (*responses.SalesOrderResponse, *responses.InternalResponse) {
	return s.Repository.GetByID(id, tenantID)
}

func (s *SalesOrdersService) Update(id, tenantID string, req *requests.UpdateSalesOrderRequest) (*responses.SalesOrderResponse, *responses.InternalResponse) {
	if req.CustomerID != nil {
		if resp := s.validateCustomer(*req.CustomerID); resp != nil {
			return nil, resp
		}
	}
	if len(req.Items) > 0 {
		if resp := validateSOItems(req.Items); resp != nil {
			return nil, resp
		}
	}
	return s.Repository.Update(id, tenantID, req)
}

func (s *SalesOrdersService) SoftDelete(id, tenantID string) *responses.InternalResponse {
	return s.Repository.SoftDelete(id, tenantID)
}

// ─────────────────────────────────────────────────────────────────────────────
// SO2 — Lifecycle
// ─────────────────────────────────────────────────────────────────────────────

func (s *SalesOrdersService) Submit(id, tenantID, userID string) (*responses.SubmitSalesOrderResult, *responses.InternalResponse) {
	return s.Repository.Submit(id, tenantID, userID)
}

func (s *SalesOrdersService) Cancel(id, tenantID, userID string) *responses.InternalResponse {
	return s.Repository.Cancel(id, tenantID, userID)
}

// ─────────────────────────────────────────────────────────────────────────────
// SO3 — Picking auto-link (called indirectly via PickingTaskRepository)
// ─────────────────────────────────────────────────────────────────────────────

func (s *SalesOrdersService) UpdatePickedQty(salesOrderID string, pickedPerSKU map[string]float64) *responses.InternalResponse {
	return s.Repository.UpdatePickedQty(salesOrderID, pickedPerSKU)
}

// Compile-time check: ClientsService satisfies clientLookup (defined in receiving_tasks_service.go).
var _ clientLookup = (*ClientsService)(nil)
