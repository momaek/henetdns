package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/wentx/henetdns/internal/errs"
	"github.com/wentx/henetdns/internal/httpclient"
	"github.com/wentx/henetdns/internal/model"
	"github.com/wentx/henetdns/internal/store"
)

type Service struct {
	client      *httpclient.Client
	sessionRepo *store.SessionRepo
	auditRepo   *store.AuditRepo
	userAgent   string
}

func NewService(client *httpclient.Client, sessionRepo *store.SessionRepo, auditRepo *store.AuditRepo, userAgent string) *Service {
	return &Service{client: client, sessionRepo: sessionRepo, auditRepo: auditRepo, userAgent: userAgent}
}

func (s *Service) Login(ctx context.Context, username, password string) error {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return fmt.Errorf("username and password are required: %w", errs.ErrInvalidInput)
	}

	if _, err := s.client.Get(ctx, "/", ""); err != nil {
		s.audit("login", nil, "failed", err)
		return err
	}

	form := url.Values{}
	// he.net form field name is "email", but it accepts account username values.
	form.Set("email", username)
	form.Set("pass", password)
	form.Set("submit", "Login!")

	resp, err := s.client.PostForm(ctx, "/", form, s.client.BaseURL().String())
	if err != nil {
		s.audit("login", nil, "failed", err)
		return err
	}

	if IsLoginPage(resp.Body) {
		err = fmt.Errorf("login failed: credentials rejected or additional verification required: %w", errs.ErrAuthRequired)
		s.audit("login", nil, "failed", err)
		return err
	}
	if !IsLoggedInBody(resp.Body) {
		err = fmt.Errorf("login response did not match known success markers; site structure may have changed: %w", errs.ErrParseChanged)
		s.audit("login", nil, "failed", err)
		return err
	}

	cookieJSON, err := SerializeCookieJarForBaseURL(s.client.HTTPClient().Jar, s.client.BaseURL())
	if err != nil {
		s.audit("login", nil, "failed", err)
		return fmt.Errorf("serialize cookies: %w", err)
	}

	now := time.Now().UTC()
	session := model.Session{
		BaseURL:        s.client.BaseURL().String(),
		Username:       username,
		CookieJarJSON:  cookieJSON,
		UserAgent:      s.userAgent,
		LastVerifiedAt: now,
		UpdatedAt:      now,
		CreatedAt:      now,
	}
	if err := s.sessionRepo.Upsert(ctx, session); err != nil {
		s.audit("login", nil, "failed", err)
		return err
	}

	s.audit("login", nil, "ok", nil)
	return nil
}

func (s *Service) EnsureSession(ctx context.Context, username string) error {
	baseURL := s.client.BaseURL().String()
	username = strings.TrimSpace(username)

	var (
		session *model.Session
		err     error
	)
	if username == "" {
		session, err = s.sessionRepo.GetLatestByBaseURL(ctx, baseURL)
	} else {
		session, err = s.sessionRepo.Get(ctx, baseURL, username)
	}
	if err != nil {
		return err
	}
	if session == nil {
		if username == "" {
			return fmt.Errorf("no local session found: run login first: %w", errs.ErrAuthRequired)
		}
		return fmt.Errorf("no local session for %s: run login first: %w", username, errs.ErrAuthRequired)
	}

	if err := RestoreCookieJarForBaseURL(s.client.HTTPClient().Jar, s.client.BaseURL(), session.CookieJarJSON); err != nil {
		return fmt.Errorf("restore cookie jar: %w", err)
	}

	resp, err := s.client.Get(ctx, "/", "")
	if err != nil {
		return err
	}
	if IsLoginPage(resp.Body) || !IsLoggedInBody(resp.Body) {
		return fmt.Errorf("session expired: run login again: %w", errs.ErrAuthRequired)
	}

	cookieJSON, err := SerializeCookieJarForBaseURL(s.client.HTTPClient().Jar, s.client.BaseURL())
	if err != nil {
		return fmt.Errorf("serialize cookies: %w", err)
	}
	now := time.Now().UTC()
	session.CookieJarJSON = cookieJSON
	session.LastVerifiedAt = now
	session.UpdatedAt = now
	if err := s.sessionRepo.Upsert(ctx, *session); err != nil {
		return err
	}
	return nil
}

func IsLoggedInBody(body []byte) bool {
	s := string(body)
	if IsLoginPage(body) {
		return false
	}
	return strings.Contains(s, "id=\"_tlogout\"") ||
		strings.Contains(s, ">Logout<") ||
		strings.Contains(s, "Welcome<br") ||
		strings.Contains(s, "id=\"domains_table\"") ||
		strings.Contains(s, "Active domains for this account")
}

func IsLoginPage(body []byte) bool {
	s := string(body)
	return strings.Contains(s, "Free DNS Login") || strings.Contains(s, "id=\"_loginbutton\"")
}

func SerializeCookieJarForBaseURL(jar http.CookieJar, baseURL *url.URL) (string, error) {
	if jar == nil {
		return "[]", nil
	}
	cookies := jar.Cookies(baseURL)
	payload, err := json.Marshal(cookies)
	if err != nil {
		return "", fmt.Errorf("marshal cookies: %w", err)
	}
	return string(payload), nil
}

func RestoreCookieJarForBaseURL(jar http.CookieJar, baseURL *url.URL, serialized string) error {
	if jar == nil {
		return fmt.Errorf("cookie jar is nil")
	}
	serialized = strings.TrimSpace(serialized)
	if serialized == "" {
		return nil
	}
	var cookies []*http.Cookie
	if err := json.Unmarshal([]byte(serialized), &cookies); err != nil {
		return fmt.Errorf("unmarshal cookies: %w", err)
	}
	jar.SetCookies(baseURL, cookies)
	return nil
}

func NewCookieJar() (http.CookieJar, error) {
	return cookiejar.New(nil)
}

func (s *Service) audit(action string, zoneID *string, status string, err error) {
	if s.auditRepo == nil {
		return
	}
	entry := model.AuditLog{
		Action:             action,
		ZoneID:             zoneID,
		RequestSummaryJSON: "{}",
		ResultStatus:       status,
		CreatedAt:          time.Now().UTC(),
	}
	if err != nil {
		msg := err.Error()
		entry.ErrorMessage = &msg
	}
	_ = s.auditRepo.Insert(context.Background(), entry)
}
