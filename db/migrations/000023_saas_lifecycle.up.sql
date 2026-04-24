-- ============================================================
-- Sprint S3 — SaaS lifecycle: tenants table real, subscriptions,
-- trial state, signup tokens, demo data tracking
-- Migration 000023
-- ============================================================
-- TODO(ARCH1 — S3 deuda): purchase_orders, sales_orders, delivery_notes, backorders
-- all have tenant_id UUID NOT NULL but NO FK to tenants(id) because 000022 runs before
-- this migration creates the tenants table. To add FK enforcement, a follow-up migration
-- should run ALTER TABLE ... ADD CONSTRAINT fk_*_tenant FOREIGN KEY (tenant_id)
-- REFERENCES tenants(id) for each commercial table. Until then, tenant isolation is
-- enforced at application level via Config.TenantID. Tracked as S3 architectural debt.
-- ============================================================

-- Tabla tenants (antes solo era UUID en otras tablas)
CREATE TABLE tenants (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name            TEXT NOT NULL,
  slug            TEXT NOT NULL UNIQUE,                 -- subdomain key
  email           TEXT NOT NULL,                        -- admin contact
  status          TEXT NOT NULL DEFAULT 'trial' CHECK (status IN ('trial','active','past_due','cancelled','suspended')),
  signup_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  trial_started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  trial_ends_at   TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '14 days'),
  is_active       BOOLEAN NOT NULL DEFAULT true,
  metadata        JSONB DEFAULT '{}',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMPTZ
);
CREATE INDEX idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_trial_ends ON tenants(trial_ends_at) WHERE status = 'trial';

-- Backfill: el tenant default existente (de configuración env) se inserta como first row
INSERT INTO tenants (id, name, slug, email, status, trial_ends_at, is_active)
VALUES (
  '00000000-0000-0000-0000-000000000001'::uuid,
  'Default Tenant',
  'default',
  'admin@eprac.com',
  'active',
  NOW() + INTERVAL '100 years',  -- never expires (legacy single-tenant)
  true
)
ON CONFLICT (id) DO NOTHING;

-- Subscriptions (Stripe-backed)
CREATE TABLE subscriptions (
  id                       TEXT PRIMARY KEY DEFAULT nanoid(),
  tenant_id                UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  stripe_subscription_id   TEXT UNIQUE,
  stripe_customer_id       TEXT,
  plan                     TEXT NOT NULL CHECK (plan IN ('trial','starter','pro','enterprise')),
  status                   TEXT NOT NULL CHECK (status IN ('active','past_due','cancelled','incomplete','trialing')),
  current_period_start     TIMESTAMPTZ,
  current_period_end       TIMESTAMPTZ,
  cancel_at_period_end     BOOLEAN NOT NULL DEFAULT false,
  trial_end                TIMESTAMPTZ,
  metadata                 JSONB DEFAULT '{}',
  created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_subscriptions_tenant ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
-- Prevent duplicate "live" subscriptions per tenant (Stripe can have cancelled/incomplete coexisting)
CREATE UNIQUE INDEX uq_subscriptions_tenant_active
  ON subscriptions(tenant_id)
  WHERE status IN ('active','trialing','past_due');

-- Signup tokens (for email verification on self-service signup)
CREATE TABLE signup_tokens (
  id              TEXT PRIMARY KEY DEFAULT nanoid(),
  email           TEXT NOT NULL,
  tenant_name     TEXT NOT NULL,                        -- pending tenant name
  tenant_slug     TEXT NOT NULL,
  token           TEXT NOT NULL UNIQUE,                 -- crypto-random 32 bytes hex
  expires_at      TIMESTAMPTZ NOT NULL,
  used_at         TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Partial index on active tokens (token UNIQUE constraint handles general lookups;
-- this partial index speeds up verification queries: WHERE token=$1 AND expires_at>NOW() AND used_at IS NULL)
CREATE INDEX idx_signup_tokens_active_token ON signup_tokens(token, expires_at) WHERE used_at IS NULL;
CREATE INDEX idx_signup_tokens_email ON signup_tokens(email);

-- Demo data seeds — tracks what was seeded per tenant
CREATE TABLE demo_data_seeds (
  id            TEXT PRIMARY KEY DEFAULT nanoid(),
  tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  seed_name     TEXT NOT NULL,                          -- 'farma-50skus', 'retail-20skus', etc.
  seeded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  metadata      JSONB DEFAULT '{}',                     -- counts: articles, locations, tasks
  UNIQUE(tenant_id, seed_name)
);
