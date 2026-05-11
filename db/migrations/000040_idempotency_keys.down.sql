DROP INDEX IF EXISTS public.idx_idempotency_keys_endpoint;
DROP INDEX IF EXISTS public.idx_idempotency_keys_expires_at;
DROP TABLE IF EXISTS public.idempotency_keys;
