-- S3-W2-B: Revert sales_order_id FK on picking_tasks
DROP INDEX IF EXISTS idx_picking_tasks_so;
ALTER TABLE picking_tasks DROP COLUMN IF EXISTS sales_order_id;
