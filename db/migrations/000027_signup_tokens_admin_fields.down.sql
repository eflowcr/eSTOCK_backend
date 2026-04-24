ALTER TABLE signup_tokens
  DROP COLUMN IF EXISTS admin_name,
  DROP COLUMN IF EXISTS admin_password_enc;
