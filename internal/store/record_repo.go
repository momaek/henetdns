package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wentx/henetdns/internal/errs"
	"github.com/wentx/henetdns/internal/model"
)

type RecordRepo struct {
	db *sql.DB
}

func NewRecordRepo(db *sql.DB) *RecordRepo {
	return &RecordRepo{db: db}
}

func (r *RecordRepo) ReplaceAllForZone(ctx context.Context, zoneID string, records []model.Record, syncedAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx records: %w: %w", err, errs.ErrStore)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM records_cache WHERE zone_id = ?`, zoneID); err != nil {
		return fmt.Errorf("clear records cache: %w: %w", err, errs.ErrStore)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO records_cache (zone_id, record_uid, record_id, name, type, value, ttl, priority, dynamic, locked, raw_json, last_synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare record insert: %w: %w", err, errs.ErrStore)
	}
	defer stmt.Close()

	synced := syncedAt.UTC().Format(time.RFC3339Nano)
	for _, record := range records {
		raw, _ := json.Marshal(record)
		var prio any
		if record.Priority != nil {
			prio = *record.Priority
		}
		dynamic := 0
		if record.Dynamic {
			dynamic = 1
		}
		locked := 0
		if record.Locked {
			locked = 1
		}
		if _, err := stmt.ExecContext(
			ctx,
			zoneID,
			record.RecordUID,
			record.RecordID,
			record.Name,
			record.Type,
			record.Value,
			record.TTL,
			prio,
			dynamic,
			locked,
			string(raw),
			synced,
		); err != nil {
			return fmt.Errorf("insert record %s: %w: %w", record.RecordUID, err, errs.ErrStore)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit records tx: %w: %w", err, errs.ErrStore)
	}
	return nil
}
