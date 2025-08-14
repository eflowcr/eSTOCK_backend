package responses

import "github.com/eflowcr/eSTOCK_backend/models/database"

type StockAlertResponse struct {
	Message string                `json:"message"`
	Alerts  []database.StockAlert `json:"alerts"`
	Summary StockAlertSumary      `json:"summary"`
}

type StockAlertSumary struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Expiring int `json:"expiring"`
}
