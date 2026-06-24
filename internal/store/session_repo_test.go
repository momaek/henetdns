package store

import (
	"context"
	"testing"
	"time"

	"github.com/momaek/henetdns/internal/model"
)

func newTestRepo(t *testing.T) *SessionRepo {
	t.Helper()
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return NewSessionRepo(st)
}

func TestSessionUpsertGet(t *testing.T) {
	repo := newTestRepo(t)
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
	repo := newTestRepo(t)
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
