package model

import "time"

type Session struct {
	BaseURL        string
	Username       string
	CookieJarJSON  string
	UserAgent      string
	LastVerifiedAt time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Zone struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	LastSyncedAt time.Time `json:"last_synced_at,omitempty"`
}

type Record struct {
	ZoneID       string    `json:"zone_id"`
	RecordID     string    `json:"record_id"`
	RecordUID    string    `json:"record_uid"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	TTL          int       `json:"ttl"`
	Priority     *int      `json:"priority,omitempty"`
	Value        string    `json:"value"`
	Dynamic      bool      `json:"dynamic"`
	Locked       bool      `json:"locked"`
	LastSyncedAt time.Time `json:"last_synced_at,omitempty"`
}

type AuditLog struct {
	Action             string
	ZoneID             *string
	RequestSummaryJSON string
	ResultStatus       string
	ErrorMessage       *string
	CreatedAt          time.Time
}
