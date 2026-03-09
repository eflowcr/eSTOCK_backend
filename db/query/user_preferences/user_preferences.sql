-- User preferences: get, update, get-or-create (from backend_template)

-- name: GetUserPreferences :one
SELECT id, user_id, theme, language, email_notifications, push_notifications,
       marketing_notifications, profile_visibility, data_sharing,
       created_at, updated_by, updated_at
FROM user_preferences
WHERE user_id = $1;

-- name: UpdateUserPreferences :one
UPDATE user_preferences SET
  theme = $2,
  language = $3,
  email_notifications = $4,
  push_notifications = $5,
  marketing_notifications = $6,
  profile_visibility = $7,
  data_sharing = $8
WHERE user_id = $1
RETURNING id, user_id, theme, language, email_notifications, push_notifications,
          marketing_notifications, profile_visibility, data_sharing,
          created_at, updated_by, updated_at;

-- name: GetOrCreateUserPreferences :one
INSERT INTO user_preferences (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING id, user_id, theme, language, email_notifications, push_notifications,
          marketing_notifications, profile_visibility, data_sharing,
          created_at, updated_by, updated_at;
