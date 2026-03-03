package repositories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/jackc/pgx/v5/pgtype"
)

// AuditLogsRepositorySQLC implements ports.AuditLogRepository using sqlc.
type AuditLogsRepositorySQLC struct {
	queries *sqlc.Queries
}

// NewAuditLogsRepositorySQLC returns an audit logs repository backed by sqlc.
func NewAuditLogsRepositorySQLC(queries *sqlc.Queries) *AuditLogsRepositorySQLC {
	return &AuditLogsRepositorySQLC{queries: queries}
}

var _ ports.AuditLogRepository = (*AuditLogsRepositorySQLC)(nil)

func (r *AuditLogsRepositorySQLC) Create(ctx context.Context, p ports.CreateAuditLogParams) error {
	arg := sqlc.CreateAuditLogParams{
		Action:       p.Action,
		ResourceType: p.ResourceType,
		ResourceID:   p.ResourceID,
		OldValue:     p.OldValue,
		NewValue:     p.NewValue,
	}
	if p.UserID != nil {
		arg.UserID = pgtype.Text{String: *p.UserID, Valid: true}
	}
	if p.IPAddress != "" {
		arg.IpAddress = pgtype.Text{String: p.IPAddress, Valid: true}
	}
	if p.UserAgent != "" {
		arg.UserAgent = pgtype.Text{String: p.UserAgent, Valid: true}
	}
	if len(p.Metadata) > 0 {
		arg.Metadata = p.Metadata
	}
	_, err := r.queries.CreateAuditLog(ctx, arg)
	return err
}

func (r *AuditLogsRepositorySQLC) List(ctx context.Context, p ports.ListAuditLogsParams) ([]ports.AuditLogEntry, error) {
	arg := sqlc.ListAuditLogsParams{
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	if p.FilterUserID != nil {
		arg.FilterUserID = pgtype.Text{String: *p.FilterUserID, Valid: true}
	}
	if p.FilterResourceType != nil {
		arg.FilterResourceType = pgtype.Text{String: *p.FilterResourceType, Valid: true}
	}
	if p.FilterResourceID != nil {
		arg.FilterResourceID = pgtype.Text{String: *p.FilterResourceID, Valid: true}
	}
	if p.FilterAction != nil {
		arg.FilterAction = pgtype.Text{String: *p.FilterAction, Valid: true}
	}
	if p.FilterStartDate != nil {
		t, err := time.Parse(time.RFC3339, *p.FilterStartDate)
		if err == nil {
			arg.FilterStartDate = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}
	if p.FilterEndDate != nil {
		t, err := time.Parse(time.RFC3339, *p.FilterEndDate)
		if err == nil {
			arg.FilterEndDate = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}
	list, err := r.queries.ListAuditLogs(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]ports.AuditLogEntry, len(list))
	for i, row := range list {
		out[i] = sqlcAuditLogToEntry(row)
	}
	return out, nil
}

func (r *AuditLogsRepositorySQLC) Count(ctx context.Context, p ports.ListAuditLogsParams) (int64, error) {
	arg := sqlc.CountAuditLogsParams{}
	if p.FilterUserID != nil {
		arg.FilterUserID = pgtype.Text{String: *p.FilterUserID, Valid: true}
	}
	if p.FilterResourceType != nil {
		arg.FilterResourceType = pgtype.Text{String: *p.FilterResourceType, Valid: true}
	}
	if p.FilterResourceID != nil {
		arg.FilterResourceID = pgtype.Text{String: *p.FilterResourceID, Valid: true}
	}
	if p.FilterAction != nil {
		arg.FilterAction = pgtype.Text{String: *p.FilterAction, Valid: true}
	}
	if p.FilterStartDate != nil {
		t, err := time.Parse(time.RFC3339, *p.FilterStartDate)
		if err == nil {
			arg.FilterStartDate = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}
	if p.FilterEndDate != nil {
		t, err := time.Parse(time.RFC3339, *p.FilterEndDate)
		if err == nil {
			arg.FilterEndDate = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}
	return r.queries.CountAuditLogs(ctx, arg)
}

func sqlcAuditLogToEntry(row sqlc.AuditLog) ports.AuditLogEntry {
	e := ports.AuditLogEntry{
		ID:           row.ID,
		Action:       row.Action,
		ResourceType: row.ResourceType,
		ResourceID:   row.ResourceID,
		OldValue:     json.RawMessage(row.OldValue),
		NewValue:     json.RawMessage(row.NewValue),
		CreatedAt:    row.CreatedAt.Format(time.RFC3339),
	}
	if row.UserID.Valid {
		e.UserID = &row.UserID.String
	}
	if row.IpAddress.Valid {
		e.IPAddress = &row.IpAddress.String
	}
	if row.UserAgent.Valid {
		e.UserAgent = &row.UserAgent.String
	}
	if len(row.Metadata) > 0 {
		e.Metadata = json.RawMessage(row.Metadata)
	}
	return e
}
