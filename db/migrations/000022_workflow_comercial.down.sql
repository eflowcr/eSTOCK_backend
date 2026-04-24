-- ============================================================
-- Sprint S3 — Workflow comercial DOWN migration
-- Drop in reverse FK dependency order
-- ============================================================

-- Backorders (ref sales_orders, articles, picking_tasks)
DROP TABLE IF EXISTS backorders;

-- Delivery note items (ref delivery_notes, articles)
DROP TABLE IF EXISTS delivery_note_items;

-- Delivery notes (ref sales_orders, picking_tasks, clients)
DROP TABLE IF EXISTS delivery_notes;

-- Sales order items (ref sales_orders, articles)
DROP TABLE IF EXISTS sales_order_items;

-- Sales orders (ref clients, users, picking_tasks)
DROP TABLE IF EXISTS sales_orders;

-- Purchase order items (ref purchase_orders, articles)
DROP TABLE IF EXISTS purchase_order_items;

-- Purchase orders (ref clients, users, receiving_tasks)
DROP TABLE IF EXISTS purchase_orders;

-- Article suppliers M-N (ref articles, clients)
DROP TABLE IF EXISTS article_suppliers;
