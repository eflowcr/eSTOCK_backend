package ports

import (
	"context"
	"encoding/json"
)

// AuditLogRepository defines persistence for audit logs (create, list, count).
type AuditLogRepository interface {
	Create(ctx context.Context, params CreateAuditLogParams) error
	List(ctx context.Context, params ListAuditLogsParams) ([]AuditLogEntry, error)
	Count(ctx context.Context, params ListAuditLogsParams) (int64, error)
}

// CreateAuditLogParams is the input for creating one audit log row.
type CreateAuditLogParams struct {
	UserID       *string
	Action       string
	ResourceType string
	ResourceID   string
	OldValue     json.RawMessage
	NewValue     json.RawMessage
	IPAddress    string
	UserAgent    string
	Metadata     json.RawMessage
}

// ListAuditLogsParams holds filters and pagination for listing audit logs.
type ListAuditLogsParams struct {
	Limit           int32
	Offset          int32
	FilterUserID       *string
	FilterResourceType *string
	FilterResourceID   *string
	FilterAction    *string
	FilterStartDate *string // RFC3339
	FilterEndDate   *string // RFC3339
}

// AuditLogEntry is a single audit log row for API responses.
type AuditLogEntry struct {
	ID           string          `json:"id"`
	UserID       *string         `json:"user_id,omitempty"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	OldValue     json.RawMessage `json:"old_value,omitempty"`
	NewValue     json.RawMessage `json:"new_value,omitempty"`
	IPAddress    *string         `json:"ip_address,omitempty"`
	UserAgent    *string         `json:"user_agent,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	CreatedAt    string          `json:"created_at"`
}
