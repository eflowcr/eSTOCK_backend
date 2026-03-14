ALTER TABLE public.articles DROP CONSTRAINT IF EXISTS articles_rotation_strategy_check;
ALTER TABLE public.articles DROP COLUMN IF EXISTS rotation_strategy;
