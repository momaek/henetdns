package app

import (
	"fmt"

	"github.com/momaek/henetdns/internal/auth"
	"github.com/momaek/henetdns/internal/config"
	"github.com/momaek/henetdns/internal/henet"
	"github.com/momaek/henetdns/internal/httpclient"
	"github.com/momaek/henetdns/internal/store"
)

type Runtime struct {
	Config      config.Config
	Store       *store.Store
	SessionRepo *store.SessionRepo
	ZoneRepo    *store.ZoneRepo
	RecordRepo  *store.RecordRepo
	Auth        *auth.Service
	HENet       *henet.Service
}

func NewRuntime(cfg config.Config) (*Runtime, error) {
	if err := config.ValidateCommon(cfg); err != nil {
		return nil, err
	}
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	sessionRepo := store.NewSessionRepo(st)
	zoneRepo := store.NewZoneRepo(st)
	recordRepo := store.NewRecordRepo(st)

	jar, err := auth.NewCookieJar()
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("init cookie jar: %w", err)
	}

	const userAgent = "henetdns/0.1"
	client, err := httpclient.New(cfg.BaseURL, cfg.Timeout, jar, userAgent)
	if err != nil {
		st.Close()
		return nil, err
	}

	authService := auth.NewService(client, sessionRepo, userAgent)
	henetService := henet.NewService(client, zoneRepo, recordRepo)

	return &Runtime{
		Config:      cfg,
		Store:       st,
		SessionRepo: sessionRepo,
		ZoneRepo:    zoneRepo,
		RecordRepo:  recordRepo,
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
