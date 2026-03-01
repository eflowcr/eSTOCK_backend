-- =============================================================================
-- NANOID EXTENSION - ROLLBACK
-- =============================================================================

DROP FUNCTION IF EXISTS nanoid(int);

-- Uncomment to remove pgcrypto (can affect other objects in production)
-- DROP EXTENSION IF EXISTS pgcrypto;
