package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wentx/henetdns/internal/errs"
	"github.com/wentx/henetdns/internal/model"
)

type AuditRepo struct {
	db *sql.DB
}

func NewAuditRepo(db *sql.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

func (r *AuditRepo) Insert(ctx context.Context, log model.AuditLog) error {
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (action, zone_id, request_summary_json, result_status, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, log.Action, log.ZoneID, log.RequestSummaryJSON, log.ResultStatus, log.ErrorMessage, log.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert audit log: %w: %w", err, errs.ErrStore)
	}
	return nil
}
