package app

import (
	"fmt"

	"github.com/wentx/henetdns/internal/auth"
	"github.com/wentx/henetdns/internal/config"
	"github.com/wentx/henetdns/internal/henet"
	"github.com/wentx/henetdns/internal/httpclient"
	"github.com/wentx/henetdns/internal/store"
)

type Runtime struct {
	Config      config.Config
	Store       *store.DB
	SessionRepo *store.SessionRepo
	ZoneRepo    *store.ZoneRepo
	RecordRepo  *store.RecordRepo
	AuditRepo   *store.AuditRepo
	Auth        *auth.Service
	HENet       *henet.Service
}

func NewRuntime(cfg config.Config) (*Runtime, error) {
	if err := config.ValidateCommon(cfg); err != nil {
		return nil, err
	}
	db, err := store.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	sessionRepo := store.NewSessionRepo(db.SQL())
	zoneRepo := store.NewZoneRepo(db.SQL())
	recordRepo := store.NewRecordRepo(db.SQL())
	auditRepo := store.NewAuditRepo(db.SQL())

	jar, err := auth.NewCookieJar()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("init cookie jar: %w", err)
	}

	const userAgent = "henetdns/0.1"
	client, err := httpclient.New(cfg.BaseURL, cfg.Timeout, jar, userAgent)
	if err != nil {
		db.Close()
		return nil, err
	}

	authService := auth.NewService(client, sessionRepo, auditRepo, userAgent)
	henetService := henet.NewService(client, zoneRepo, recordRepo, auditRepo)

	return &Runtime{
		Config:      cfg,
		Store:       db,
		SessionRepo: sessionRepo,
		ZoneRepo:    zoneRepo,
		RecordRepo:  recordRepo,
		AuditRepo:   auditRepo,
		Auth:        authService,
		HENet:       henetService,
	}, nil
}

func (r *Runtime) Close() error {
	if r == nil || r.Store == nil {
		return nil
	}
	return r.Store.Close()
}

func WithRuntime(cfg config.Config, fn func(*Runtime) error) error {
	rt, err := NewRuntime(cfg)
	if err != nil {
		return err
	}
	defer rt.Close()
	return fn(rt)
}
