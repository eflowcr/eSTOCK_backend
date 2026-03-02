package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// DashboardRepository defines persistence operations for dashboard stats.
type DashboardRepository interface {
	GetDashboardStats() (map[string]interface{}, *responses.InternalResponse)
}
