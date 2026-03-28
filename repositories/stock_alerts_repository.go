package repositories

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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

// analyzeCacheTTL: how long the last Analyze() result is reused before re-running.
// Multiple users opening /stock-alerts within this window share a single DB run.
const analyzeCacheTTL = 60 * time.Second

type analyzeCache struct {
	result    *responses.StockAlertResponse
	cachedAt  time.Time
}

const redisAnalyzeKey = "stock_alerts:analyze"

type StockAlertsRepository struct {
	DB    *gorm.DB
	Redis *redis.Client // nil → in-memory fallback

	// analyzeMu serializes concurrent Analyze() calls so only one runs at a time.
	analyzeMu    sync.Mutex
	analyzeCache analyzeCache // in-memory fallback when Redis is nil
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
			Message: "Error al obtener las alertas de stock",
			Handled: false,
		}
	}

	return stockAlerts, nil
}

func (r *StockAlertsRepository) Analyze() (*responses.StockAlertResponse, *responses.InternalResponse) {
	// Serialize concurrent calls: only one Analyze() runs at a time.
	r.analyzeMu.Lock()
	defer r.analyzeMu.Unlock()

	// --- Cache read ---
	if r.Redis != nil {
		// Try Redis first (survives restarts, shared across instances).
		if cached, err := r.Redis.Get(context.Background(), redisAnalyzeKey).Bytes(); err == nil {
			var resp responses.StockAlertResponse
			if json.Unmarshal(cached, &resp) == nil {
				return &resp, nil
			}
		}
	} else if r.analyzeCache.result != nil && time.Since(r.analyzeCache.cachedAt) < analyzeCacheTTL {
		// Fall back to in-memory cache.
		return r.analyzeCache.result, nil
	}

	// Begin transaction
	tx := r.DB.Begin()
	if tx.Error != nil {
		return nil, &responses.InternalResponse{
			Error:   tx.Error,
			Message: "Error al iniciar la transacción",
			Handled: false,
		}
	}

	// TRUNCATE is ~10× faster than DELETE for full-table clears (no row-level logging).
	err := tx.Exec("TRUNCATE TABLE " + database.StockAlert{}.TableName()).Error
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al limpiar alertas de stock existentes",
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
			Message: "Error al obtener el inventario",
			Handled: false,
		}
	}

	// Fix 1: Batch-fetch all outbound movements in the last 30 days — one query instead of N.
	const movementLookbackDays = 30
	lookbackCutoff := time.Now().AddDate(0, 0, -movementLookbackDays)

	var allMovements []database.InventoryMovement
	err = tx.
		Table(database.InventoryMovement{}.TableName()).
		Where("movement_type = ? AND created_at >= ?", "outbound", lookbackCutoff).
		Find(&allMovements).Error

	if err != nil {
		tx.Rollback()
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener los movimientos de inventario",
			Handled: false,
		}
	}

	// Build lookup map: "sku|location" → []InventoryMovement
	movementMap := make(map[string][]database.InventoryMovement, len(allMovements))
	for _, m := range allMovements {
		key := m.SKU + "|" + m.Location
		movementMap[key] = append(movementMap[key], m)
	}

	var alerts []database.StockAlert

	for i := 0; i < len(inventory); i++ {
		// Look up pre-fetched movements — empty slice if none in last 30 days.
		// analyzeInventoryItem handles empty slices by classifying on stock level alone.
		movements := movementMap[inventory[i].SKU+"|"+inventory[i].Location]

		analysis, errResponse := analyzeInventoryItem(inventory[i], movements)
		if errResponse != nil {
			tx.Rollback()
			return nil, errResponse
		}

		if analysis != nil && analysis.AlertLevel != "" {
			// Use Go-generated UUID — no DB round-trip per alert.
			alert := database.StockAlert{
				ID:               uuid.NewString(),
				SKU:              analysis.SKU,
				AlertType:        analysis.AlertType,
				CurrentStock:     analysis.CurrentStock,
				RecommendedStock: analysis.RecommendedStock,
				AlertLevel:       analysis.AlertLevel,
				Message:          analysis.Message,
				IsResolved:       false,
				CreatedAt:        time.Now(),
			}

			// Fix 2 (tail): sentinel MaxInt32 means "infinite" — store nil in DB.
			if analysis.PredictedStockOutDays < math.MaxInt32 {
				days := analysis.PredictedStockOutDays
				alert.PredictedStockOutDays = &days
			} else {
				alert.PredictedStockOutDays = nil
			}

			alerts = append(alerts, alert)
		}
	}

	// Build lot expiration alerts in memory (no individual inserts).
	lotAlerts, err := r.buildLotExpirationAlerts(tx)
	if err != nil {
		tx.Rollback()
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al generar las alertas de expiración de lotes",
			Handled: false,
		}
	}
	alerts = append(alerts, lotAlerts...)

	// Batch-insert all alerts in a single statement.
	if len(alerts) > 0 {
		err = tx.Create(&alerts).Error
		if err != nil {
			tx.Rollback()
			return nil, &responses.InternalResponse{
				Error:   err,
				Message: "Error al guardar las alertas de stock",
				Handled: false,
			}
		}
	}

	// Commit transaction
	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al confirmar la transacción",
			Handled: false,
		}
	}

	// If no alerts were generated
	if len(alerts) == 0 {
		return nil, nil
	}

	// Fix 4: compute summary counts and use them (were hardcoded to 0 before).
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

	response := &responses.StockAlertResponse{
		Message: "Alertas de stock generadas con éxito",
		Alerts:  alerts,
		Summary: responses.StockAlertSumary{
			Total:    len(alerts),
			Critical: criticalCount,
			High:     highCount,
			Medium:   mediumCount,
			Expiring: expiringCount,
		},
	}

	// --- Cache write ---
	if r.Redis != nil {
		if b, err := json.Marshal(response); err == nil {
			r.Redis.Set(context.Background(), redisAnalyzeKey, b, analyzeCacheTTL)
		}
	} else {
		r.analyzeCache = analyzeCache{result: response, cachedAt: time.Now()}
	}

	return response, nil
}

func analyzeInventoryItem(item database.Inventory, movements []database.InventoryMovement) (*dto.StockAnalysis, *responses.InternalResponse) {
	quantity := float64(item.Quantity)
	consumptionTrend, errResponse := analyzeConsumptionTrend(movements, int(item.Quantity), 30)

	if errResponse != nil {
		return nil, errResponse
	}

	// Fix 2: Cap before int conversion to prevent +Inf → MaxInt64 overflow.
	predictedFloat := consumptionTrend.PredictedStockOutDays
	var predictedInt int
	switch {
	case math.IsInf(predictedFloat, 0) || math.IsNaN(predictedFloat) || predictedFloat > 2147483647:
		predictedInt = math.MaxInt32 // sentinel: stock won't run out soon; classify by absolute level
	case predictedFloat < 0:
		predictedInt = 0
	default:
		predictedInt = int(math.Floor(predictedFloat))
	}

	alertLevel := classifyAlertLevel(int(quantity), predictedInt)

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
		PredictedStockOutDays:   predictedInt,
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
			return fmt.Sprintf("Crítico: SKU %s tiene solo %d unidades restantes. Se requiere un reabastecimiento inmediato.", sku, currentStock)
		case AlertLevelHigh:
			return fmt.Sprintf("Alto: SKU %s está quedando bajo con %d unidades. Considere reabastecer pronto.", sku, currentStock)
		}

	case AlertTypePredictive:
		if math.IsNaN(predictedStockOutDays) || math.IsInf(predictedStockOutDays, 0) {
			return fmt.Sprintf("Alerta para SKU %s: Stock actual %d, se recomienda un nuevo pedido de %d unidades.", sku, currentStock, recommendedStock)
		}
		daysText := int(math.Floor(predictedStockOutDays))
		if daysText < 0 {
			daysText = 0
		}

		switch alertLevel {
		case AlertLevelCritical:
			return fmt.Sprintf("Crítico: SKU %s predice que se agotará en %d días. Se recomienda un pedido urgente de %d unidades.", sku, daysText, recommendedStock)
		case AlertLevelHigh:
			return fmt.Sprintf("Alto: SKU %s predice que se agotará en %d días. Se recomienda un pedido de %d unidades.", sku, daysText, recommendedStock)
		}
	}

	// Mensaje por defecto
	return fmt.Sprintf("Alerta para SKU %s: Stock actual %d, se recomienda un nuevo pedido de %d unidades.", sku, currentStock, recommendedStock)
}

// buildLotExpirationAlerts fetches lots and builds alert structs in memory — no DB inserts.
// Used by Analyze() which batch-inserts everything at the end.
func (r *StockAlertsRepository) buildLotExpirationAlerts(tx *gorm.DB) ([]database.StockAlert, error) {
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

	for i := range lots {
		if lots[i].ExpirationDate == nil {
			continue
		}

		daysToExpire := int(math.Floor(lots[i].ExpirationDate.Sub(date).Hours() / 24))

		var level string
		switch {
		case daysToExpire <= 7:
			level = "critical"
		case daysToExpire <= 30:
			level = "high"
		case daysToExpire <= 90:
			level = "medium"
		default:
			continue
		}

		daysToExpireCopy := daysToExpire
		alerts = append(alerts, database.StockAlert{
			ID:               uuid.NewString(),
			SKU:              lots[i].SKU,
			AlertType:        "lot_expiration",
			CurrentStock:     int(lots[i].Quantity),
			RecommendedStock: 0,
			AlertLevel:       level,
			Message: fmt.Sprintf("Lote %s del SKU %s está por expirar en %d días (el %s). Cantidad actual del lote: %.2f.",
				lots[i].LotNumber, lots[i].SKU, daysToExpire,
				lots[i].ExpirationDate.Format("2006-01-02"), lots[i].Quantity,
			),
			IsResolved:       false,
			CreatedAt:        time.Now(),
			LotNumber:        &lots[i].LotNumber,
			ExpirationDate:   lots[i].ExpirationDate,
			DaysToExpiration: &daysToExpireCopy,
		})
	}

	return alerts, nil
}

// generateLotExpirationAlertsInTransaction builds and immediately inserts lot expiration alerts.
// Used by LotExpiration() only.
func (r *StockAlertsRepository) generateLotExpirationAlertsInTransaction(tx *gorm.DB) ([]database.StockAlert, error) {
	alerts, err := r.buildLotExpirationAlerts(tx)
	if err != nil {
		return nil, err
	}
	if len(alerts) > 0 {
		if err = tx.Create(&alerts).Error; err != nil {
			return nil, fmt.Errorf("failed to create lot expiration alerts: %w", err)
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
			Message: "Error al generar las alertas de expiración de lotes",
			Handled: false,
		}
	}

	tx.Commit()

	if len(alerts) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No se generaron alertas de expiración de lotes",
			Handled: true,
		}
	}

	summary := sumarizeAlerts(alerts)

	response := &responses.StockAlertResponse{
		Message: "Alertas de expiración de lotes generadas con éxito",
		Alerts:  alerts,
		Summary: summary,
	}

	return response, nil
}

func (r *StockAlertsRepository) ResolveAlert(alertID string) *responses.InternalResponse {
	var alert database.StockAlert
	err := r.DB.Where("id = ?", alertID).First(&alert).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Error al encontrar la alerta",
			Handled: false,
		}
	}

	if alert.IsResolved {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Alerta ya resuelta",
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
			Message: "Error al resolver la alerta",
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
			Message: "Error al obtener las alertas de stock",
			Handled: false,
		}
	}

	if len(alerts) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No se encontraron alertas de stock",
			Handled: true,
		}
	}

	summary := sumarizeAlerts(alerts)

	return &responses.StockAlertResponse{
		Message: "Resumen de alertas de stock obtenido con éxito",
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
			Message: "No se encontraron alertas de stock para exportar",
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
			Message: "Error al generar el archivo Excel",
			Handled: false,
		}
	}

	return buf.Bytes(), nil
}
