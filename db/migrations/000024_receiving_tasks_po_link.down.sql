-- Rollback 000024: Remove purchase_order_id from receiving_tasks.
DROP INDEX IF EXISTS idx_receiving_tasks_po;
ALTER TABLE receiving_tasks DROP COLUMN IF EXISTS purchase_order_id;
