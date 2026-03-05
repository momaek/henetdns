package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wentx/henetdns/internal/errs"
	"github.com/wentx/henetdns/internal/model"
)

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) Get(ctx context.Context, baseURL, username string) (*model.Session, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT base_url, email, cookie_jar_json, user_agent, last_verified_at, created_at, updated_at
		FROM sessions
		WHERE base_url = ? AND email = ?
	`, baseURL, username)
	return scanSessionRow(row)
}

func (r *SessionRepo) GetLatestByBaseURL(ctx context.Context, baseURL string) (*model.Session, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT base_url, email, cookie_jar_json, user_agent, last_verified_at, created_at, updated_at
		FROM sessions
		WHERE base_url = ?
		ORDER BY updated_at DESC
		LIMIT 1
	`, baseURL)
	return scanSessionRow(row)
}

func scanSessionRow(row *sql.Row) (*model.Session, error) {
	var s model.Session
	var verified, created, updated string
	if err := row.Scan(&s.BaseURL, &s.Username, &s.CookieJarJSON, &s.UserAgent, &verified, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query session: %w: %w", err, errs.ErrStore)
	}
	var err error
	s.LastVerifiedAt, err = time.Parse(time.RFC3339Nano, verified)
	if err != nil {
		return nil, fmt.Errorf("parse last_verified_at: %w: %w", err, errs.ErrStore)
	}
	s.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w: %w", err, errs.ErrStore)
	}
	s.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w: %w", err, errs.ErrStore)
	}
	return &s, nil
}

func (r *SessionRepo) Upsert(ctx context.Context, s model.Session) error {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = s.CreatedAt
	}
	if s.LastVerifiedAt.IsZero() {
		s.LastVerifiedAt = s.UpdatedAt
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (base_url, email, cookie_jar_json, user_agent, last_verified_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(base_url, email) DO UPDATE SET
			cookie_jar_json = excluded.cookie_jar_json,
			user_agent = excluded.user_agent,
			last_verified_at = excluded.last_verified_at,
			updated_at = excluded.updated_at
	`, s.BaseURL, s.Username, s.CookieJarJSON, s.UserAgent, s.LastVerifiedAt.Format(time.RFC3339Nano), s.CreatedAt.Format(time.RFC3339Nano), s.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("upsert session: %w: %w", err, errs.ErrStore)
	}
	return nil
}
