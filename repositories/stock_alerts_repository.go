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
	result   *responses.StockAlertResponse
	cachedAt time.Time
}

// redisAnalyzeKeyPrefix is namespaced per tenant so the cached payload from one tenant
// can never be served to another (S3.5 W2-B).
const redisAnalyzeKeyPrefix = "stock_alerts:analyze:"

func redisAnalyzeKey(tenantID string) string {
	return redisAnalyzeKeyPrefix + tenantID
}

type StockAlertsRepository struct {
	DB    *gorm.DB
	Redis *redis.Client // nil → in-memory fallback

	// analyzeMu serializes concurrent Analyze() calls so only one runs at a time.
	// Locking is global (not per-tenant) because TRUNCATE locks the entire stock_alerts
	// table. Per-tenant DELETE in the new tenant-scoped flow could be parallelised, but
	// the simple mutex keeps semantics aligned with the previous implementation.
	analyzeMu sync.Mutex
	// analyzeCache: in-memory fallback when Redis is nil. Map keyed by tenantID so the
	// fallback is also tenant-scoped (otherwise tenant A would receive tenant B's cache).
	analyzeCache map[string]analyzeCache
}

func (r *StockAlertsRepository) GetAllStockAlerts(tenantID string, resolved bool) ([]database.StockAlert, *responses.InternalResponse) {
	var stockAlerts []database.StockAlert

	err := r.DB.
		Table(database.StockAlert{}.TableName()).
		Where("tenant_id = ? AND is_resolved = ?", tenantID, resolved).
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

// Analyze recomputes stock alerts for a single tenant. The previous implementation
// TRUNCATEd the entire stock_alerts table and re-derived alerts from globally-scanned
// inventory/movements; in a multi-tenant deployment that would erase tenant B's alerts
// every time tenant A's user opened the dashboard.
//
// S3.5 W2-B refactor:
//   - DELETE only this tenant's rows instead of TRUNCATE.
//   - All inventory/movement/lot reads carry WHERE tenant_id = ?.
//   - The Redis/in-memory cache key is tenant-scoped.
//
// Cron callers must invoke Analyze() per tenant; see tools/cron.go.
func (r *StockAlertsRepository) Analyze(tenantID string) (*responses.StockAlertResponse, *responses.InternalResponse) {
	if tenantID == "" {
		return nil, &responses.InternalResponse{
			Message:    "tenant_id es requerido para analizar alertas de stock",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}

	// Serialize concurrent calls: only one Analyze() runs at a time across all tenants.
	r.analyzeMu.Lock()
	defer r.analyzeMu.Unlock()

	// --- Cache read (per-tenant) ---
	cacheKey := redisAnalyzeKey(tenantID)
	if r.Redis != nil {
		if cached, err := r.Redis.Get(context.Background(), cacheKey).Bytes(); err == nil {
			var resp responses.StockAlertResponse
			if json.Unmarshal(cached, &resp) == nil {
				return &resp, nil
			}
		}
	} else if r.analyzeCache != nil {
		if entry, ok := r.analyzeCache[tenantID]; ok && entry.result != nil && time.Since(entry.cachedAt) < analyzeCacheTTL {
			return entry.result, nil
		}
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

	// Per-tenant clear (replaces global TRUNCATE). Slower than TRUNCATE but isolation-safe.
	err := tx.
		Exec("DELETE FROM "+database.StockAlert{}.TableName()+" WHERE tenant_id = ?", tenantID).
		Error
	if err != nil {
		tx.Rollback()
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al limpiar alertas de stock existentes",
			Handled: false,
		}
	}

	// Get inventory for this tenant only.
	// S3.5.4 (B16 fix): inventory + inventory_movements tables do NOT have tenant_id columns
	// (deferred to S3.6 structural migration). Scope via JOIN through articles.tenant_id,
	// which is safe because articles.sku has a global UNIQUE index — every SKU belongs to
	// exactly one tenant.
	var inventory []database.Inventory
	err = tx.
		Table(database.Inventory{}.TableName()+" AS i").
		Joins("JOIN articles a ON a.sku = i.sku").
		Where("a.tenant_id = ?", tenantID).
		Select("i.*").
		Find(&inventory).Error

	if err != nil {
		tx.Rollback()
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener el inventario",
			Handled: false,
		}
	}

	// Batch-fetch all outbound movements in the last 30 days for this tenant — one query.
	// Same JOIN-via-articles approach: inventory_movements has no tenant_id column.
	const movementLookbackDays = 30
	lookbackCutoff := time.Now().AddDate(0, 0, -movementLookbackDays)

	var allMovements []database.InventoryMovement
	err = tx.
		Table(database.InventoryMovement{}.TableName()+" AS im").
		Joins("JOIN articles a ON a.sku = im.sku").
		Where("a.tenant_id = ? AND im.movement_type = ? AND im.created_at >= ?", tenantID, "outbound", lookbackCutoff).
		Select("im.*").
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
		movements := movementMap[inventory[i].SKU+"|"+inventory[i].Location]

		analysis, errResponse := analyzeInventoryItem(inventory[i], movements)
		if errResponse != nil {
			tx.Rollback()
			return nil, errResponse
		}

		if analysis != nil && analysis.AlertLevel != "" {
			alert := database.StockAlert{
				ID:               uuid.NewString(),
				TenantID:         tenantID,
				SKU:              analysis.SKU,
				AlertType:        analysis.AlertType,
				CurrentStock:     analysis.CurrentStock,
				RecommendedStock: analysis.RecommendedStock,
				AlertLevel:       analysis.AlertLevel,
				Message:          analysis.Message,
				IsResolved:       false,
				CreatedAt:        time.Now(),
			}

			if analysis.PredictedStockOutDays < math.MaxInt32 {
				days := analysis.PredictedStockOutDays
				alert.PredictedStockOutDays = &days
			} else {
				alert.PredictedStockOutDays = nil
			}

			alerts = append(alerts, alert)
		}
	}

	// Build lot expiration alerts for this tenant only.
	lotAlerts, err := r.buildLotExpirationAlerts(tx, tenantID)
	if err != nil {
		tx.Rollback()
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al generar las alertas de expiración de lotes",
			Handled: false,
		}
	}
	alerts = append(alerts, lotAlerts...)

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

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al confirmar la transacción",
			Handled: false,
		}
	}

	if len(alerts) == 0 {
		return nil, nil
	}

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

	// --- Cache write (per-tenant) ---
	if r.Redis != nil {
		if b, err := json.Marshal(response); err == nil {
			r.Redis.Set(context.Background(), cacheKey, b, analyzeCacheTTL)
		}
	} else {
		if r.analyzeCache == nil {
			r.analyzeCache = make(map[string]analyzeCache)
		}
		r.analyzeCache[tenantID] = analyzeCache{result: response, cachedAt: time.Now()}
	}

	return response, nil
}

func analyzeInventoryItem(item database.Inventory, movements []database.InventoryMovement) (*dto.StockAnalysis, *responses.InternalResponse) {
	quantity := float64(item.Quantity)
	consumptionTrend, errResponse := analyzeConsumptionTrend(movements, int(item.Quantity), 30)

	if errResponse != nil {
		return nil, errResponse
	}

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

	if alertLevel == nil {
		return nil, nil
	}

	recommendedStock := calculateRecommendedStock(quantity, consumptionTrend.AverageDailyConsumption, *alertLevel)

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

// buildLotExpirationAlerts fetches lots for the given tenant and builds alert structs in
// memory — no DB inserts. Used by Analyze() which batch-inserts everything at the end.
func (r *StockAlertsRepository) buildLotExpirationAlerts(tx *gorm.DB, tenantID string) ([]database.StockAlert, error) {
	var alerts []database.StockAlert
	date := tools.GetCurrentTime()

	var lots []database.Lot
	err := tx.
		Table("lots").
		Select("id, lot_number, sku, quantity, expiration_date").
		Where("tenant_id = ? AND expiration_date IS NOT NULL AND expiration_date > ?", tenantID, date).
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
			TenantID:         tenantID,
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
func (r *StockAlertsRepository) generateLotExpirationAlertsInTransaction(tx *gorm.DB, tenantID string) ([]database.StockAlert, error) {
	alerts, err := r.buildLotExpirationAlerts(tx, tenantID)
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

func (r *StockAlertsRepository) LotExpiration(tenantID string) (*responses.StockAlertResponse, *responses.InternalResponse) {
	if tenantID == "" {
		return nil, &responses.InternalResponse{
			Message:    "tenant_id es requerido para generar alertas de expiración de lotes",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}

	tx := r.DB.Begin()
	defer func() {
		if rec := recover(); rec != nil {
			tx.Rollback()
		}
	}()

	alerts, err := r.generateLotExpirationAlertsInTransaction(tx, tenantID)
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

// ResolveAlert flips is_resolved=true for an alert owned by tenantID. Cross-tenant
// resolves return NotFound so the existence of another tenant's alert is not leaked.
func (r *StockAlertsRepository) ResolveAlert(tenantID, alertID string) *responses.InternalResponse {
	var alert database.StockAlert
	err := r.DB.Where("id = ? AND tenant_id = ?", alertID, tenantID).First(&alert).Error
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

func (r *StockAlertsRepository) Summary(tenantID string) (*responses.StockAlertResponse, *responses.InternalResponse) {
	var alerts []database.StockAlert

	err := r.DB.
		Table(database.StockAlert{}.TableName()).
		Where("tenant_id = ? AND is_resolved = ?", tenantID, false).
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

func (r *StockAlertsRepository) ExportAlertsToExcel(tenantID string) ([]byte, *responses.InternalResponse) {
	alerts, errResp := r.GetAllStockAlerts(tenantID, false)
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
