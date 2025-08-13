package dto

type ConsumptionTrend struct {
	AverageDailyConsumption float64    `json:"averageDailyConsumption"`
	Trend                   string `json:"trend"` // 'increasing' | 'decreasing' | 'stable'
	PredictedStockOutDays   float64    `json:"predictedStockOutDays"`
}
