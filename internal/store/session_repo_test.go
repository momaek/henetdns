package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/wentx/henetdns/internal/model"
)

func TestSessionUpsertGet(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewSessionRepo(db.SQL())
	now := time.Now().UTC()
	s := model.Session{
		BaseURL:        "https://dns.he.net",
		Username:       "testuser",
		CookieJarJSON:  `[{"Name":"sid","Value":"abc"}]`,
		UserAgent:      "ua",
		LastVerifiedAt: now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repo.Upsert(context.Background(), s); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := repo.Get(context.Background(), s.BaseURL, s.Username)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil || got.CookieJarJSON != s.CookieJarJSON {
		t.Fatalf("unexpected session: %+v", got)
	}
}

func TestSessionGetLatestByBaseURL(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test-latest.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewSessionRepo(db.SQL())
	now := time.Now().UTC()
	s1 := model.Session{
		BaseURL:        "https://dns.he.net",
		Username:       "userA",
		CookieJarJSON:  `[{"Name":"sid","Value":"a"}]`,
		UserAgent:      "ua",
		LastVerifiedAt: now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s2 := model.Session{
		BaseURL:        "https://dns.he.net",
		Username:       "userB",
		CookieJarJSON:  `[{"Name":"sid","Value":"b"}]`,
		UserAgent:      "ua",
		LastVerifiedAt: now.Add(2 * time.Second),
		CreatedAt:      now.Add(2 * time.Second),
		UpdatedAt:      now.Add(2 * time.Second),
	}
	if err := repo.Upsert(context.Background(), s1); err != nil {
		t.Fatalf("upsert s1: %v", err)
	}
	if err := repo.Upsert(context.Background(), s2); err != nil {
		t.Fatalf("upsert s2: %v", err)
	}

	got, err := repo.GetLatestByBaseURL(context.Background(), "https://dns.he.net")
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if got == nil || got.Username != "userB" {
		t.Fatalf("unexpected latest session: %+v", got)
	}
}
