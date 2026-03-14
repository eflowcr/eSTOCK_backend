-- Adjustment reason codes: determine add (inbound) vs subtract (outbound) for stock adjustments. Two fixed system codes + optional custom.
CREATE TABLE IF NOT EXISTS public.adjustment_reason_codes (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    code VARCHAR(80) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    direction VARCHAR(20) NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    is_system BOOLEAN NOT NULL DEFAULT false,
    display_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS adjustment_reason_codes_code_idx ON public.adjustment_reason_codes(code);
CREATE INDEX IF NOT EXISTS adjustment_reason_codes_direction_idx ON public.adjustment_reason_codes(direction);
CREATE INDEX IF NOT EXISTS adjustment_reason_codes_is_active_idx ON public.adjustment_reason_codes(is_active);
CREATE INDEX IF NOT EXISTS adjustment_reason_codes_display_order_idx ON public.adjustment_reason_codes(display_order);

COMMENT ON TABLE public.adjustment_reason_codes IS 'Reason codes for stock adjustments; direction determines add (inbound) or subtract (outbound).';

-- Seed two fixed system codes
INSERT INTO public.adjustment_reason_codes (id, code, name, direction, is_system, display_order, is_active) VALUES
    (nanoid(), 'outbound_physical_withdrawal', 'Outbound due to physical withdrawal', 'outbound', true, 10, true),
    (nanoid(), 'inbound_physical_withdrawal', 'Inbound due to physical withdrawal', 'inbound', true, 20, true)
ON CONFLICT (code) DO NOTHING;
