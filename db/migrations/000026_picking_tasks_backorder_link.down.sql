-- Rollback 000026
DROP INDEX IF EXISTS idx_picking_tasks_backorder;
ALTER TABLE picking_tasks DROP COLUMN IF EXISTS source_backorder_id;
