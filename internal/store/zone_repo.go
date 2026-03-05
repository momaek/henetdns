package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

func (r *ZoneRepo) List(ctx context.Context) ([]model.Zone, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT zone_id, zone_name, last_synced_at
		FROM zones_cache
		ORDER BY lower(zone_name), zone_id
	`)
	if err != nil {
		return nil, fmt.Errorf("query zones cache: %w: %w", err, errs.ErrStore)
	}
	defer rows.Close()

	var zones []model.Zone
	for rows.Next() {
		var (
			id       string
			name     string
			syncedAt string
		)
		if err := rows.Scan(&id, &name, &syncedAt); err != nil {
			return nil, fmt.Errorf("scan zones cache: %w: %w", err, errs.ErrStore)
		}
		zone := model.Zone{ID: id, Name: name}
		if ts, err := parseCacheTime(syncedAt); err == nil {
			zone.LastSyncedAt = ts
		}
		zones = append(zones, zone)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate zones cache: %w: %w", err, errs.ErrStore)
	}
	return zones, nil
}

func (r *ZoneRepo) FindIDByName(ctx context.Context, zoneName string) (string, bool, error) {
	zoneName = strings.TrimSpace(zoneName)
	if zoneName == "" {
		return "", false, nil
	}
	var id string
	err := r.db.QueryRowContext(ctx, `
		SELECT zone_id
		FROM zones_cache
		WHERE lower(zone_name) = lower(?)
		LIMIT 1
	`, zoneName).Scan(&id)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("query zone id by name from cache: %w: %w", err, errs.ErrStore)
	}
	return id, true, nil
}

func parseCacheTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}
