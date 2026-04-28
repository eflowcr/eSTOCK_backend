-- 000036_signup_tokens_seed_demo_data.down.sql
-- Reverses 000036_signup_tokens_seed_demo_data.up.sql.

ALTER TABLE signup_tokens DROP COLUMN IF EXISTS seed_demo_data;
