package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/eflowcr/eSTOCK_backend/ports"
)

// AuditService provides Log and List for audit trail. Log is fire-and-forget with timeout.
type AuditService struct {
	repo ports.AuditLogRepository
}

// NewAuditService returns an AuditService that uses the given repository.
func NewAuditService(repo ports.AuditLogRepository) *AuditService {
	return &AuditService{repo: repo}
}

// Log records an audit event. It runs the insert in a goroutine with a 5s timeout so request latency is not affected.
// Pass nil for userID if unauthenticated; oldValue/newValue can be nil.
func (s *AuditService) Log(ctx context.Context, userID *string, action, resourceType, resourceID string, oldValue, newValue json.RawMessage, ipAddress, userAgent string) {
	go func() {
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		params := ports.CreateAuditLogParams{
			UserID:       userID,
			Action:       action,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			OldValue:     oldValue,
			NewValue:     newValue,
			IPAddress:    ipAddress,
			UserAgent:    userAgent,
		}
		_ = s.repo.Create(timeout, params)
	}()
}

// List returns audit log entries and total count for the given filters and pagination.
func (s *AuditService) List(ctx context.Context, params ports.ListAuditLogsParams) ([]ports.AuditLogEntry, int64, error) {
	entries, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}
