-- Audit logs: who did what, when, how
-- Schema: db/migrations (000003_audit_logs_schema)

-- name: CreateAuditLog :one
INSERT INTO audit_logs (
    user_id, action, resource_type, resource_id,
    old_value, new_value, ip_address, user_agent, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, user_id, action, resource_type, resource_id,
          old_value, new_value, ip_address, user_agent, metadata, created_at;

-- name: ListAuditLogs :many
SELECT id, user_id, action, resource_type, resource_id,
       old_value, new_value, ip_address, user_agent, metadata, created_at
FROM audit_logs
WHERE
    (sqlc.narg('filter_user_id')::text IS NULL OR user_id = sqlc.narg('filter_user_id'))
    AND (sqlc.narg('filter_resource_type')::text IS NULL OR resource_type = sqlc.narg('filter_resource_type'))
    AND (sqlc.narg('filter_resource_id')::text IS NULL OR resource_id = sqlc.narg('filter_resource_id'))
    AND (sqlc.narg('filter_action')::text IS NULL OR action = sqlc.narg('filter_action'))
    AND (sqlc.narg('filter_start_date')::timestamptz IS NULL OR created_at >= sqlc.narg('filter_start_date'))
    AND (sqlc.narg('filter_end_date')::timestamptz IS NULL OR created_at <= sqlc.narg('filter_end_date'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs
WHERE
    (sqlc.narg('filter_user_id')::text IS NULL OR user_id = sqlc.narg('filter_user_id'))
    AND (sqlc.narg('filter_resource_type')::text IS NULL OR resource_type = sqlc.narg('filter_resource_type'))
    AND (sqlc.narg('filter_resource_id')::text IS NULL OR resource_id = sqlc.narg('filter_resource_id'))
    AND (sqlc.narg('filter_action')::text IS NULL OR action = sqlc.narg('filter_action'))
    AND (sqlc.narg('filter_start_date')::timestamptz IS NULL OR created_at >= sqlc.narg('filter_start_date'))
    AND (sqlc.narg('filter_end_date')::timestamptz IS NULL OR created_at <= sqlc.narg('filter_end_date'));
