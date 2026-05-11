-- Idempotency-Key dedup table for mobile S7.2 write-side outbox.
-- Mobile generates a UUID v4 per mutation before queueing into its local
-- outbox. The same key is replayed on every retry — backend deduplicates
-- by (key, user_id) and returns the cached response for hits.
--
-- TTL window: 7 days. After that the key entry expires and a replay would
-- re-apply the mutation. Operators shouldn't be queuing mutations for more
-- than a week offline; cron sweeps expired rows.
--
-- Schema depends on: users(id) (TEXT id from 000004).

CREATE TABLE IF NOT EXISTS public.idempotency_keys (
    key             TEXT NOT NULL,
    user_id         TEXT NOT NULL REFERENCES public.users(id),
    endpoint        VARCHAR(255) NOT NULL,
    method          VARCHAR(10) NOT NULL,
    response_status INTEGER NOT NULL,
    response_body   JSONB NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at      TIMESTAMP NOT NULL,
    CONSTRAINT pk_idempotency_keys PRIMARY KEY (key, user_id),
    CONSTRAINT chk_idempotency_keys_method CHECK (method IN ('PATCH', 'POST'))
);

-- Hot path: middleware lookup `WHERE key = ? AND user_id = ?` (covered by PK).
-- Secondary path: cron cleanup `WHERE expires_at < NOW()`.
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expires_at
    ON public.idempotency_keys(expires_at);

-- Optional ops query: "how many cached responses per endpoint" for capacity sizing.
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_endpoint
    ON public.idempotency_keys(endpoint);
