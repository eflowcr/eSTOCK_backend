-- =============================================================================
-- 000004 — Roles, users & sessions schema (consolidated)
-- =============================================================================
-- Replaces 000004–000012. Requires 000001 (nanoid), 000002 (estock base), 000003 (audit_logs).
-- Outcome: roles (nanoid id; name is identifier), session_types, token-based sessions, users identity-only.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. Drop dependent objects (FKs then tables; children first)
-- -----------------------------------------------------------------------------
ALTER TABLE IF EXISTS public.audit_logs DROP CONSTRAINT IF EXISTS audit_logs_user_id_fkey;
ALTER TABLE IF EXISTS public.inventory DROP CONSTRAINT IF EXISTS inventory_sku_fkey;
ALTER TABLE IF EXISTS public.user_badges DROP CONSTRAINT IF EXISTS user_badges_badge_id_fkey;
ALTER TABLE IF EXISTS public.inventory_lots DROP CONSTRAINT IF EXISTS inventory_lots_inventory_id_fkey;
ALTER TABLE IF EXISTS public.inventory_lots DROP CONSTRAINT IF EXISTS inventory_lots_lot_id_fkey;
ALTER TABLE IF EXISTS public.inventory_serials DROP CONSTRAINT IF EXISTS inventory_serials_inventory_id_fkey;
ALTER TABLE IF EXISTS public.inventory_serials DROP CONSTRAINT IF EXISTS inventory_serials_serial_id_fkey;
ALTER TABLE IF EXISTS public.users DROP CONSTRAINT IF EXISTS users_role_id_fkey;

DROP TABLE IF EXISTS public.audit_logs;
DROP TABLE IF EXISTS public.inventory_lots;
DROP TABLE IF EXISTS public.inventory_serials;
DROP TABLE IF EXISTS public.user_badges;
DROP TABLE IF EXISTS public.inventory;
DROP TABLE IF EXISTS public.badges;
DROP TABLE IF EXISTS public.lots;
DROP TABLE IF EXISTS public.serials;
DROP TABLE IF EXISTS public.articles;
DROP TABLE IF EXISTS public.adjustments;
DROP TABLE IF EXISTS public.locations;
DROP TABLE IF EXISTS public.stock_alerts;
DROP TABLE IF EXISTS public.user_stats;
DROP TABLE IF EXISTS public.receiving_tasks;
DROP TABLE IF EXISTS public.picking_tasks;
DROP TABLE IF EXISTS public.inventory_movements;
DROP TABLE IF EXISTS public.presentations;
DROP TABLE IF EXISTS public.users;
DROP TABLE IF EXISTS public.sessions;

DROP SEQUENCE IF EXISTS articles_id_seq;
DROP SEQUENCE IF EXISTS inventory_id_seq;
DROP SEQUENCE IF EXISTS adjustments_id_seq;
DROP SEQUENCE IF EXISTS locations_id_seq;
DROP SEQUENCE IF EXISTS stock_alerts_id_seq;
DROP SEQUENCE IF EXISTS user_stats_id_seq;
DROP SEQUENCE IF EXISTS badges_id_seq;
DROP SEQUENCE IF EXISTS user_badges_id_seq;
DROP SEQUENCE IF EXISTS inventory_lots_id_seq;
DROP SEQUENCE IF EXISTS inventory_serials_id_seq;
DROP SEQUENCE IF EXISTS lots_id_seq;
DROP SEQUENCE IF EXISTS receiving_tasks_id_seq;
DROP SEQUENCE IF EXISTS serials_id_seq;
DROP SEQUENCE IF EXISTS inventory_movements_id_seq;
DROP SEQUENCE IF EXISTS picking_tasks_id_seq;

-- -----------------------------------------------------------------------------
-- 2. Helper functions
-- -----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION public.set_updated_by_from_session()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_by = COALESCE(
        current_setting('app.current_user_id', true),
        NEW.updated_by
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION public.set_default_user_role()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.role_id IS NULL OR NEW.role_id = '' THEN
        SELECT id INTO NEW.role_id FROM public.roles WHERE LOWER(name) = 'operator' LIMIT 1;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -----------------------------------------------------------------------------
-- 3. Roles (nanoid id; name is the stable identifier)
-- -----------------------------------------------------------------------------
CREATE TABLE public.roles (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    name TEXT NOT NULL,
    description TEXT,
    permissions JSONB NOT NULL DEFAULT '{}',
    is_active bool NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
COMMENT ON TABLE public.roles IS 'RBAC roles; name is the stable identifier (Admin, Operator, Viewer).';

INSERT INTO public.roles (id, name, description, permissions) VALUES
    (nanoid(), 'Admin', 'Full access', '{"all": true}'),
    (nanoid(), 'Operator', 'Create, read, update for articles, lots, locations, serials', '{"articles": {"create": true, "read": true, "update": true, "delete": true}, "lots": {"create": true, "read": true, "update": true, "delete": true}, "locations": {"create": true, "read": true, "update": true, "delete": true}, "serials": {"create": true, "read": true, "update": true, "delete": true}, "inventory": {"create": true, "read": true, "update": true, "delete": true}}'),
    (nanoid(), 'Viewer', 'Read-only', '{"articles": {"read": true}, "lots": {"read": true}, "locations": {"read": true}, "serials": {"read": true}, "inventory": {"read": true}}');

-- -----------------------------------------------------------------------------
-- 4. Session types (id default nanoid; seed with fixed ids for API)
-- -----------------------------------------------------------------------------
CREATE TABLE public.session_types (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    name varchar NOT NULL,
    description text,
    duration_minutes integer NOT NULL DEFAULT 60,
    is_active boolean DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_by TEXT,
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ DEFAULT NULL
);
COMMENT ON TABLE public.session_types IS 'Session types for different session configurations';
COMMENT ON COLUMN public.session_types.id IS 'Unique identifier for the session type';
COMMENT ON COLUMN public.session_types.name IS 'Name of the session type (must be unique)';
COMMENT ON COLUMN public.session_types.duration_minutes IS 'Default duration for this session type in minutes';

CREATE UNIQUE INDEX session_types_name_key ON public.session_types(name) WHERE deleted_at IS NULL;
CREATE INDEX session_types_deleted_at_idx ON public.session_types(deleted_at);
CREATE INDEX session_types_created_at_idx ON public.session_types(created_at);
CREATE INDEX session_types_updated_at_idx ON public.session_types(updated_at);
CREATE INDEX session_types_is_active_idx ON public.session_types(is_active);
CREATE INDEX session_types_duration_idx ON public.session_types(duration_minutes);

CREATE TRIGGER set_session_types_updated_by
    BEFORE INSERT OR UPDATE ON public.session_types
    FOR EACH ROW EXECUTE PROCEDURE public.set_updated_by_from_session();

INSERT INTO public.session_types (id, name, description, duration_minutes, is_active) VALUES
    ('web_session', 'Web Session', 'Standard web browser session', 60, true),
    ('mobile_session', 'Mobile Session', 'Mobile application session', 1440, true),
    ('api_session', 'API Session', 'API access session', 30, true),
    ('admin_session', 'Admin Session', 'Extended administrative session', 480, true);

-- -----------------------------------------------------------------------------
-- 5. Users (identity only; no tokens — sessions table owns tokens)
-- -----------------------------------------------------------------------------
CREATE TABLE public.users (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    name varchar NOT NULL DEFAULT '',
    email varchar,
    first_name varchar,
    last_name varchar,
    profile_image_url varchar,
    password varchar,
    role_id TEXT NOT NULL REFERENCES public.roles(id),
    is_active bool NOT NULL DEFAULT true,
    email_verified boolean DEFAULT false,
    email_verified_at TIMESTAMPTZ DEFAULT NULL,
    updated_by TEXT,
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);
COMMENT ON TABLE public.users IS 'User identity and account; session and token data are in sessions table.';
COMMENT ON COLUMN public.users.name IS 'Full name of the user (display)';
COMMENT ON COLUMN public.users.email_verified IS 'Whether the email has been verified';
COMMENT ON COLUMN public.users.email_verified_at IS 'When the email was verified';
COMMENT ON COLUMN public.users.updated_by IS 'User id who last updated (set via app.current_user_id)';
COMMENT ON COLUMN public.users.deleted_at IS 'Soft delete; NULL = active';

CREATE UNIQUE INDEX users_email_key ON public.users(email) WHERE deleted_at IS NULL AND email IS NOT NULL;
CREATE INDEX users_deleted_at_idx ON public.users(deleted_at);
CREATE INDEX users_email_verified_idx ON public.users(email_verified);
CREATE INDEX users_created_at_idx ON public.users(created_at);
CREATE INDEX users_updated_at_idx ON public.users(updated_at);
CREATE INDEX users_is_active_idx ON public.users(is_active);
CREATE INDEX users_role_id_idx ON public.users(role_id);

CREATE TRIGGER set_users_updated_by
    BEFORE INSERT OR UPDATE ON public.users
    FOR EACH ROW EXECUTE PROCEDURE public.set_updated_by_from_session();

CREATE TRIGGER users_set_default_role
    BEFORE INSERT ON public.users
    FOR EACH ROW EXECUTE PROCEDURE public.set_default_user_role();

-- -----------------------------------------------------------------------------
-- 6. Sessions (token-based only; no legacy sid/sess/expire)
-- -----------------------------------------------------------------------------
CREATE TABLE public.sessions (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    user_id TEXT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    session_type_id TEXT NOT NULL REFERENCES public.session_types(id) ON DELETE RESTRICT,
    token_hash TEXT NOT NULL,
    refresh_token_hash TEXT,
    user_agent TEXT,
    client_ip TEXT,
    ip_address INET,
    device_info JSONB,
    is_active boolean DEFAULT true,
    expires_at TIMESTAMPTZ NOT NULL,
    last_activity_at TIMESTAMPTZ DEFAULT now(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_by TEXT,
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ DEFAULT NULL
);
COMMENT ON TABLE public.sessions IS 'Sessions table for managing user authentication sessions';
COMMENT ON COLUMN public.sessions.token_hash IS 'Hash of access token';
COMMENT ON COLUMN public.sessions.refresh_token_hash IS 'Hash of refresh token';
COMMENT ON COLUMN public.sessions.expires_at IS 'Session expiry';
COMMENT ON COLUMN public.sessions.ip_address IS 'IP address of the client (INET type)';

CREATE UNIQUE INDEX sessions_refresh_token_hash_key ON public.sessions(refresh_token_hash) WHERE deleted_at IS NULL AND refresh_token_hash IS NOT NULL;
CREATE INDEX sessions_user_id_idx ON public.sessions(user_id);
CREATE INDEX sessions_session_type_id_idx ON public.sessions(session_type_id);
CREATE INDEX sessions_expires_at_idx ON public.sessions(expires_at);
CREATE INDEX sessions_is_active_idx ON public.sessions(is_active);
CREATE INDEX sessions_deleted_at_idx ON public.sessions(deleted_at);
CREATE INDEX sessions_created_at_idx ON public.sessions(created_at);
CREATE INDEX sessions_updated_at_idx ON public.sessions(updated_at);
CREATE INDEX sessions_client_ip_idx ON public.sessions(client_ip);

CREATE TRIGGER set_sessions_updated_by
    BEFORE INSERT OR UPDATE ON public.sessions
    FOR EACH ROW EXECUTE PROCEDURE public.set_updated_by_from_session();

-- -----------------------------------------------------------------------------
-- 7. Presentations (pk = code)
-- -----------------------------------------------------------------------------
CREATE TABLE public.presentations (
    presentation_id varchar(6) NOT NULL PRIMARY KEY,
    description varchar(25)
);
CREATE UNIQUE INDEX IF NOT EXISTS pk_presentations ON public.presentations (presentation_id);

-- -----------------------------------------------------------------------------
-- 8. Articles
-- -----------------------------------------------------------------------------
CREATE TABLE public.articles (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
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
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS articles_sku_key ON public.articles (sku);

-- -----------------------------------------------------------------------------
-- 9. Inventory
-- -----------------------------------------------------------------------------
CREATE TABLE public.inventory (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    sku varchar NOT NULL,
    name varchar NOT NULL,
    description text,
    location varchar NOT NULL,
    quantity numeric(10,3) NOT NULL DEFAULT 0,
    status varchar NOT NULL DEFAULT 'available',
    presentation varchar NOT NULL DEFAULT 'unit',
    unit_price numeric(10,2),
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS sku_location_idx ON public.inventory (sku, location);
ALTER TABLE public.inventory ADD CONSTRAINT inventory_sku_fkey
    FOREIGN KEY (sku) REFERENCES public.articles(sku) ON DELETE CASCADE;

-- -----------------------------------------------------------------------------
-- 10. Adjustments, locations, stock_alerts, user_stats, badges, user_badges
-- -----------------------------------------------------------------------------
CREATE TABLE public.adjustments (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    sku varchar NOT NULL,
    location varchar NOT NULL,
    previous_quantity int4 NOT NULL,
    adjustment_quantity int4 NOT NULL,
    new_quantity int4 NOT NULL,
    reason varchar NOT NULL,
    notes text,
    user_id TEXT NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE public.locations (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    location_code varchar(50) NOT NULL,
    description varchar(255),
    zone varchar(100),
    type varchar(20) NOT NULL DEFAULT 'shelf',
    is_active bool NOT NULL DEFAULT true,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS locations_location_code_key ON public.locations (location_code);

CREATE TABLE public.stock_alerts (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
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
    resolved_at timestamp
);

CREATE TABLE public.user_stats (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    user_id TEXT NOT NULL,
    receiving_tasks_completed int4 DEFAULT 0,
    picking_tasks_completed int4 DEFAULT 0,
    avg_pick_time int4 DEFAULT 0,
    pick_accuracy int4 DEFAULT 100,
    total_picking_time int4 DEFAULT 0,
    correct_picks int4 DEFAULT 0,
    total_picks int4 DEFAULT 0,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS user_stats_user_id_key ON public.user_stats (user_id);

CREATE TABLE public.badges (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    name varchar NOT NULL,
    description text NOT NULL,
    emoji varchar NOT NULL,
    rule_type varchar NOT NULL,
    criteria jsonb NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE public.user_badges (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    user_id TEXT NOT NULL,
    badge_id TEXT NOT NULL REFERENCES public.badges(id),
    awarded_at timestamp DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_user_badges_user_id ON public.user_badges (user_id);
CREATE INDEX IF NOT EXISTS idx_user_badges_badge_id ON public.user_badges (badge_id);

-- -----------------------------------------------------------------------------
-- 11. Lots, serials, inventory_lots, inventory_serials
-- -----------------------------------------------------------------------------
CREATE TABLE public.lots (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    lot_number varchar NOT NULL,
    sku varchar NOT NULL,
    quantity numeric(10,3) NOT NULL DEFAULT 0,
    expiration_date timestamp,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    status varchar(10) NOT NULL DEFAULT 'pending'
);

CREATE TABLE public.inventory_lots (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    inventory_id TEXT NOT NULL REFERENCES public.inventory(id),
    lot_id TEXT NOT NULL REFERENCES public.lots(id),
    quantity numeric(10,3) NOT NULL DEFAULT 0,
    location varchar NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE public.serials (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    serial_number varchar NOT NULL,
    sku varchar NOT NULL,
    status varchar NOT NULL DEFAULT 'available',
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE public.inventory_serials (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    inventory_id TEXT NOT NULL REFERENCES public.inventory(id),
    serial_id TEXT NOT NULL REFERENCES public.serials(id),
    location varchar NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);

-- -----------------------------------------------------------------------------
-- 12. Receiving_tasks, picking_tasks, inventory_movements
-- -----------------------------------------------------------------------------
CREATE TABLE public.receiving_tasks (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    task_id varchar NOT NULL,
    inbound_number varchar NOT NULL,
    created_by TEXT NOT NULL,
    assigned_to TEXT,
    status varchar NOT NULL DEFAULT 'open',
    priority varchar NOT NULL DEFAULT 'normal',
    notes text,
    items jsonb NOT NULL DEFAULT '[]',
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    completed_at timestamp
);
CREATE UNIQUE INDEX IF NOT EXISTS receiving_tasks_task_id_key ON public.receiving_tasks (task_id);

CREATE TABLE public.picking_tasks (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    task_id varchar NOT NULL,
    order_number varchar NOT NULL,
    created_by TEXT NOT NULL,
    assigned_to TEXT,
    status varchar NOT NULL DEFAULT 'open',
    priority varchar NOT NULL DEFAULT 'normal',
    notes text,
    items jsonb NOT NULL DEFAULT '[]',
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    completed_at timestamp
);
CREATE UNIQUE INDEX IF NOT EXISTS picking_tasks_task_id_key ON public.picking_tasks (task_id);

CREATE TABLE public.inventory_movements (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    sku varchar(50) NOT NULL,
    location varchar NOT NULL,
    movement_type varchar(20) NOT NULL,
    quantity numeric(10,3) NOT NULL DEFAULT 0,
    remaining_stock numeric(10,3) NOT NULL DEFAULT 0,
    reason varchar(100),
    created_by TEXT NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);

-- -----------------------------------------------------------------------------
-- 13. Audit logs
-- -----------------------------------------------------------------------------
CREATE TABLE public.audit_logs (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    user_id TEXT REFERENCES public.users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL DEFAULT '',
    old_value JSONB,
    new_value JSONB,
    ip_address TEXT,
    user_agent TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON public.audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON public.audit_logs(created_at DESC);
