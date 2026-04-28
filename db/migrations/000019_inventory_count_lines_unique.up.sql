-- Idempotency guarantee for inventory_count_lines: two operators (or one operator
-- on a flaky network retrying a scan) hitting POST /scan-line for the same
-- (count, location, sku, lot, serial) tuple must converge to a single row, not
-- create duplicates that would double-count variance at submit time.
--
-- The repository's AddLine pairs this index with an ON CONFLICT DO UPDATE
-- (last-scan-wins). See W0 hostile review N1-3.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_count_line_unique_key
    ON public.inventory_count_lines (count_id, location_id, sku, COALESCE(lot, ''), COALESCE(serial, ''));
