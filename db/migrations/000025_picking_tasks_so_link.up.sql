-- S3-W2-B: Add sales_order_id FK to picking_tasks for SO auto-link (SO3)
ALTER TABLE picking_tasks
    ADD COLUMN IF NOT EXISTS sales_order_id TEXT REFERENCES sales_orders(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_picking_tasks_so
    ON picking_tasks(sales_order_id)
    WHERE sales_order_id IS NOT NULL;
