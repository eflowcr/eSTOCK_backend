-- Migration 000027: Add admin_name and admin_password_encrypted to signup_tokens.
-- These fields store the pending admin data so VerifySignup can create the user
-- without re-prompting for the password.
-- admin_password_encrypted holds the password already encrypted with Argon2+AES
-- so it can be stored safely at rest in the pending row.

ALTER TABLE signup_tokens
  ADD COLUMN IF NOT EXISTS admin_name             TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS admin_password_enc     TEXT NOT NULL DEFAULT '';
