BEGIN;

-- Revertir columnas existentes (orden inverso al up)
ALTER TABLE adjustments DROP COLUMN IF EXISTS adjustment_type;

ALTER TABLE inventory_movements
  DROP COLUMN IF EXISTS user_id,
  DROP COLUMN IF EXISTS after_qty,
  DROP COLUMN IF EXISTS before_qty,
  DROP COLUMN IF EXISTS unit_cost,
  DROP COLUMN IF EXISTS serial_id,
  DROP COLUMN IF EXISTS lot_id,
  DROP COLUMN IF EXISTS reference_id,
  DROP COLUMN IF EXISTS reference_type;

ALTER TABLE picking_tasks DROP COLUMN IF EXISTS customer_id;

ALTER TABLE receiving_tasks
  DROP COLUMN IF EXISTS incoterms,
  DROP COLUMN IF EXISTS reception_method,
  DROP COLUMN IF EXISTS tracking_number,
  DROP COLUMN IF EXISTS vendor_ref,
  DROP COLUMN IF EXISTS supplier_id;

-- receiving_task_items are JSONB in receiving_tasks.items — no DB column to drop.

ALTER TABLE lots
  DROP COLUMN IF EXISTS best_before_date,
  DROP COLUMN IF EXISTS manufactured_at,
  DROP COLUMN IF EXISTS lot_notes;

ALTER TABLE articles
  DROP COLUMN IF EXISTS shipping_notes,
  DROP COLUMN IF EXISTS receiving_notes,
  DROP COLUMN IF EXISTS default_location_id,
  DROP COLUMN IF EXISTS min_order_qty,
  DROP COLUMN IF EXISTS serial_number_series,
  DROP COLUMN IF EXISTS batch_number_series,
  DROP COLUMN IF EXISTS safety_stock,
  DROP COLUMN IF EXISTS shelf_life_in_days,
  DROP COLUMN IF EXISTS category_id;

-- Drop tablas nuevas (hojas primero)
DROP TABLE IF EXISTS stock_settings;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS clients;

COMMIT;
