package tools

import (
	"context"
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

// TrialEmailSender is the minimal interface RunTrialExpirationCheck needs from
// a notifications service. Using an interface avoids an import cycle between
// tools and services packages.
type TrialEmailSender interface {
	SendTrialEmail(ctx context.Context, toEmail, tenantName, templateType string, daysLeft int) error
}

// trialTenantRow is the raw DB row selected for each active trial tenant.
type trialTenantRow struct {
	ID          string    `gorm:"column:id"`
	Name        string    `gorm:"column:name"`
	Email       string    `gorm:"column:email"`
	TrialEndsAt time.Time `gorm:"column:trial_ends_at"`
}

// RunTrialExpirationCheck inspects all tenants in 'trial' status and:
//   - Sends a reminder email at exactly 13, 11, or 7 days remaining.
//   - Marks the tenant as past_due + deactivates it when trial has expired (daysLeft <= 0).
//
// Advisory lock 987654323 (transaction-scoped) ensures only one pod fires per tick.
// Email sends are fire-and-forget via sendFn — a failure never blocks the cron.
// sendFn signature: (ctx, toEmail, tenantName, templateType, daysLeft) error
// templateType is one of: "trial_reminder_7d", "trial_reminder_11d", "trial_reminder_13d", "trial_expired".
func RunTrialExpirationCheck(db *gorm.DB, sendFn func(ctx context.Context, toEmail, tenantName, templateType string, daysLeft int) error) error {
	if db == nil {
		return errors.New("cron: nil db")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// Advisory lock (transaction-scoped) — released automatically on commit/rollback.
		var locked bool
		if err := tx.Raw("SELECT pg_try_advisory_xact_lock(987654323)").Scan(&locked).Error; err != nil {
			return err
		}
		if !locked {
			log.Debug().Msg("trial_expiration: otro pod tiene el lock, skipping")
			return nil
		}

		var tenants []trialTenantRow
		if err := tx.Raw(`
			SELECT id, name, email, trial_ends_at
			  FROM tenants
			 WHERE status = 'trial'
			   AND deleted_at IS NULL
		`).Scan(&tenants).Error; err != nil {
			return fmt.Errorf("trial_expiration: query tenants: %w", err)
		}

		now := time.Now().UTC()
		ctx := context.Background()

		for _, t := range tenants {
			endsAt := t.TrialEndsAt.UTC()
			// TODO(CS1 — S3.5): integer truncation means a tenant who signed up at 23:59 with
			// trial_ends_at 14d later at 23:59 will show daysLeft=6 at midnight on day 7 (only
			// 23h elapsed). Use math.Round or add +0.5 before int() to fix off-by-one on reminder day.
			daysLeft := int(endsAt.Sub(now).Hours() / 24)

			switch {
			case daysLeft <= 0:
				// Trial expired — mark tenant past_due and deactivate.
				if err := tx.Exec(`
					UPDATE tenants
					   SET status     = 'past_due',
					       is_active  = false,
					       updated_at = NOW()
					 WHERE id = ? AND status = 'trial'
				`, t.ID).Error; err != nil {
					log.Error().Err(err).Str("tenant_id", t.ID).Msg("cron: trial_expiration: failed to deactivate tenant")
					continue
				}
				log.Info().Str("tenant_id", t.ID).Str("tenant", t.Name).Msg("cron: trial expired — tenant deactivated")

				if sendFn != nil {
					go func(email, name string) {
						if err := sendFn(ctx, email, name, "trial_expired", 0); err != nil {
							log.Warn().Err(err).Str("email", email).Msg("cron: trial_expired email failed")
						}
					}(t.Email, t.Name)
				}

			case daysLeft == 7, daysLeft == 11, daysLeft == 13:
				templateType := fmt.Sprintf("trial_reminder_%dd", daysLeft)

				// M1 fix: per-day dedup — skip if a reminder with this event_type was already
				// sent to this tenant in the last 23 hours (cron fires hourly, so without this
				// a tenant could receive up to 24 reminder emails in a single day).
				var lastSentCount int64
				if err := tx.Raw(`
					SELECT COUNT(*) FROM notifications
					 WHERE tenant_id = ? AND event_type = ? AND created_at > NOW() - INTERVAL '23 hours'
				`, t.ID, templateType).Scan(&lastSentCount).Error; err != nil {
					log.Warn().Err(err).Str("tenant_id", t.ID).Str("event_type", templateType).
						Msg("cron: trial_reminder dedup check failed — skipping to be safe")
					continue
				}
				if lastSentCount > 0 {
					log.Debug().Str("tenant_id", t.ID).Str("template", templateType).
						Msg("cron: trial reminder already sent today — skipping")
					continue
				}

				log.Info().Str("tenant_id", t.ID).Str("tenant", t.Name).Int("days_left", daysLeft).
					Str("template", templateType).Msg("cron: sending trial reminder")

				// Insert notification record inside tx to mark as sent (dedup gate).
				// title/body are informational only — the actual content is in the email.
				notifID := fmt.Sprintf("trial-%s-%s", t.ID[:8], templateType)
				if err := tx.Exec(`
					INSERT INTO notifications (id, tenant_id, user_id, event_type, title, channels, created_at)
					VALUES (?, ?, '', ?, ?, 'email', NOW())
					ON CONFLICT DO NOTHING
				`, notifID, t.ID, templateType,
					fmt.Sprintf("Trial reminder: %d días restantes", daysLeft),
				).Error; err != nil {
					log.Warn().Err(err).Str("tenant_id", t.ID).Str("template", templateType).
						Msg("cron: failed to insert trial reminder notification record — skipping email to avoid duplicates")
					continue
				}

				if sendFn != nil {
					capturedEmail := t.Email
					capturedName := t.Name
					capturedDays := daysLeft
					capturedTemplate := templateType
					go func() {
						if err := sendFn(ctx, capturedEmail, capturedName, capturedTemplate, capturedDays); err != nil {
							log.Warn().Err(err).Str("email", capturedEmail).Str("template", capturedTemplate).
								Msg("cron: trial reminder email failed")
						}
					}()
				}

			default:
				// Nothing to do for other day counts.
			}
		}

		log.Info().Int("tenants_checked", len(tenants)).Msg("cron: trial_expiration check completed")
		return nil
	})
}

// CronDispatch ejecuta todos los jobs del cron en secuencia.
// Se invoca: una vez al arrancar (tras delay de estabilización) y luego cada hora por el ticker.
// Los errores se loggean sin parar la ejecución del siguiente job.
//
// Callbacks (all optional, pass nil to skip):
//   - lotNotifyFn: called per expiring lot event (eventType, title, body)
//   - lowStockNotifyFn: called per unresolved low-stock alert (sku, message)
//   - trialSendFn: called per trial tenant requiring a reminder or expiration email
func CronDispatch(db *gorm.DB, analyzer func() error, lotNotifyFn func(eventType, title, body string) error, lowStockNotifyFn func(sku, message string) error, trialSendFn func(ctx context.Context, toEmail, tenantName, templateType string, daysLeft int) error) {
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
	// S3-W5-C: trial lifecycle reminders + expiration.
	if err := RunTrialExpirationCheck(db, trialSendFn); err != nil {
		log.Error().Err(err).Msg("cron: trial expiration check failed")
	}
}

