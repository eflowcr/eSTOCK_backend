-- Seed system reason codes used by the inventory-counts Submit pipeline.
-- The counts service routes adjustments through AdjustmentsService.CreateAdjustment(Tx),
-- which validates the reason code and flips the sign for outbound directions.
-- Without these rows, every cycle-count submit would fail with "invalid or
-- inactive reason code". See W0 hostile review N1-1.
--
-- Schema reference: db/migrations/000011_adjustment_reason_codes_schema.up.sql
-- Columns: id, code, name, direction, is_system, display_order, is_active.
INSERT INTO public.adjustment_reason_codes (id, code, name, direction, is_system, display_order, is_active) VALUES
    (nanoid(), 'INVENTORY_COUNT_INBOUND',  'Inventory Count Adjustment (Inbound)',  'inbound',  true, 30, true),
    (nanoid(), 'INVENTORY_COUNT_OUTBOUND', 'Inventory Count Adjustment (Outbound)', 'outbound', true, 40, true)
ON CONFLICT (code) DO NOTHING;
