package repositories

import (
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type DashboardRepository struct {
	DB *gorm.DB
}

func (r *DashboardRepository) GetDashboardStats(tasksPeriod string, lowStockThreshold int) (map[string]interface{}, *responses.InternalResponse) {
	if lowStockThreshold <= 0 {
		lowStockThreshold = 20
	}
	var totalSkus int64
	err := r.DB.Table("inventory").Distinct("sku").Count(&totalSkus).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to count SKUs", Handled: false}
	}

	var inventoryValuePtr *float64
	err = r.DB.
		Table("inventory").
		Select("SUM(inventory.quantity * COALESCE(articles.unit_price, 0))").
		Joins("LEFT JOIN articles ON inventory.sku = articles.sku").
		Scan(&inventoryValuePtr).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al calcular el valor del inventario", Handled: false}
	}
	inventoryValue := 0.0
	if inventoryValuePtr != nil {
		inventoryValue = *inventoryValuePtr
	}

	var lowStockCount int64
	err = r.DB.
		Table("inventory").
		Where("quantity < ?", lowStockThreshold).
		Count(&lowStockCount).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al contar el stock bajo", Handled: false}
	}

	var activeReceiving int64
	err = r.DB.
		Table("receiving_tasks").
		Where("status IN ?", []string{"open", "in_progress"}).
		Count(&activeReceiving).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al contar las tareas de recepción", Handled: false}
	}

	var activePicking int64
	err = r.DB.
		Table("picking_tasks").
		Where("status IN ?", []string{"open", "in_progress"}).
		Count(&activePicking).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al contar las tareas de picking", Handled: false}
	}

	// Tasks by period — weekly (by DOW) or monthly (by week-of-month)
	type taskDayRow struct {
		Slot  int   `gorm:"column:slot"`
		Count int64 `gorm:"column:count"`
	}
	var taskDayRows []taskDayRow
	var tasksByDay []map[string]interface{}

	if tasksPeriod == "monthly" {
		err = r.DB.Raw(`
			SELECT
				(EXTRACT(WEEK FROM created_at) - EXTRACT(WEEK FROM date_trunc('month', NOW())))::int + 1 AS slot,
				COUNT(*) AS count
			FROM (
				SELECT created_at FROM receiving_tasks
				UNION ALL
				SELECT created_at FROM picking_tasks
			) t
			WHERE created_at >= date_trunc('month', NOW())
			GROUP BY slot
			ORDER BY slot
		`).Scan(&taskDayRows).Error
		if err != nil {
			return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener tareas por semana", Handled: false}
		}
		weekCounts := make(map[int]int64)
		for _, row := range taskDayRows {
			weekCounts[row.Slot] = row.Count
		}
		tasksByDay = make([]map[string]interface{}, 5)
		for i := 0; i < 5; i++ {
			tasksByDay[i] = map[string]interface{}{
				"day":   "W" + string(rune('0'+i+1)),
				"count": weekCounts[i+1],
			}
		}
	} else {
		// weekly (default): current week by DOW (Sun=0 … Sat=6)
		err = r.DB.Raw(`
			SELECT EXTRACT(DOW FROM created_at)::int AS slot, COUNT(*) AS count
			FROM (
				SELECT created_at FROM receiving_tasks
				UNION ALL
				SELECT created_at FROM picking_tasks
			) t
			WHERE created_at >= date_trunc('week', NOW())
			GROUP BY slot
			ORDER BY slot
		`).Scan(&taskDayRows).Error
		if err != nil {
			return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener tareas por día", Handled: false}
		}
		dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
		dayCounts := make(map[int]int64)
		for _, row := range taskDayRows {
			dayCounts[row.Slot] = row.Count
		}
		tasksByDay = make([]map[string]interface{}, 7)
		for i := 0; i < 7; i++ {
			tasksByDay[i] = map[string]interface{}{
				"day":   dayNames[i],
				"count": dayCounts[i],
			}
		}
	}

	// Movements last 7 days — inbound vs outbound per day
	type movementDayRow struct {
		Date     string `gorm:"column:date"`
		Inbound  int64  `gorm:"column:inbound"`
		Outbound int64  `gorm:"column:outbound"`
	}
	var movementRows []movementDayRow
	err = r.DB.Raw(`
		SELECT TO_CHAR(created_at::date, 'YYYY-MM-DD') AS date,
			SUM(CASE WHEN movement_type = 'inbound' THEN 1 ELSE 0 END) AS inbound,
			SUM(CASE WHEN movement_type = 'outbound' THEN 1 ELSE 0 END) AS outbound
		FROM inventory_movements
		WHERE created_at >= NOW() - INTERVAL '7 days'
		GROUP BY created_at::date
		ORDER BY created_at::date
	`).Scan(&movementRows).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener movimientos de los últimos 7 días", Handled: false}
	}
	// Build a full 7-day series (fill missing dates with 0)
	movementByDate := make(map[string]movementDayRow)
	for _, row := range movementRows {
		movementByDate[row.Date] = row
	}
	movementLast7Days := make([]map[string]interface{}, 7)
	today := time.Now()
	for i := 0; i < 7; i++ {
		d := today.AddDate(0, 0, -(6 - i))
		dateStr := d.Format("2006-01-02")
		row := movementByDate[dateStr]
		movementLast7Days[i] = map[string]interface{}{
			"date":     dateStr,
			"inbound":  row.Inbound,
			"outbound": row.Outbound,
		}
	}

	// KPI trends: compare tasks created in last 30d vs previous 30d
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	sixtyDaysAgo := now.AddDate(0, 0, -60)

	var tasksCurrentPeriod int64
	err = r.DB.Raw(`
		SELECT COUNT(*) FROM (
			SELECT created_at FROM receiving_tasks WHERE created_at >= ?
			UNION ALL
			SELECT created_at FROM picking_tasks WHERE created_at >= ?
		) t
	`, thirtyDaysAgo, thirtyDaysAgo).Scan(&tasksCurrentPeriod).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al calcular tendencia de tareas", Handled: false}
	}

	var tasksPreviousPeriod int64
	err = r.DB.Raw(`
		SELECT COUNT(*) FROM (
			SELECT created_at FROM receiving_tasks WHERE created_at >= ? AND created_at < ?
			UNION ALL
			SELECT created_at FROM picking_tasks WHERE created_at >= ? AND created_at < ?
		) t
	`, sixtyDaysAgo, thirtyDaysAgo, sixtyDaysAgo, thirtyDaysAgo).Scan(&tasksPreviousPeriod).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al calcular tendencia anterior de tareas", Handled: false}
	}

	var tasksChangePercent float64
	if tasksPreviousPeriod > 0 {
		tasksChangePercent = float64(tasksCurrentPeriod-tasksPreviousPeriod) / float64(tasksPreviousPeriod) * 100
	}

	// Low stock trend: compare count vs 30 days ago using movements to approximate
	var movCurrentPeriod int64
	err = r.DB.Table("inventory_movements").
		Where("created_at >= ?", thirtyDaysAgo).
		Count(&movCurrentPeriod).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al calcular tendencia de movimientos", Handled: false}
	}

	var movPreviousPeriod int64
	err = r.DB.Table("inventory_movements").
		Where("created_at >= ? AND created_at < ?", sixtyDaysAgo, thirtyDaysAgo).
		Count(&movPreviousPeriod).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al calcular tendencia anterior de movimientos", Handled: false}
	}

	var movChangePercent float64
	if movPreviousPeriod > 0 {
		movChangePercent = float64(movCurrentPeriod-movPreviousPeriod) / float64(movPreviousPeriod) * 100
	}

	result := map[string]interface{}{
		"totalSkus":          totalSkus,
		"inventoryValue":     inventoryValue,
		"lowStockCount":      lowStockCount,
		"activeTasks":        activeReceiving + activePicking,
		"tasksThisWeek":      tasksByDay,
		"movementLast7Days":  movementLast7Days,
		"tasksChangePercent": tasksChangePercent,
		"movChangePercent":   movChangePercent,
	}

	return result, nil
}

func (r *DashboardRepository) GetInventorySummary(period string) (map[string]interface{}, *responses.InternalResponse) {
	// Top 5 articles by total inventory value (quantity * unit_price) — always current snapshot
	type topArticleRow struct {
		SKU        string  `gorm:"column:sku"`
		Name       string  `gorm:"column:name"`
		Quantity   float64 `gorm:"column:total_quantity"`
		UnitPrice  float64 `gorm:"column:unit_price"`
		TotalValue float64 `gorm:"column:total_value"`
	}
	var topArticles []topArticleRow
	err := r.DB.Raw(`
		SELECT
			i.sku,
			COALESCE(a.name, i.sku) AS name,
			SUM(i.quantity) AS total_quantity,
			COALESCE(a.unit_price, 0) AS unit_price,
			SUM(i.quantity * COALESCE(a.unit_price, 0)) AS total_value
		FROM inventory i
		LEFT JOIN articles a ON i.sku = a.sku
		GROUP BY i.sku, a.name, a.unit_price
		ORDER BY total_value DESC
		LIMIT 5
	`).Scan(&topArticles).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener artículos top", Handled: false}
	}

	var totalValuePtr *float64
	err = r.DB.Raw(`
		SELECT SUM(i.quantity * COALESCE(a.unit_price, 0))
		FROM inventory i
		LEFT JOIN articles a ON i.sku = a.sku
	`).Scan(&totalValuePtr).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al calcular valor total", Handled: false}
	}
	totalValue := 0.0
	if totalValuePtr != nil {
		totalValue = *totalValuePtr
	}

	topArticlesResult := make([]map[string]interface{}, len(topArticles))
	for i, row := range topArticles {
		ratePercent := 0.0
		if totalValue > 0 {
			ratePercent = row.TotalValue / totalValue * 100
		}
		topArticlesResult[i] = map[string]interface{}{
			"id":          row.SKU,
			"name":        row.Name,
			"type":        "SKU",
			"ratePercent": ratePercent,
			"amount":      row.TotalValue,
		}
	}

	// Location distribution — period controls the time window for movement totals
	// monthly=last 30d, quarterly=last 90d, annual=last 365d
	var intervalExpr string
	switch period {
	case "quarterly":
		intervalExpr = "90 days"
	case "annual":
		intervalExpr = "365 days"
	default: // monthly
		intervalExpr = "30 days"
	}

	type locationRow struct {
		Location string  `gorm:"column:location"`
		Count    int64   `gorm:"column:count"`
		Value    float64 `gorm:"column:value"`
	}
	var locationRows []locationRow
	err = r.DB.Raw(`
		SELECT
			im.location,
			COUNT(*) AS count,
			SUM(im.quantity * COALESCE(a.unit_price, 0)) AS value
		FROM inventory_movements im
		LEFT JOIN articles a ON im.sku = a.sku
		WHERE im.created_at >= NOW() - INTERVAL '`+intervalExpr+`'
		GROUP BY im.location
		ORDER BY value DESC
		LIMIT 6
	`).Scan(&locationRows).Error
	if err != nil {
		// Fall back to current inventory snapshot on error
		err = r.DB.Raw(`
			SELECT
				i.location,
				COUNT(*) AS count,
				SUM(i.quantity * COALESCE(a.unit_price, 0)) AS value
			FROM inventory i
			LEFT JOIN articles a ON i.sku = a.sku
			GROUP BY i.location
			ORDER BY value DESC
			LIMIT 6
		`).Scan(&locationRows).Error
		if err != nil {
			return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener distribución por ubicación", Handled: false}
		}
	}

	// Compute total for percentages
	var locTotal float64
	for _, row := range locationRows {
		locTotal += row.Value
	}

	colors := []string{"var(--chart-1)", "var(--chart-2)", "var(--chart-3)", "var(--chart-4)", "var(--chart-5)"}
	locationResult := make([]map[string]interface{}, len(locationRows))
	for i, row := range locationRows {
		pct := 0.0
		if locTotal > 0 {
			pct = row.Value / locTotal * 100
		}
		color := "var(--chart-other)"
		if i < len(colors) {
			color = colors[i]
		}
		locationResult[i] = map[string]interface{}{
			"label":  row.Location,
			"value":  pct,
			"amount": row.Value,
			"color":  color,
		}
	}

	return map[string]interface{}{
		"topArticles":          topArticlesResult,
		"locationDistribution": locationResult,
	}, nil
}

func (r *DashboardRepository) GetMovementsMonthly(period string) (map[string]interface{}, *responses.InternalResponse) {
	type periodRow struct {
		PeriodLabel string `gorm:"column:period_label"`
		SortKey     string `gorm:"column:sort_key"`
		Inbound     int64  `gorm:"column:inbound"`
		Outbound    int64  `gorm:"column:outbound"`
		Adjusted    int64  `gorm:"column:adjusted"`
	}

	var query string
	switch period {
	case "weekly":
		query = `
			SELECT
				TO_CHAR(created_at::date, 'Dy DD') AS period_label,
				created_at::date::text AS sort_key,
				SUM(CASE WHEN movement_type = 'inbound'  THEN 1 ELSE 0 END) AS inbound,
				SUM(CASE WHEN movement_type = 'outbound' THEN 1 ELSE 0 END) AS outbound,
				SUM(CASE WHEN movement_type NOT IN ('inbound', 'outbound') THEN 1 ELSE 0 END) AS adjusted
			FROM inventory_movements
			WHERE created_at >= NOW() - INTERVAL '7 days'
			GROUP BY created_at::date
			ORDER BY created_at::date`
	case "quarterly":
		query = `
			SELECT
				'Q' || EXTRACT(QUARTER FROM created_at)::text || ' ' || EXTRACT(YEAR FROM created_at)::text AS period_label,
				EXTRACT(YEAR FROM created_at)::text || '-' || EXTRACT(QUARTER FROM created_at)::text AS sort_key,
				SUM(CASE WHEN movement_type = 'inbound'  THEN 1 ELSE 0 END) AS inbound,
				SUM(CASE WHEN movement_type = 'outbound' THEN 1 ELSE 0 END) AS outbound,
				SUM(CASE WHEN movement_type NOT IN ('inbound', 'outbound') THEN 1 ELSE 0 END) AS adjusted
			FROM inventory_movements
			WHERE created_at >= NOW() - INTERVAL '12 months'
			GROUP BY EXTRACT(YEAR FROM created_at), EXTRACT(QUARTER FROM created_at)
			ORDER BY EXTRACT(YEAR FROM created_at), EXTRACT(QUARTER FROM created_at)`
	case "annual":
		query = `
			SELECT
				EXTRACT(YEAR FROM created_at)::text AS period_label,
				EXTRACT(YEAR FROM created_at)::text AS sort_key,
				SUM(CASE WHEN movement_type = 'inbound'  THEN 1 ELSE 0 END) AS inbound,
				SUM(CASE WHEN movement_type = 'outbound' THEN 1 ELSE 0 END) AS outbound,
				SUM(CASE WHEN movement_type NOT IN ('inbound', 'outbound') THEN 1 ELSE 0 END) AS adjusted
			FROM inventory_movements
			WHERE created_at >= NOW() - INTERVAL '3 years'
			GROUP BY EXTRACT(YEAR FROM created_at)
			ORDER BY EXTRACT(YEAR FROM created_at)`
	default: // monthly
		query = `
			SELECT
				TO_CHAR(DATE_TRUNC('month', created_at), 'Mon') AS period_label,
				DATE_TRUNC('month', created_at)::text AS sort_key,
				SUM(CASE WHEN movement_type = 'inbound'  THEN 1 ELSE 0 END) AS inbound,
				SUM(CASE WHEN movement_type = 'outbound' THEN 1 ELSE 0 END) AS outbound,
				SUM(CASE WHEN movement_type NOT IN ('inbound', 'outbound') THEN 1 ELSE 0 END) AS adjusted
			FROM inventory_movements
			WHERE created_at >= DATE_TRUNC('month', NOW()) - INTERVAL '5 months'
			GROUP BY DATE_TRUNC('month', created_at)
			ORDER BY DATE_TRUNC('month', created_at)`
	}

	var rows []periodRow
	err := r.DB.Raw(query).Scan(&rows).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener movimientos por período", Handled: false}
	}

	months := make([]map[string]interface{}, len(rows))
	for i, row := range rows {
		total := row.Inbound + row.Outbound + row.Adjusted
		months[i] = map[string]interface{}{
			"period":   row.PeriodLabel,
			"total":    total,
			"inbound":  row.Inbound,
			"outbound": row.Outbound,
			"adjusted": row.Adjusted,
		}
	}

	return map[string]interface{}{"months": months}, nil
}

func (r *DashboardRepository) GetRecentActivity() (map[string]interface{}, *responses.InternalResponse) {
	type activityRow struct {
		ID           string `gorm:"column:id"`
		Action       string `gorm:"column:action"`
		ResourceType string `gorm:"column:resource_type"`
		ResourceID   string `gorm:"column:resource_id"`
		UserEmail    string `gorm:"column:user_email"`
		CreatedAt    string `gorm:"column:created_at"`
	}

	var rows []activityRow
	err := r.DB.Raw(`
		SELECT
			al.id,
			al.action,
			al.resource_type,
			al.resource_id,
			COALESCE(u.email, 'system') AS user_email,
			al.created_at::text AS created_at
		FROM audit_logs al
		LEFT JOIN users u ON al.user_id = u.id
		ORDER BY al.created_at DESC
		LIMIT 10
	`).Scan(&rows).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener actividad reciente", Handled: false}
	}

	// Fall back to inventory_movements if audit_logs is empty
	if len(rows) == 0 {
		type movRow struct {
			ID           string  `gorm:"column:id"`
			MovementType string  `gorm:"column:movement_type"`
			SKU          string  `gorm:"column:sku"`
			Location     string  `gorm:"column:location"`
			Quantity     float64 `gorm:"column:quantity"`
			CreatedBy    string  `gorm:"column:created_by"`
			CreatedAt    string  `gorm:"column:created_at"`
		}
		var movRows []movRow
		err = r.DB.Raw(`
			SELECT id, movement_type, sku, location, quantity, created_by,
				created_at::text AS created_at
			FROM inventory_movements
			ORDER BY created_at DESC
			LIMIT 10
		`).Scan(&movRows).Error
		if err != nil {
			return nil, &responses.InternalResponse{Error: err, Message: "Error al obtener movimientos recientes", Handled: false}
		}
		activities := make([]map[string]interface{}, len(movRows))
		for i, m := range movRows {
			activities[i] = map[string]interface{}{
				"id":      m.ID,
				"type":    m.MovementType,
				"message": m.MovementType + " " + m.SKU + " @ " + m.Location,
				"user":    m.CreatedBy,
				"time":    m.CreatedAt,
			}
		}
		return map[string]interface{}{"activities": activities}, nil
	}

	activities := make([]map[string]interface{}, len(rows))
	for i, row := range rows {
		activities[i] = map[string]interface{}{
			"id":      row.ID,
			"type":    row.Action,
			"message": row.Action + " " + row.ResourceType + " " + row.ResourceID,
			"user":    row.UserEmail,
			"time":    row.CreatedAt,
		}
	}

	return map[string]interface{}{"activities": activities}, nil
}
