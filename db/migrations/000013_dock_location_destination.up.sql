-- Dock location (destination only): optional dock/loading bay at destination for stock transfers.
ALTER TABLE public.stock_transfers
ADD COLUMN IF NOT EXISTS dock_location VARCHAR(100) NULL;

COMMENT ON COLUMN public.stock_transfers.dock_location IS 'Optional dock or loading bay at destination for receiving (WMS).';
