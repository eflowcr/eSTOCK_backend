BEGIN;

-- ============================================================
-- TABLAS NUEVAS
-- ============================================================

-- clients (supplier / customer / both)
CREATE TABLE clients (
  id VARCHAR(40) PRIMARY KEY,
  tenant_id UUID NOT NULL,
  type VARCHAR(20) NOT NULL CHECK (type IN ('supplier','customer','both')),
  code VARCHAR(50) NOT NULL,
  name VARCHAR(200) NOT NULL,
  email VARCHAR(150),
  phone VARCHAR(50),
  address TEXT,
  tax_id VARCHAR(50),
  notes TEXT,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_by VARCHAR(40),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tenant_id, code)
);
CREATE INDEX idx_clients_tenant_type ON clients(tenant_id, type);
CREATE INDEX idx_clients_active ON clients(tenant_id) WHERE is_active = TRUE;

-- categories (árbol 2 niveles)
CREATE TABLE categories (
  id VARCHAR(40) PRIMARY KEY,
  tenant_id UUID NOT NULL,
  name VARCHAR(150) NOT NULL,
  parent_id VARCHAR(40) REFERENCES categories(id) ON DELETE SET NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tenant_id, name, parent_id)
);
CREATE INDEX idx_categories_tenant ON categories(tenant_id);

-- notifications (in-app + email log)
CREATE TABLE notifications (
  id VARCHAR(40) PRIMARY KEY,
  tenant_id UUID NOT NULL,
  user_id VARCHAR(40) NOT NULL,
  event_type VARCHAR(50) NOT NULL,
  title VARCHAR(200) NOT NULL,
  body TEXT,
  resource_type VARCHAR(50),
  resource_id VARCHAR(80),
  channels VARCHAR(100) NOT NULL DEFAULT 'in_app',
  is_read BOOLEAN NOT NULL DEFAULT FALSE,
  read_at TIMESTAMPTZ,
  sent_email_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notifications_user_unread ON notifications(tenant_id, user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);

-- notification_preferences (PK compuesta)
CREATE TABLE notification_preferences (
  user_id VARCHAR(40) NOT NULL,
  event_type VARCHAR(50) NOT NULL,
  tenant_id UUID NOT NULL,
  in_app BOOLEAN NOT NULL DEFAULT TRUE,
  email BOOLEAN NOT NULL DEFAULT TRUE,
  push BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, event_type)
);
CREATE INDEX idx_notif_prefs_tenant ON notification_preferences(tenant_id);

-- stock_settings por tenant
CREATE TABLE stock_settings (
  tenant_id UUID PRIMARY KEY,
  valuation_method VARCHAR(10) NOT NULL DEFAULT 'avco' CHECK (valuation_method IN ('avco','fifo')),
  pick_batch_based_on VARCHAR(10) NOT NULL DEFAULT 'fefo' CHECK (pick_batch_based_on IN ('fefo','fifo','lifo')),
  over_receipt_allowance_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
  over_delivery_allowance_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
  over_picking_allowance_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
  auto_reserve_stock BOOLEAN NOT NULL DEFAULT TRUE,
  allow_partial_reservation BOOLEAN NOT NULL DEFAULT TRUE,
  expiry_alert_days INTEGER NOT NULL DEFAULT 30,
  auto_create_material_request BOOLEAN NOT NULL DEFAULT FALSE,
  partial_delivery_policy VARCHAR(20) NOT NULL DEFAULT 'immediate' CHECK (partial_delivery_policy IN ('immediate','when_all_ready')),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- COLUMNAS NUEVAS EN TABLAS EXISTENTES
-- ============================================================

-- articles: category + logística + instrucciones
ALTER TABLE articles
  ADD COLUMN category_id VARCHAR(40) REFERENCES categories(id) ON DELETE SET NULL,
  ADD COLUMN shelf_life_in_days INTEGER,
  ADD COLUMN safety_stock NUMERIC(12,4) NOT NULL DEFAULT 0,
  ADD COLUMN batch_number_series VARCHAR(50),
  ADD COLUMN serial_number_series VARCHAR(50),
  ADD COLUMN min_order_qty NUMERIC(12,4) NOT NULL DEFAULT 0,
  ADD COLUMN default_location_id VARCHAR(40) REFERENCES locations(id) ON DELETE SET NULL,
  ADD COLUMN receiving_notes TEXT,
  ADD COLUMN shipping_notes TEXT;
CREATE INDEX idx_articles_category ON articles(category_id) WHERE category_id IS NOT NULL;

-- lots: metadatos fabricación / BBD
ALTER TABLE lots
  ADD COLUMN lot_notes TEXT,
  ADD COLUMN manufactured_at DATE,
  ADD COLUMN best_before_date DATE;

-- NOTE: receiving_task_items are stored as JSONB inside receiving_tasks.items —
-- accepted_qty / rejected_qty are Go struct fields (Wave 3 Track A, bloque R1).

-- receiving_tasks: metadata de proveedor
ALTER TABLE receiving_tasks
  ADD COLUMN supplier_id VARCHAR(40) REFERENCES clients(id) ON DELETE SET NULL,
  ADD COLUMN vendor_ref VARCHAR(100),
  ADD COLUMN tracking_number VARCHAR(100),
  ADD COLUMN reception_method VARCHAR(100),
  ADD COLUMN incoterms VARCHAR(10);
CREATE INDEX idx_receiving_tasks_supplier ON receiving_tasks(supplier_id) WHERE supplier_id IS NOT NULL;

-- picking_tasks: cliente destino
ALTER TABLE picking_tasks
  ADD COLUMN customer_id VARCHAR(40) REFERENCES clients(id) ON DELETE SET NULL;
CREATE INDEX idx_picking_tasks_customer ON picking_tasks(customer_id) WHERE customer_id IS NOT NULL;

-- inventory_movements: trazabilidad extendida
ALTER TABLE inventory_movements
  ADD COLUMN reference_type VARCHAR(30),
  ADD COLUMN reference_id VARCHAR(80),
  ADD COLUMN lot_id VARCHAR(40) REFERENCES lots(id) ON DELETE SET NULL,
  ADD COLUMN serial_id VARCHAR(40) REFERENCES serials(id) ON DELETE SET NULL,
  ADD COLUMN unit_cost NUMERIC(12,4),
  ADD COLUMN before_qty NUMERIC(12,4),
  ADD COLUMN after_qty NUMERIC(12,4),
  ADD COLUMN user_id VARCHAR(40) REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX idx_inv_movements_reference ON inventory_movements(reference_type, reference_id);
CREATE INDEX idx_inv_movements_lot ON inventory_movements(lot_id) WHERE lot_id IS NOT NULL;
CREATE INDEX idx_inv_movements_user ON inventory_movements(user_id) WHERE user_id IS NOT NULL;

-- adjustments: type para diferenciar (deuda S1 D1)
ALTER TABLE adjustments
  ADD COLUMN adjustment_type VARCHAR(20) NOT NULL DEFAULT 'increase' CHECK (adjustment_type IN ('increase','decrease','count_reconcile'));

COMMIT;
