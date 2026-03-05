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

type ZoneRepo struct {
	db *sql.DB
}

func NewZoneRepo(db *sql.DB) *ZoneRepo {
	return &ZoneRepo{db: db}
}

func (r *ZoneRepo) ReplaceAll(ctx context.Context, zones []model.Zone, syncedAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx zones: %w: %w", err, errs.ErrStore)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM zones_cache`); err != nil {
		return fmt.Errorf("clear zones cache: %w: %w", err, errs.ErrStore)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO zones_cache (zone_id, zone_name, raw_json, last_synced_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare zone insert: %w: %w", err, errs.ErrStore)
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	synced := syncedAt.UTC().Format(time.RFC3339Nano)
	for _, zone := range zones {
		raw, _ := json.Marshal(zone)
		if _, err := stmt.ExecContext(ctx, zone.ID, zone.Name, string(raw), synced, now); err != nil {
			return fmt.Errorf("insert zone %s: %w: %w", zone.ID, err, errs.ErrStore)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit zones tx: %w: %w", err, errs.ErrStore)
	}
	return nil
}
