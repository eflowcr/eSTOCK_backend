-- =============================================================================
-- eSTOCK schema (baseline from app structure)
-- =============================================================================

CREATE TABLE IF NOT EXISTS public.sessions (
    sid varchar NOT NULL,
    sess jsonb NOT NULL,
    expire timestamp NOT NULL,
    PRIMARY KEY (sid)
);

CREATE TABLE IF NOT EXISTS public.users (
    id varchar NOT NULL,
    email varchar,
    first_name varchar,
    last_name varchar,
    profile_image_url varchar,
    password varchar,
    role varchar NOT NULL DEFAULT 'operator',
    is_active bool NOT NULL DEFAULT true,
    auth_provider varchar DEFAULT 'local',
    reset_token varchar,
    reset_token_expires timestamp,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS articles_id_seq;
CREATE TABLE IF NOT EXISTS public.articles (
    id int4 NOT NULL DEFAULT nextval('articles_id_seq'::regclass),
    sku varchar NOT NULL,
    name varchar NOT NULL,
    description text,
    unit_price numeric(10,2),
    presentation varchar NOT NULL DEFAULT 'unit',
    track_by_lot bool NOT NULL DEFAULT false,
    track_by_serial bool NOT NULL DEFAULT false,
    track_expiration bool NOT NULL DEFAULT false,
    min_quantity int4 DEFAULT 10,
    max_quantity int4 DEFAULT 1000,
    image_url varchar,
    is_active bool DEFAULT true,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS inventory_id_seq;
CREATE TABLE IF NOT EXISTS public.inventory (
    id int4 NOT NULL DEFAULT nextval('inventory_id_seq'::regclass),
    sku varchar NOT NULL,
    name varchar NOT NULL,
    description text,
    location varchar NOT NULL,
    quantity numeric(10,3) NOT NULL DEFAULT 0,
    status varchar NOT NULL DEFAULT 'available',
    presentation varchar NOT NULL DEFAULT 'unit',
    unit_price numeric(10,2),
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS public.presentations (
    presentation_id varchar(6) NOT NULL,
    description varchar(25),
    PRIMARY KEY (presentation_id)
);

CREATE SEQUENCE IF NOT EXISTS adjustments_id_seq;
CREATE TABLE IF NOT EXISTS public.adjustments (
    id int4 NOT NULL DEFAULT nextval('adjustments_id_seq'::regclass),
    sku varchar NOT NULL,
    location varchar NOT NULL,
    previous_quantity int4 NOT NULL,
    adjustment_quantity int4 NOT NULL,
    new_quantity int4 NOT NULL,
    reason varchar NOT NULL,
    notes text,
    user_id varchar NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS locations_id_seq;
CREATE TABLE IF NOT EXISTS public.locations (
    id int4 NOT NULL DEFAULT nextval('locations_id_seq'::regclass),
    location_code varchar(50) NOT NULL,
    description varchar(255),
    zone varchar(100),
    type varchar(20) NOT NULL DEFAULT 'shelf',
    is_active bool NOT NULL DEFAULT true,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS stock_alerts_id_seq;
CREATE TABLE IF NOT EXISTS public.stock_alerts (
    id int4 NOT NULL DEFAULT nextval('stock_alerts_id_seq'::regclass),
    sku varchar(50) NOT NULL,
    alert_type varchar(20) NOT NULL,
    current_stock int4 NOT NULL,
    recommended_stock int4 NOT NULL,
    alert_level varchar(10) NOT NULL,
    predicted_stock_out_days int4,
    message text NOT NULL,
    is_resolved bool NOT NULL DEFAULT false,
    lot_number varchar(50),
    expiration_date timestamp,
    days_to_expiration int4,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    resolved_at timestamp,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS user_stats_id_seq;
CREATE TABLE IF NOT EXISTS public.user_stats (
    id int4 NOT NULL DEFAULT nextval('user_stats_id_seq'::regclass),
    user_id varchar NOT NULL,
    receiving_tasks_completed int4 DEFAULT 0,
    picking_tasks_completed int4 DEFAULT 0,
    avg_pick_time int4 DEFAULT 0,
    pick_accuracy int4 DEFAULT 100,
    total_picking_time int4 DEFAULT 0,
    correct_picks int4 DEFAULT 0,
    total_picks int4 DEFAULT 0,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS badges_id_seq;
CREATE TABLE IF NOT EXISTS public.badges (
    id int4 NOT NULL DEFAULT nextval('badges_id_seq'::regclass),
    name varchar NOT NULL,
    description text NOT NULL,
    emoji varchar NOT NULL,
    rule_type varchar NOT NULL,
    criteria jsonb NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS user_badges_id_seq;
CREATE TABLE IF NOT EXISTS public.user_badges (
    id int4 NOT NULL DEFAULT nextval('user_badges_id_seq'::regclass),
    user_id varchar NOT NULL,
    badge_id int4 NOT NULL,
    awarded_at timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS inventory_lots_id_seq;
CREATE TABLE IF NOT EXISTS public.inventory_lots (
    id int4 NOT NULL DEFAULT nextval('inventory_lots_id_seq'::regclass),
    inventory_id int4 NOT NULL,
    lot_id int4 NOT NULL,
    quantity numeric(10,3) NOT NULL DEFAULT 0,
    location varchar NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS inventory_serials_id_seq;
CREATE TABLE IF NOT EXISTS public.inventory_serials (
    id int4 NOT NULL DEFAULT nextval('inventory_serials_id_seq'::regclass),
    inventory_id int4 NOT NULL,
    serial_id int4 NOT NULL,
    location varchar NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS lots_id_seq;
CREATE TABLE IF NOT EXISTS public.lots (
    id int4 NOT NULL DEFAULT nextval('lots_id_seq'::regclass),
    lot_number varchar NOT NULL,
    sku varchar NOT NULL,
    quantity numeric(10,3) NOT NULL DEFAULT 0,
    expiration_date timestamp,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    status varchar(10) NOT NULL DEFAULT 'pending',
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS receiving_tasks_id_seq;
CREATE TABLE IF NOT EXISTS public.receiving_tasks (
    id int4 NOT NULL DEFAULT nextval('receiving_tasks_id_seq'::regclass),
    task_id varchar NOT NULL,
    inbound_number varchar NOT NULL,
    created_by varchar NOT NULL,
    assigned_to varchar,
    status varchar NOT NULL DEFAULT 'open',
    priority varchar NOT NULL DEFAULT 'normal',
    notes text,
    items jsonb NOT NULL DEFAULT '[]',
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    completed_at timestamp,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS serials_id_seq;
CREATE TABLE IF NOT EXISTS public.serials (
    id int4 NOT NULL DEFAULT nextval('serials_id_seq'::regclass),
    serial_number varchar NOT NULL,
    sku varchar NOT NULL,
    status varchar NOT NULL DEFAULT 'available',
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS inventory_movements_id_seq;
CREATE TABLE IF NOT EXISTS public.inventory_movements (
    id int4 NOT NULL DEFAULT nextval('inventory_movements_id_seq'::regclass),
    sku varchar(50) NOT NULL,
    location varchar NOT NULL,
    movement_type varchar(20) NOT NULL,
    quantity numeric(10,3) NOT NULL DEFAULT 0,
    remaining_stock numeric(10,3) NOT NULL DEFAULT 0,
    reason varchar(100),
    created_by varchar(50) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE SEQUENCE IF NOT EXISTS picking_tasks_id_seq;
CREATE TABLE IF NOT EXISTS public.picking_tasks (
    id int4 NOT NULL DEFAULT nextval('picking_tasks_id_seq'::regclass),
    task_id varchar NOT NULL,
    order_number varchar NOT NULL,
    created_by varchar NOT NULL,
    assigned_to varchar,
    status varchar NOT NULL DEFAULT 'open',
    priority varchar NOT NULL DEFAULT 'normal',
    notes text,
    items jsonb NOT NULL DEFAULT '[]',
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    completed_at timestamp,
    PRIMARY KEY (id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_session_expire ON public.sessions (expire);
CREATE UNIQUE INDEX IF NOT EXISTS users_email_key ON public.users (email);
CREATE UNIQUE INDEX IF NOT EXISTS articles_sku_key ON public.articles (sku);
CREATE INDEX IF NOT EXISTS sku_location_idx ON public.inventory (sku, location);
CREATE UNIQUE INDEX IF NOT EXISTS pk_presentations ON public.presentations (presentation_id);
CREATE UNIQUE INDEX IF NOT EXISTS locations_location_code_key ON public.locations (location_code);
CREATE UNIQUE INDEX IF NOT EXISTS user_stats_user_id_key ON public.user_stats (user_id);
CREATE INDEX IF NOT EXISTS idx_user_badges_user_id ON public.user_badges (user_id);
CREATE INDEX IF NOT EXISTS idx_user_badges_badge_id ON public.user_badges (badge_id);
CREATE UNIQUE INDEX IF NOT EXISTS receiving_tasks_task_id_key ON public.receiving_tasks (task_id);
CREATE UNIQUE INDEX IF NOT EXISTS picking_tasks_task_id_key ON public.picking_tasks (task_id);

-- Foreign keys
ALTER TABLE public.inventory ADD CONSTRAINT inventory_sku_fkey
    FOREIGN KEY (sku) REFERENCES public.articles(sku) ON DELETE CASCADE;
ALTER TABLE public.user_badges ADD CONSTRAINT user_badges_badge_id_fkey
    FOREIGN KEY (badge_id) REFERENCES public.badges(id);
