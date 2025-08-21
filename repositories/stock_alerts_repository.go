package repositories

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type AlertType string

const (
	AlertTypeLowStock   AlertType = "low_stock"
	AlertTypePredictive AlertType = "predictive"
)

type AlertLevel string

const (
	AlertLevelCritical AlertLevel = "critical"
	AlertLevelHigh     AlertLevel = "high"
	AlertLevelMedium   AlertLevel = "medium"
)

type StockAlertsRepository struct {
	DB *gorm.DB
}

func (r *StockAlertsRepository) GetAllStockAlerts(resolved bool) ([]database.StockAlert, *responses.InternalResponse) {
	var stockAlerts []database.StockAlert

	err := r.DB.
		Table(database.StockAlert{}.TableName()).
		Where("is_resolved = ?", resolved).
		Order("created_at ASC").
		Find(&stockAlerts).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch stock alerts",
			Handled: false,
		}
	}

	if len(stockAlerts) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No stock alerts found",
			Handled: true,
		}
	}

	return stockAlerts, nil
}

func (r *StockAlertsRepository) Analyze() (*responses.StockAlertResponse, *responses.InternalResponse) {
	// Begin transaction
	tx := r.DB.Begin()
	if tx.Error != nil {
		return nil, &responses.InternalResponse{
			Error:   tx.Error,
			Message: "Failed to start transaction",
			Handled: false,
		}
	}

	// Delete all stock alerts where is_resolved is true
	err := tx.
		Table(database.StockAlert{}.TableName()).
		Where("is_resolved = ?", true).
		Delete(&database.StockAlert{}).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to delete resolved stock alerts",
			Handled: false,
		}
	}

	// Get all inventory
	var inventory []database.Inventory
	err = tx.
		Table(database.Inventory{}.TableName()).
		Find(&inventory).Error

	if err != nil {
		tx.Rollback()
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch inventory",
			Handled: false,
		}
	}

	var alerts []database.StockAlert

	for i := 0; i < len(inventory); i++ {
		// Get inventory movements
		var movements []database.InventoryMovement
		err = tx.
			Table(database.InventoryMovement{}.TableName()).
			Where("sku = ? AND location = ?", inventory[i].SKU, inventory[i].Location).
			Order("created_at DESC").
			Find(&movements).Error

		if err != nil {
			tx.Rollback()
			return nil, &responses.InternalResponse{
				Error:   err,
				Message: "Failed to fetch inventory movements",
				Handled: false,
			}
		}

		if len(movements) == 0 {
			continue
		}

		analysis, errResponse := analyzeInventoryItem(inventory[i], movements)
		if errResponse != nil {
			tx.Rollback()
			return nil, errResponse
		}

		if analysis != nil && analysis.AlertLevel != "" {
			alert := database.StockAlert{
				SKU:              analysis.SKU,
				AlertType:        analysis.AlertType,
				CurrentStock:     analysis.CurrentStock,
				RecommendedStock: analysis.RecommendedStock,
				AlertLevel:       analysis.AlertLevel,
				Message:          analysis.Message,
				IsResolved:       false,
				CreatedAt:        time.Now(),
			}

			// Add predicted stock out days only if it's finite and within int4 range
			if !math.IsInf(float64(analysis.PredictedStockOutDays), 0) &&
				!math.IsNaN(float64(analysis.PredictedStockOutDays)) {

				predictedDays := analysis.PredictedStockOutDays

				// Check if value is within PostgreSQL int4 range
				if predictedDays >= -2147483648 && predictedDays <= 2147483647 {
					alert.PredictedStockOutDays = &predictedDays
				} else {
					// Handle out-of-range value (set to nil or use a default)
					alert.PredictedStockOutDays = nil
					// Optionally log this issue
					log.Printf("Warning: PredictedStockOutDays out of range: %d", predictedDays)
				}
			} else {
				alert.PredictedStockOutDays = nil
			}

			alerts = append(alerts, alert)

			// Add alert to the database
			err = tx.Create(&alert).Error
			if err != nil {
				tx.Rollback()
				return nil, &responses.InternalResponse{
					Error:   err,
					Message: "Failed to create stock alert: " + err.Error(),
					Handled: false,
				}
			}
		}
	}

	// Generate lot expiration alerts
	lotAlerts, err := r.generateLotExpirationAlertsInTransaction(tx)
	if err != nil {
		tx.Rollback()
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to generate lot expiration alerts",
			Handled: false,
		}
	}
	alerts = append(alerts, lotAlerts...)

	// Commit transaction
	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to commit transaction",
			Handled: false,
		}
	}

	// If no alerts were generated
	if len(alerts) == 0 {
		return nil, nil
	}

	criticialCount := 0
	highCount := 0
	mediumCount := 0
	expiringCount := 0

	for _, alert := range alerts {
		switch strings.ToLower(strings.TrimSpace(alert.AlertLevel)) {
		case "critical":
			criticialCount++
		case "high":
			highCount++
		case "medium":
			mediumCount++
		}

		if strings.ToLower(strings.TrimSpace(alert.AlertType)) == "lot_expiration" {
			expiringCount++
		}
	}

	response := &responses.StockAlertResponse{
		Message: "Stock alerts generated successfully",
		Alerts:  alerts,
		Summary: responses.StockAlertSumary{
			Total:    len(alerts),
			Critical: 0,
			High:     0,
			Medium:   0,
			Expiring: 0,
		},
	}

	return response, nil
}

func analyzeInventoryItem(item database.Inventory, movements []database.InventoryMovement) (*dto.StockAnalysis, *responses.InternalResponse) {
	quantity := float64(item.Quantity)
	consumptionTrend, errResponse := analyzeConsumptionTrend(movements, int(item.Quantity), 30)

	if errResponse != nil {
		return nil, errResponse
	}

	alertLevel := classifyAlertLevel(int(quantity), int(consumptionTrend.PredictedStockOutDays))

	// If null, no alert is needed
	if alertLevel == nil {
		return nil, nil
	}

	recommendedStock := calculateRecommendedStock(quantity, consumptionTrend.AverageDailyConsumption, *alertLevel)

	// Determine alert type
	alertType := classifyAlertType(int(quantity))

	message := GenerateAlertMessage(
		item.SKU,
		alertType,
		AlertLevel(*alertLevel),
		int(quantity),
		consumptionTrend.PredictedStockOutDays,
		recommendedStock,
	)

	analysis := dto.StockAnalysis{
		SKU:                     item.SKU,
		CurrentStock:            int(quantity),
		AverageDailyConsumption: int(math.Round(consumptionTrend.AverageDailyConsumption)),
		PredictedStockOutDays:   int(math.Floor(consumptionTrend.PredictedStockOutDays)),
		AlertLevel:              *alertLevel,
		RecommendedStock:        recommendedStock,
		AlertType:               string(alertType),
		Message:                 message,
	}

	return &analysis, nil
}

func analyzeConsumptionTrend(
	movements []database.InventoryMovement,
	currentStock, days int,
) (dto.ConsumptionTrend, *responses.InternalResponse) {
	if days <= 0 {
		days = 30
	}

	now := time.Now()
	cutoffDate := now.AddDate(0, 0, -days)

	outbound := make([]database.InventoryMovement, 0, len(movements))
	for _, m := range movements {
		if strings.ToLower(strings.TrimSpace(m.MovementType)) != "outbound" {
			continue
		}
		if m.CreatedAt.IsZero() {
			continue
		}
		if m.CreatedAt.Before(cutoffDate) {
			continue
		}
		outbound = append(outbound, m)
	}

	if len(outbound) == 0 {
		trend := dto.ConsumptionTrend{
			AverageDailyConsumption: 0,
			Trend:                   "stable",
			PredictedStockOutDays:   0,
		}
		if currentStock > 0 {
			trend.PredictedStockOutDays = math.Inf(1)
		}
		return trend, nil
	}

	var totalConsumption float64
	for _, m := range outbound {
		q := float64(m.Quantity)
		if q < 0 {
			q = -q
		}
		totalConsumption += q
	}
	avgDaily := totalConsumption / float64(days)

	midPoint := days / 2
	if midPoint <= 0 {
		midPoint = 1
	}
	recentCutoff := now.AddDate(0, 0, -midPoint)

	var recentSum, olderSum float64
	for _, m := range outbound {
		q := float64(m.Quantity)
		if q < 0 {
			q = -q
		}
		if m.CreatedAt.After(recentCutoff) || m.CreatedAt.Equal(recentCutoff) {
			recentSum += q
		} else {
			olderSum += q
		}
	}
	recentAvg := recentSum / float64(midPoint)
	olderAvg := olderSum / float64(midPoint)

	trendStr := "stable"
	if recentAvg > olderAvg*1.1 {
		trendStr = "increasing"
	} else if recentAvg < olderAvg*0.9 {
		trendStr = "decreasing"
	}

	predicted := math.Inf(1)
	if avgDaily > 0 {
		predicted = float64(currentStock) / avgDaily
	}

	return dto.ConsumptionTrend{
		AverageDailyConsumption: avgDaily,
		Trend:                   trendStr,
		PredictedStockOutDays:   predicted,
	}, nil
}

func classifyAlertLevel(currentStock, predictedStockOutDays int) *string {
	str := func(s string) *string { return &s }

	if predictedStockOutDays <= 7 {
		return str("critical")
	}
	if predictedStockOutDays <= 14 {
		return str("high")
	}

	if currentStock <= 5 {
		return str("critical")
	}
	if currentStock <= 10 {
		return str("high")
	}
	return nil
}

func classifyAlertType(quantity int) AlertType {
	if quantity <= 10 {
		return AlertTypeLowStock
	}
	return AlertTypePredictive
}

func calculateRecommendedStock(currentStock, averageDailyConsumption float64, alertLevel string) int {
	if currentStock < 0 {
		currentStock = 0
	}
	if averageDailyConsumption < 0 {
		averageDailyConsumption = 0
	}

	if currentStock <= 10 {
		v := math.Max(50, currentStock*5)
		return int(math.Ceil(v))
	}

	if averageDailyConsumption > 0 {
		if strings.EqualFold(alertLevel, "critical") {
			return int(math.Ceil(averageDailyConsumption * 30))
		}
		if strings.EqualFold(alertLevel, "high") {
			return int(math.Ceil(averageDailyConsumption * 21))
		}
	}

	v := math.Max(50, currentStock*3)
	return int(math.Ceil(v))
}

func GenerateAlertMessage(
	sku string,
	alertType AlertType,
	alertLevel AlertLevel,
	currentStock int,
	predictedStockOutDays float64,
	recommendedStock int,
) string {
	switch alertType {
	case AlertTypeLowStock:
		switch alertLevel {
		case AlertLevelCritical:
			return fmt.Sprintf("CRITICAL: SKU %s has only %d units remaining. Immediate restocking required.", sku, currentStock)
		case AlertLevelHigh:
			return fmt.Sprintf("HIGH: SKU %s is running low with %d units. Consider restocking soon.", sku, currentStock)
		}

	case AlertTypePredictive:
		if math.IsNaN(predictedStockOutDays) || math.IsInf(predictedStockOutDays, 0) {
			return fmt.Sprintf("Alert for SKU %s: Current stock %d, recommended restock %d units.", sku, currentStock, recommendedStock)
		}
		daysText := int(math.Floor(predictedStockOutDays))
		if daysText < 0 {
			daysText = 0
		}

		switch alertLevel {
		case AlertLevelCritical:
			return fmt.Sprintf("CRITICAL: SKU %s predicted to stock out in %d days. Urgent reorder of %d units recommended.", sku, daysText, recommendedStock)
		case AlertLevelHigh:
			return fmt.Sprintf("HIGH: SKU %s predicted to stock out in %d days. Reorder of %d units recommended.", sku, daysText, recommendedStock)
		}
	}

	// Mensaje por defecto
	return fmt.Sprintf("Alert for SKU %s: Current stock %d, recommended restock %d units.", sku, currentStock, recommendedStock)
}

func (r *StockAlertsRepository) generateLotExpirationAlertsInTransaction(tx *gorm.DB) ([]database.StockAlert, error) {
	var alerts []database.StockAlert
	date := tools.GetCurrentTime()

	var lots []database.Lot

	err := tx.
		Table("lots").
		Select("id, lot_number, sku, quantity, expiration_date").
		Where("expiration_date IS NOT NULL AND expiration_date > ?", date).
		Order("expiration_date ASC").
		Find(&lots).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch lots: %w", err)
	}

	for i := 0; i < len(lots); i++ {
		if lots[i].ExpirationDate == nil {
			continue
		}

		daysToExpire := int(math.Floor(lots[i].ExpirationDate.Sub(date).Hours() / 24))

		var alertLevel *string
		if daysToExpire <= 7 {
			level := "critical"
			alertLevel = &level
		} else if daysToExpire <= 30 {
			level := "high"
			alertLevel = &level
		} else if daysToExpire <= 90 {
			level := "medium"
			alertLevel = &level
		}

		shouldAlert := false

		if daysToExpire <= 7 {
			alertLevel = tools.StrPtr("critical")
			shouldAlert = true
		} else if daysToExpire <= 30 {
			alertLevel = tools.StrPtr("high")
			shouldAlert = true
		} else if daysToExpire <= 90 {
			alertLevel = tools.StrPtr("medium")
			shouldAlert = true
		}

		if shouldAlert {
			alert := database.StockAlert{
				SKU:              lots[i].SKU,
				AlertType:        "lot_expiration",
				CurrentStock:     int(lots[i].Quantity),
				RecommendedStock: 0,
				AlertLevel:       *alertLevel,
				Message: fmt.Sprintf("Lot %s of SKU %s is expiring in %d days (on %s). Current lot quantity: %.2f.",
					lots[i].LotNumber,
					lots[i].SKU,
					daysToExpire,
					lots[i].ExpirationDate.Format("2006-01-02"),
					lots[i].Quantity,
				),
				IsResolved:       false,
				CreatedAt:        time.Now(),
				LotNumber:        &lots[i].LotNumber,
				ExpirationDate:   lots[i].ExpirationDate,
				DaysToExpiration: &daysToExpire,
			}

			alerts = append(alerts, alert)

			// Add alert to the database
			err = tx.Create(&alert).Error
			if err != nil {
				return nil, fmt.Errorf("failed to create lot expiration alert: %w", err)
			}
		}
	}

	return alerts, nil
}

func (r *StockAlertsRepository) LotExpiration() (*responses.StockAlertResponse, *responses.InternalResponse) {
	// Begin transaction
	tx := r.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	alerts, err := r.generateLotExpirationAlertsInTransaction(tx)
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to generate lot expiration alerts",
			Handled: false,
		}
	}

	tx.Commit()

	if len(alerts) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No lot expiration alerts generated",
			Handled: true,
		}
	}

	summary := sumarizeAlerts(alerts)

	response := &responses.StockAlertResponse{
		Message: "Lot expiration alerts generated successfully",
		Alerts:  alerts,
		Summary: summary,
	}

	return response, nil
}

func (r *StockAlertsRepository) ResolveAlert(alertID int) *responses.InternalResponse {
	var alert database.StockAlert
	err := r.DB.First(&alert, alertID).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to find alert",
			Handled: false,
		}
	}

	if alert.IsResolved {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Alert already resolved",
			Handled: true,
		}
	}

	alert.IsResolved = true
	resolveDate := tools.GetCurrentTime()
	alert.ResolvedAt = &resolveDate

	err = r.DB.Save(&alert).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to resolve alert",
			Handled: false,
		}
	}

	return nil
}

func (r *StockAlertsRepository) Summary() (*responses.StockAlertResponse, *responses.InternalResponse) {
	var alerts []database.StockAlert

	err := r.DB.
		Table(database.StockAlert{}.TableName()).
		Where("is_resolved = ?", false).
		Order("created_at ASC").
		Find(&alerts).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch stock alerts",
			Handled: false,
		}
	}

	if len(alerts) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No stock alerts found",
			Handled: true,
		}
	}

	summary := sumarizeAlerts(alerts)

	return &responses.StockAlertResponse{
		Message: "Stock alerts summary fetched successfully",
		Alerts:  alerts,
		Summary: summary,
	}, nil
}

func sumarizeAlerts(alerts []database.StockAlert) responses.StockAlertSumary {
	criticalCount := 0
	highCount := 0
	mediumCount := 0
	expiringCount := 0

	for _, alert := range alerts {
		switch strings.ToLower(strings.TrimSpace(alert.AlertLevel)) {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		case "medium":
			mediumCount++
		}

		if strings.ToLower(strings.TrimSpace(alert.AlertType)) == "lot_expiration" {
			expiringCount++
		}
	}

	return responses.StockAlertSumary{
		Total:    len(alerts),
		Critical: criticalCount,
		High:     highCount,
		Medium:   mediumCount,
		Expiring: expiringCount,
	}
}

func (r *StockAlertsRepository) ExportAlertsToExcel() ([]byte, *responses.InternalResponse) {
	alerts, errResp := r.GetAllStockAlerts(false)
	if errResp != nil {
		return nil, errResp
	}
	if len(alerts) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No stock alerts found to export",
			Handled: true,
		}
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"ID", "SKU", "Alert Type", "Current Stock", "Recommended Stock",
		"Alert Level", "Message", "Created At", "Resolved At", "Is Resolved",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 6)
		_ = f.SetCellValue(sheet, cell, h)
	}

	timeOrEmpty := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.Format(time.RFC3339)
	}
	boolToSiNo := func(b bool) string {
		if b {
			return "Sí"
		}
		return "No"
	}

	for idx, alert := range alerts {
		row := idx + 7
		values := []interface{}{
			alert.ID,
			alert.SKU,
			alert.AlertType,
			alert.CurrentStock,
			alert.RecommendedStock,
			alert.AlertLevel,
			alert.Message,
			alert.CreatedAt.Format(time.RFC3339),
			timeOrEmpty(alert.ResolvedAt),
			boolToSiNo(alert.IsResolved),
		}
		for col, val := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			_ = f.SetCellValue(sheet, cell, val)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to generate Excel file",
			Handled: false,
		}
	}

	return buf.Bytes(), nil
}
