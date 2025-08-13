package dto

import "github.com/eflowcr/eSTOCK_backend/models/database"

type AdjustmentDetails struct {
	Adjustment database.Adjustment `json:"adjustment"`
	Inventory  database.Inventory  `json:"inventory"`
	Lots       []database.Lot      `json:"lots"`
	Serials    []database.Serial   `json:"serials"`
	Article    database.Article    `json:"article"`
}
