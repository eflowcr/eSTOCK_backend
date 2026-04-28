-- Migration 000028: stripe_webhook_events idempotency table
-- Provides atomic webhook dedup via PRIMARY KEY conflict gate.
-- INSERT ... ON CONFLICT (event_id) DO NOTHING is the idempotency gate in
-- BillingRepository.AttemptMarkWebhookEventProcessed — no separate SELECT needed.

CREATE TABLE stripe_webhook_events (
  event_id     TEXT PRIMARY KEY,
  event_type   TEXT NOT NULL,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  raw_event    JSONB
);

CREATE INDEX idx_stripe_webhook_events_processed_at
  ON stripe_webhook_events(processed_at);
