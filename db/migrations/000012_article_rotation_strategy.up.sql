-- Add WMS rotation strategy to articles master: FIFO (First In, First Out) or FEFO (First Expiry, First Out).
-- Used for picking/receiving suggestions and traceability; changes are audited via existing audit log.

ALTER TABLE public.articles
ADD COLUMN IF NOT EXISTS rotation_strategy VARCHAR(20) NOT NULL DEFAULT 'fifo';

ALTER TABLE public.articles
ADD CONSTRAINT articles_rotation_strategy_check
CHECK (rotation_strategy IN ('fifo', 'fefo'));

COMMENT ON COLUMN public.articles.rotation_strategy IS 'WMS rotation rule: fifo = oldest first, fefo = earliest expiry first (requires track_expiration)';
