package tools

import (
	"errors"

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
// A7: advisory lock a nivel de sesión para no duplicar en multi-replica.
// El caller provee la función analyzer que crea el repo/service y llama Analyze()+LotExpiration().
func RunStockAlertAnalysis(db *gorm.DB, analyzer func() error) error {
	if db == nil {
		return errors.New("cron: nil db")
	}
	var locked bool
	if err := db.Raw("SELECT pg_try_advisory_lock(987654321)").Scan(&locked).Error; err != nil {
		return err
	}
	if !locked {
		log.Debug().Msg("cron: stock_alerts: otro pod tiene el lock, skipping")
		return nil
	}
	defer db.Exec("SELECT pg_advisory_unlock(987654321)")

	return analyzer()
}

// CronDispatch ejecuta todos los jobs del cron en secuencia.
// Se invoca: una vez al arrancar (tras delay de estabilización) y luego cada hora por el ticker.
// Los errores se loggean sin parar la ejecución del siguiente job.
func CronDispatch(db *gorm.DB, analyzer func() error) {
	if err := RunStockAlertAnalysis(db, analyzer); err != nil {
		log.Error().Err(err).Msg("cron: stock alerts failed")
	}
	if err := RunStaleReservationsCleanup(db); err != nil {
		log.Error().Err(err).Msg("cron: stale reservations cleanup failed")
	}
}
