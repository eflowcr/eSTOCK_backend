-- Migration 000026: Add source_backorder_id to picking_tasks (S3-W3-A BO2)
-- Allows picking tasks generated from a backorder fulfillment to be traced back.
-- If source_backorder_id IS SET, CompletePickingTask will NOT generate another backorder (max depth=1).
ALTER TABLE picking_tasks
  ADD COLUMN IF NOT EXISTS source_backorder_id TEXT REFERENCES backorders(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_picking_tasks_backorder ON picking_tasks(source_backorder_id) WHERE source_backorder_id IS NOT NULL;
