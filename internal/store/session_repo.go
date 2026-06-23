package store

import (
	"context"
	"time"

	"github.com/momaek/henetdns/internal/model"
)

// SessionRepo persists login sessions in <dir>/session.json. Sessions are keyed
// by (base_url, username) so multiple accounts can coexist, but in practice a
// CLI user has one.
type SessionRepo struct {
	st *Store
}

func NewSessionRepo(st *Store) *SessionRepo {
	return &SessionRepo{st: st}
}

func (r *SessionRepo) load() ([]model.Session, error) {
	var sessions []model.Session
	if _, err := readJSON(r.st.sessionPath(), &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *SessionRepo) Get(ctx context.Context, baseURL, username string) (*model.Session, error) {
	sessions, err := r.load()
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		if sessions[i].BaseURL == baseURL && sessions[i].Username == username {
			s := sessions[i]
			return &s, nil
		}
	}
	return nil, nil
}

func (r *SessionRepo) GetLatestByBaseURL(ctx context.Context, baseURL string) (*model.Session, error) {
	sessions, err := r.load()
	if err != nil {
		return nil, err
	}
	var latest *model.Session
	for i := range sessions {
		if sessions[i].BaseURL != baseURL {
			continue
		}
		if latest == nil || sessions[i].UpdatedAt.After(latest.UpdatedAt) {
			s := sessions[i]
			latest = &s
		}
	}
	return latest, nil
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

	r.st.mu.Lock()
	defer r.st.mu.Unlock()

	sessions, err := r.load()
	if err != nil {
		return err
	}
	replaced := false
	for i := range sessions {
		if sessions[i].BaseURL == s.BaseURL && sessions[i].Username == s.Username {
			s.CreatedAt = sessions[i].CreatedAt
			sessions[i] = s
			replaced = true
			break
		}
	}
	if !replaced {
		sessions = append(sessions, s)
	}
	return writeJSON(r.st.sessionPath(), sessions)
}
