package dto

type StockAnalysis struct {
	SKU                     string `json:"sku"`
	CurrentStock            int    `json:"currentStock"`
	AverageDailyConsumption int    `json:"averageDailyConsumption"`
	PredictedStockOutDays   int    `json:"predictedStockOutDays"`
	AlertLevel              string `json:"alertLevel"` // 'critical' | 'high' | 'medium' | null
	RecommendedStock        int    `json:"recommendedStock"`
	AlertType               string `json:"alertType"` // 'low_stock' | 'predictive'
	Message                 string `json:"message"`
}
