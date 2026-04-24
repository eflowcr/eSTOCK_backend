-- Revert M7: restore the original PK (user_id, event_type) without tenant_id.
-- NOTE: this will fail if duplicate (user_id, event_type) rows exist across tenants.
ALTER TABLE notification_preferences DROP CONSTRAINT notification_preferences_pkey;
ALTER TABLE notification_preferences ADD PRIMARY KEY (user_id, event_type);
