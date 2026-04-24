package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// StockSettingsRepository defines persistence operations for per-tenant stock configuration.
type StockSettingsRepository interface {
	GetOrCreate(tenantID string) (*database.StockSetting, *responses.InternalResponse)
	Upsert(tenantID string, data *requests.UpdateStockSettingsRequest) (*database.StockSetting, *responses.InternalResponse)
}
