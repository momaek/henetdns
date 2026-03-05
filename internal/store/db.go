package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/wentx/henetdns/internal/errs"
)

type DB struct {
	sql *sql.DB
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w: %w", err, errs.ErrStore)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w: %w", err, errs.ErrStore)
	}
	conn := &DB{sql: db}
	if err := conn.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return conn, nil
}

func (d *DB) SQL() *sql.DB {
	return d.sql
}

func (d *DB) Close() error {
	if d == nil || d.sql == nil {
		return nil
	}
	return d.sql.Close()
}

func (d *DB) migrate(ctx context.Context) error {
	if _, err := d.sql.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("migrate schema: %w: %w", err, errs.ErrStore)
	}
	return nil
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  base_url TEXT NOT NULL,
  email TEXT NOT NULL,
  cookie_jar_json TEXT NOT NULL,
  user_agent TEXT,
  last_verified_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(base_url, email)
);

CREATE TABLE IF NOT EXISTS zones_cache (
  zone_id TEXT PRIMARY KEY,
  zone_name TEXT NOT NULL,
  raw_json TEXT NOT NULL,
  last_synced_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS records_cache (
  zone_id TEXT NOT NULL,
  record_uid TEXT NOT NULL,
  record_id TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  value TEXT NOT NULL,
  ttl INTEGER,
  priority INTEGER,
  dynamic INTEGER NOT NULL,
  locked INTEGER NOT NULL,
  raw_json TEXT NOT NULL,
  last_synced_at TEXT NOT NULL,
  PRIMARY KEY(zone_id, record_uid)
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  action TEXT NOT NULL,
  zone_id TEXT,
  request_summary_json TEXT NOT NULL,
  result_status TEXT NOT NULL,
  error_message TEXT,
  created_at TEXT NOT NULL
);
`
