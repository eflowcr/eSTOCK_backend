package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// DashboardRepository defines persistence operations for dashboard stats.
type DashboardRepository interface {
	GetDashboardStats(tasksPeriod string, lowStockThreshold int) (map[string]interface{}, *responses.InternalResponse)
	GetInventorySummary(period string) (map[string]interface{}, *responses.InternalResponse)
	GetMovementsMonthly(period string) (map[string]interface{}, *responses.InternalResponse)
	GetRecentActivity() (map[string]interface{}, *responses.InternalResponse)
}
