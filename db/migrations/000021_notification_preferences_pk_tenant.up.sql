-- M7: Extend notification_preferences PK to include tenant_id so that different
-- tenants sharing the same user_id + event_type can have independent preferences.
-- The original PK (user_id, event_type) would collapse preferences across tenants.

ALTER TABLE notification_preferences DROP CONSTRAINT notification_preferences_pkey;
ALTER TABLE notification_preferences ADD PRIMARY KEY (user_id, event_type, tenant_id);
