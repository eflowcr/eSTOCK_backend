-- ============================================================
-- Sprint S3 — Workflow comercial (Purchase Orders, Sales Orders,
-- Delivery Notes, Backorders, Article-Supplier link)
-- Migration 000022
-- ============================================================

-- M-N: artículo puede tener N proveedores con preferencia + lead time + precio
CREATE TABLE article_suppliers (
  id              TEXT PRIMARY KEY DEFAULT nanoid(),
  tenant_id       UUID NOT NULL,
  article_sku     TEXT NOT NULL REFERENCES articles(sku) ON DELETE CASCADE,
  supplier_id     TEXT NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
  is_preferred    BOOLEAN NOT NULL DEFAULT false,
  lead_time_days  INT,
  unit_cost       NUMERIC(12,4),
  supplier_sku    TEXT,
  notes           TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMPTZ,
  UNIQUE(tenant_id, article_sku, supplier_id)
);
CREATE INDEX idx_article_suppliers_tenant_sku ON article_suppliers(tenant_id, article_sku);
CREATE INDEX idx_article_suppliers_supplier ON article_suppliers(supplier_id);

-- Purchase Orders header
CREATE TABLE purchase_orders (
  id                TEXT PRIMARY KEY DEFAULT nanoid(),
  tenant_id         UUID NOT NULL,
  po_number         TEXT NOT NULL,                     -- e.g., "PO-2026-0001"
  supplier_id       TEXT NOT NULL REFERENCES clients(id),
  status            TEXT NOT NULL DEFAULT 'draft',     -- draft|submitted|partial|completed|cancelled
  expected_date     DATE,
  notes             TEXT,
  created_by        TEXT REFERENCES users(id),
  submitted_at      TIMESTAMPTZ,
  completed_at      TIMESTAMPTZ,
  cancelled_at      TIMESTAMPTZ,
  receiving_task_id TEXT REFERENCES receiving_tasks(id),  -- auto-generated on submit
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at        TIMESTAMPTZ,
  UNIQUE(tenant_id, po_number)
);
CREATE INDEX idx_po_tenant_status ON purchase_orders(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_po_supplier ON purchase_orders(supplier_id);
CREATE INDEX idx_po_receiving_task ON purchase_orders(receiving_task_id);

-- Purchase Order items
CREATE TABLE purchase_order_items (
  id                  TEXT PRIMARY KEY DEFAULT nanoid(),
  purchase_order_id   TEXT NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
  article_sku         TEXT NOT NULL REFERENCES articles(sku),
  expected_qty        NUMERIC(12,3) NOT NULL CHECK (expected_qty > 0),
  received_qty        NUMERIC(12,3) NOT NULL DEFAULT 0,
  rejected_qty        NUMERIC(12,3) NOT NULL DEFAULT 0,
  unit_cost           NUMERIC(12,4),
  discrepancy         NUMERIC(12,3) GENERATED ALWAYS AS (expected_qty - (received_qty + rejected_qty)) STORED,
  notes               TEXT,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(purchase_order_id, article_sku)
);
CREATE INDEX idx_po_items_po ON purchase_order_items(purchase_order_id);
CREATE INDEX idx_po_items_sku ON purchase_order_items(article_sku);

-- Sales Orders header (mirror de PO pero hacia cliente)
CREATE TABLE sales_orders (
  id              TEXT PRIMARY KEY DEFAULT nanoid(),
  tenant_id       UUID NOT NULL,
  so_number       TEXT NOT NULL,                       -- e.g., "SO-2026-0001"
  customer_id     TEXT NOT NULL REFERENCES clients(id),
  status          TEXT NOT NULL DEFAULT 'draft',       -- draft|submitted|partial|completed|cancelled
  expected_date   DATE,
  notes           TEXT,
  created_by      TEXT REFERENCES users(id),
  submitted_at    TIMESTAMPTZ,
  completed_at    TIMESTAMPTZ,
  cancelled_at    TIMESTAMPTZ,
  picking_task_id TEXT REFERENCES picking_tasks(id),    -- auto-generated on submit
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMPTZ,
  UNIQUE(tenant_id, so_number)
);
CREATE INDEX idx_so_tenant_status ON sales_orders(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_so_customer ON sales_orders(customer_id);
CREATE INDEX idx_so_picking_task ON sales_orders(picking_task_id);

-- Sales Order items
CREATE TABLE sales_order_items (
  id              TEXT PRIMARY KEY DEFAULT nanoid(),
  sales_order_id  TEXT NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
  article_sku     TEXT NOT NULL REFERENCES articles(sku),
  expected_qty    NUMERIC(12,3) NOT NULL CHECK (expected_qty > 0),
  picked_qty      NUMERIC(12,3) NOT NULL DEFAULT 0,
  unit_price      NUMERIC(12,4),
  notes           TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(sales_order_id, article_sku)
);
CREATE INDEX idx_so_items_so ON sales_order_items(sales_order_id);
CREATE INDEX idx_so_items_sku ON sales_order_items(article_sku);

-- Delivery Notes (generated when picking is completed against a SO)
CREATE TABLE delivery_notes (
  id              TEXT PRIMARY KEY DEFAULT nanoid(),
  tenant_id       UUID NOT NULL,
  dn_number       TEXT NOT NULL,
  sales_order_id  TEXT NOT NULL REFERENCES sales_orders(id),
  picking_task_id TEXT REFERENCES picking_tasks(id),
  customer_id     TEXT NOT NULL REFERENCES clients(id),
  total_items     INT NOT NULL DEFAULT 0,
  pdf_url         TEXT,                                 -- S3/local storage path
  pdf_generated_at TIMESTAMPTZ,
  delivered_at    TIMESTAMPTZ,
  signed_by       TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(tenant_id, dn_number)
);
CREATE INDEX idx_dn_tenant ON delivery_notes(tenant_id);
CREATE INDEX idx_dn_so ON delivery_notes(sales_order_id);

-- Delivery Note items (snapshot of what was actually delivered)
CREATE TABLE delivery_note_items (
  id                  TEXT PRIMARY KEY DEFAULT nanoid(),
  delivery_note_id    TEXT NOT NULL REFERENCES delivery_notes(id) ON DELETE CASCADE,
  article_sku         TEXT NOT NULL REFERENCES articles(sku),
  qty                 NUMERIC(12,3) NOT NULL,
  lot_numbers         TEXT[],                            -- snapshot at picking time
  notes               TEXT,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_dn_items_dn ON delivery_note_items(delivery_note_id);

-- Backorders (auto-created when picking has insufficient stock)
CREATE TABLE backorders (
  id                       TEXT PRIMARY KEY DEFAULT nanoid(),
  tenant_id                UUID NOT NULL,
  original_sales_order_id  TEXT NOT NULL REFERENCES sales_orders(id),
  article_sku              TEXT NOT NULL REFERENCES articles(sku),
  remaining_qty            NUMERIC(12,3) NOT NULL CHECK (remaining_qty > 0),
  status                   TEXT NOT NULL DEFAULT 'pending',  -- pending|fulfilled|cancelled
  generated_picking_task_id TEXT REFERENCES picking_tasks(id),
  fulfilled_at             TIMESTAMPTZ,
  created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_no_recursive_backorder CHECK (true)  -- enforced at app level: max depth 1
);
CREATE INDEX idx_backorders_tenant_status ON backorders(tenant_id, status);
CREATE INDEX idx_backorders_so ON backorders(original_sales_order_id);
