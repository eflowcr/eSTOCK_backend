package tools

import (
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// RunStaleReservationsCleanup libera reservas de picking tasks abandonadas.
// A2: protege contra "stock fantasma" (reserved_qty que nunca se libera).
//
// Con B1 Lazy reservation, solo los tasks `in_progress` tienen reservas aplicadas.
// Entonces el cron maneja DOS casos separados:
//
//  1. in_progress sin actividad >7 días → liberar reservas + marcar abandoned
//  2. open/assigned viejos >7 días     → solo marcar abandoned (no hay reservas)
//
// La "actividad" se mide por pt.updated_at para in_progress (operator puede estar
// en medio del picking; updated_at se refresca en cada CompletePickingLine via GORM).
// Para open/assigned se mide por created_at (esos nunca se modifican hasta que inicie).
func RunStaleReservationsCleanup(db *gorm.DB) error {
	if db == nil {
		return errors.New("cron: nil db")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		// Advisory lock a nivel de transacción — un pod a la vez.
		// Se libera automáticamente al commit/rollback.
		var locked bool
		if err := tx.Raw("SELECT pg_try_advisory_xact_lock(987654322)").Scan(&locked).Error; err != nil {
			return err
		}
		if !locked {
			log.Debug().Msg("stale_reservations: otro pod tiene el lock, skipping")
			return nil
		}

		// Caso 1: in_progress abandonados → liberar reservas primero.
		// Agrupa por (sku, location) recorriendo items→allocations (formato A1 Wave 4).
		if err := tx.Exec(`
			WITH stale AS (
			  SELECT
			      item->>'sku'        AS sku,
			      alloc->>'location'  AS location,
			      SUM((alloc->>'quantity')::numeric) AS qty
			  FROM picking_tasks pt,
			       jsonb_array_elements(pt.items) item,
			       jsonb_array_elements(item->'allocations') alloc
			  WHERE pt.status = 'in_progress'
			    AND pt.updated_at < NOW() - INTERVAL '7 days'
			  GROUP BY 1, 2
			)
			UPDATE inventory i
			   SET reserved_qty = GREATEST(0, reserved_qty - stale.qty),
			       updated_at   = NOW()
			  FROM stale
			 WHERE i.sku = stale.sku AND i.location = stale.location;
		`).Error; err != nil {
			return err
		}

		// Caso 1b: después de liberar reservas, marcar esos in_progress como abandoned.
		if err := tx.Exec(`
			UPDATE picking_tasks
			   SET status = 'abandoned', updated_at = NOW()
			 WHERE status = 'in_progress'
			   AND updated_at < NOW() - INTERVAL '7 days';
		`).Error; err != nil {
			return err
		}

		// Caso 2: open/assigned viejos — no tienen reservas aplicadas, solo marcarlos.
		if err := tx.Exec(`
			UPDATE picking_tasks
			   SET status = 'abandoned', updated_at = NOW()
			 WHERE status IN ('open', 'assigned')
			   AND created_at < NOW() - INTERVAL '7 days';
		`).Error; err != nil {
			return err
		}

		log.Info().Msg("cron: stale_reservations cleanup completed")
		return nil
	})
}

// RunStockAlertAnalysis corre el análisis de alertas de stock (low + expiring).
// HR1-C2: usa pg_try_advisory_xact_lock (transaction-scoped) dentro de db.Transaction(),
// igual que RunStaleReservationsCleanup. El lock se libera automáticamente al commit/rollback,
// eliminando el bug donde lock y unlock corrían en distintas conexiones del pool (session-level
// pg_try_advisory_lock + defer pg_advisory_unlock en conexiones distintas del pool).
// analyzer() corre dentro de la misma transacción; GORM maneja transacciones anidadas vía
// savepoints, por lo que los Begin() internos del repo no causan deadlock.
func RunStockAlertAnalysis(db *gorm.DB, analyzer func() error) error {
	if db == nil {
		return errors.New("cron: nil db")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		// Advisory lock a nivel de transacción — un pod a la vez.
		// Se libera automáticamente al commit/rollback.
		var locked bool
		if err := tx.Raw("SELECT pg_try_advisory_xact_lock(987654321)").Scan(&locked).Error; err != nil {
			return err
		}
		if !locked {
			log.Debug().Msg("cron: stock_alerts: otro pod tiene el lock, skipping")
			return nil
		}
		return analyzer()
	})
}

// LotExpirationWindow describes a lot approaching expiration.
type LotExpirationWindow struct {
	LotID          string
	LotNumber      string
	SKU            string
	ExpirationDate time.Time
	DaysToExpire   int
	AssignedUsers  []string // user IDs to notify (e.g. admins, supervisors)
}

// RunLotExpirationCheck queries lots expiring in the next 8 days and calls notifyFn for each.
// It emits "lot_expiring_7d" for lots with 6-8 days to expire, and "lot_expiring_1d" for 0-2 days.
// notifyFn is responsible for calling NotificationsService.Send; it's injected to avoid import cycles.
func RunLotExpirationCheck(db *gorm.DB, notifyFn func(eventType, title, body string) error) error {
	if db == nil {
		return errors.New("cron: nil db")
	}

	type lotRow struct {
		ID             string     `gorm:"column:id"`
		LotNumber      string     `gorm:"column:lot_number"`
		SKU            string     `gorm:"column:sku"`
		ExpirationDate *time.Time `gorm:"column:expiration_date"`
	}

	var lots []lotRow
	now := time.Now().UTC()
	in8days := now.AddDate(0, 0, 8)

	if err := db.Table("lots").
		Select("id, lot_number, sku, expiration_date").
		Where("expiration_date IS NOT NULL AND expiration_date BETWEEN ? AND ? AND quantity > 0", now, in8days).
		Scan(&lots).Error; err != nil {
		return fmt.Errorf("lot_expiration_check: query: %w", err)
	}

	for _, lot := range lots {
		if lot.ExpirationDate == nil {
			continue
		}
		days := int(lot.ExpirationDate.Sub(now).Hours() / 24)

		var eventType, title, body string
		switch {
		case days <= 2:
			eventType = "lot_expiring_1d"
			title = fmt.Sprintf("⚠ Lote por vencer: %s (%s)", lot.LotNumber, lot.SKU)
			body = fmt.Sprintf("El lote %s del SKU %s vence en %d día(s) (%s).",
				lot.LotNumber, lot.SKU, days, lot.ExpirationDate.Format("2006-01-02"))
		case days <= 8:
			eventType = "lot_expiring_7d"
			title = fmt.Sprintf("Lote próximo a vencer: %s (%s)", lot.LotNumber, lot.SKU)
			body = fmt.Sprintf("El lote %s del SKU %s vence en %d día(s) (%s). Tome acción pronto.",
				lot.LotNumber, lot.SKU, days, lot.ExpirationDate.Format("2006-01-02"))
		default:
			continue
		}

		if notifyFn != nil {
			if err := notifyFn(eventType, title, body); err != nil {
				log.Warn().Err(err).Str("lot", lot.LotNumber).Str("event", eventType).Msg("cron: lot_expiration notify failed")
			}
		}
	}

	return nil
}

// RunLowStockNotifications queries open stock_alerts and calls notifyFn for each unresolved low-stock alert.
// notifyFn is injected to avoid import cycles with services.
func RunLowStockNotifications(db *gorm.DB, notifyFn func(sku, message string) error) error {
	if db == nil {
		return nil
	}

	type alertRow struct {
		SKU     string `gorm:"column:sku"`
		Message string `gorm:"column:message"`
	}

	var alerts []alertRow
	if err := db.Table("stock_alerts").
		Select("sku, message").
		Where("is_resolved = false AND alert_type = 'low_stock'").
		Order("created_at DESC").
		Limit(50).
		Scan(&alerts).Error; err != nil {
		log.Warn().Err(err).Msg("cron: low_stock_notifications query failed")
		return nil
	}

	for _, a := range alerts {
		if notifyFn != nil {
			if err := notifyFn(a.SKU, a.Message); err != nil {
				log.Warn().Err(err).Str("sku", a.SKU).Msg("cron: low_stock notify failed")
			}
		}
	}
	return nil
}

// CronDispatch ejecuta todos los jobs del cron en secuencia.
// Se invoca: una vez al arrancar (tras delay de estabilización) y luego cada hora por el ticker.
// Los errores se loggean sin parar la ejecución del siguiente job.
//
// Callbacks (both optional, pass nil to skip):
//   - lotNotifyFn: called per expiring lot event (eventType, title, body)
//   - lowStockNotifyFn: called per unresolved low-stock alert (sku, message)
func CronDispatch(db *gorm.DB, analyzer func() error, lotNotifyFn func(eventType, title, body string) error, lowStockNotifyFn func(sku, message string) error) {
	if err := RunStockAlertAnalysis(db, analyzer); err != nil {
		log.Error().Err(err).Msg("cron: stock alerts failed")
	}
	if err := RunStaleReservationsCleanup(db); err != nil {
		log.Error().Err(err).Msg("cron: stale reservations cleanup failed")
	}
	if err := RunLotExpirationCheck(db, lotNotifyFn); err != nil {
		log.Error().Err(err).Msg("cron: lot expiration check failed")
	}
	// HR1-M5: wire RunLowStockNotifications so low-stock email alerts are actually sent.
	if err := RunLowStockNotifications(db, lowStockNotifyFn); err != nil {
		log.Error().Err(err).Msg("cron: low stock notifications failed")
	}
}

