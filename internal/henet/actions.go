package henet

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/wentx/henetdns/internal/errs"
	"github.com/wentx/henetdns/internal/httpclient"
	"github.com/wentx/henetdns/internal/model"
	"github.com/wentx/henetdns/internal/store"
)

type Service struct {
	client     *httpclient.Client
	zoneRepo   *store.ZoneRepo
	recordRepo *store.RecordRepo
	auditRepo  *store.AuditRepo
}

type RecordInput struct {
	Type        string
	Name        string
	Value       string
	TTL         int
	Priority    int
	HasPriority bool
}

func NewService(client *httpclient.Client, zoneRepo *store.ZoneRepo, recordRepo *store.RecordRepo, auditRepo *store.AuditRepo) *Service {
	return &Service{client: client, zoneRepo: zoneRepo, recordRepo: recordRepo, auditRepo: auditRepo}
}

func (s *Service) ListZones(ctx context.Context) ([]model.Zone, error) {
	resp, err := s.client.Get(ctx, "/", "")
	if err != nil {
		return nil, err
	}
	if strings.Contains(string(resp.Body), "Free DNS Login") {
		return nil, fmt.Errorf("login page returned: %w", errs.ErrAuthRequired)
	}
	zones, err := ParseZones(resp.Body)
	if err != nil {
		return nil, err
	}
	if s.zoneRepo != nil {
		_ = s.zoneRepo.ReplaceAll(ctx, zones, time.Now().UTC())
	}
	return zones, nil
}

func (s *Service) ListZonesFromCache(ctx context.Context) ([]model.Zone, error) {
	if s.zoneRepo == nil {
		return nil, nil
	}
	return s.zoneRepo.List(ctx)
}

func (s *Service) ListZonesCachedFirst(ctx context.Context) ([]model.Zone, error) {
	zones, err := s.ListZonesFromCache(ctx)
	if err == nil && len(zones) > 0 {
		return zones, nil
	}
	return s.ListZones(ctx)
}

func (s *Service) ResolveZoneID(ctx context.Context, zoneOrID string) (string, error) {
	zoneOrID = strings.TrimSpace(zoneOrID)
	if zoneOrID == "" {
		return "", fmt.Errorf("zone is required: %w", errs.ErrInvalidInput)
	}
	if isDigits(zoneOrID) {
		return zoneOrID, nil
	}
	zones, err := s.ListZones(ctx)
	if err != nil {
		return "", err
	}
	for _, z := range zones {
		if strings.EqualFold(z.Name, zoneOrID) {
			return z.ID, nil
		}
	}
	return "", fmt.Errorf("zone %q not found: %w", zoneOrID, errs.ErrInvalidInput)
}

func (s *Service) ResolveZoneIDFromCache(ctx context.Context, zoneOrID string) (string, error) {
	zoneOrID = strings.TrimSpace(zoneOrID)
	if zoneOrID == "" {
		return "", fmt.Errorf("zone is required: %w", errs.ErrInvalidInput)
	}
	if isDigits(zoneOrID) {
		return zoneOrID, nil
	}
	if s.zoneRepo == nil {
		return "", fmt.Errorf("zone %q not found: %w", zoneOrID, errs.ErrInvalidInput)
	}
	if id, found, err := s.zoneRepo.FindIDByName(ctx, zoneOrID); err != nil {
		return "", err
	} else if found {
		return id, nil
	}
	return "", fmt.Errorf("zone %q not found in cache: %w", zoneOrID, errs.ErrInvalidInput)
}

func (s *Service) ListRecords(ctx context.Context, zoneID string) ([]model.Record, error) {
	resp, err := s.client.Get(ctx, ZonePagePath(zoneID), s.client.BaseURL().String())
	if err != nil {
		return nil, err
	}
	if strings.Contains(string(resp.Body), "Free DNS Login") {
		return nil, fmt.Errorf("login page returned for zone: %w", errs.ErrAuthRequired)
	}
	records, err := ParseRecords(zoneID, resp.Body)
	if err != nil {
		return nil, err
	}
	if s.recordRepo != nil {
		_ = s.recordRepo.ReplaceAllForZone(ctx, zoneID, records, time.Now().UTC())
	}
	return records, nil
}

func (s *Service) ListRecordsFromCache(ctx context.Context, zoneID string) ([]model.Record, error) {
	if s.recordRepo == nil {
		return nil, nil
	}
	return s.recordRepo.ListByZone(ctx, zoneID)
}

func (s *Service) ListRecordsCachedFirst(ctx context.Context, zoneID string) ([]model.Record, error) {
	records, err := s.ListRecordsFromCache(ctx, zoneID)
	if err == nil && len(records) > 0 {
		return records, nil
	}
	return s.ListRecords(ctx, zoneID)
}

func (s *Service) UpsertRecord(ctx context.Context, zoneID string, in RecordInput) error {
	normalized, err := normalizeRecordInput(in)
	if err != nil {
		return err
	}
	records, err := s.ListRecords(ctx, zoneID)
	if err != nil {
		return err
	}
	if _, found := findExactRecord(records, normalized); found {
		return nil
	}

	form := url.Values{}
	form.Set("menu", "edit_zone")
	form.Set("Type", normalized.Type)
	form.Set("hosted_dns_zoneid", zoneID)
	form.Set("hosted_dns_recordid", "")
	form.Set("hosted_dns_editzone", "1")
	form.Set("Name", normalized.Name)
	form.Set("Content", normalized.Value)
	form.Set("TTL", strconv.Itoa(normalized.TTL))
	if normalized.Type == "MX" {
		if normalized.HasPriority {
			form.Set("Priority", strconv.Itoa(normalized.Priority))
		} else {
			form.Set("Priority", "10")
		}
	} else {
		form.Set("Priority", "")
	}
	form.Set("hosted_dns_editrecord", "Submit")

	_, err = s.client.PostForm(ctx, "/index.cgi", form, s.client.BaseURL().String()+ZonePagePath(zoneID))
	if err != nil {
		s.audit(ctx, "upsert_record", &zoneID, "failed", err)
		return err
	}

	records, err = s.ListRecords(ctx, zoneID)
	if err != nil {
		s.audit(ctx, "upsert_record", &zoneID, "failed", err)
		return err
	}
	if _, found := findExactRecord(records, normalized); !found {
		err = fmt.Errorf("record not found after upsert: %w", errs.ErrRemote)
		s.audit(ctx, "upsert_record", &zoneID, "failed", err)
		return err
	}
	s.audit(ctx, "upsert_record", &zoneID, "ok", nil)
	return nil
}

func (s *Service) DeleteRecord(ctx context.Context, zoneID string, in RecordInput) error {
	normalized, err := normalizeRecordInput(in)
	if err != nil {
		return err
	}
	records, err := s.ListRecords(ctx, zoneID)
	if err != nil {
		return err
	}
	match, found := findExactRecord(records, normalized)
	if !found {
		return fmt.Errorf("record not found for delete: %w", errs.ErrInvalidInput)
	}
	if match.Locked {
		return fmt.Errorf("record %s is locked and cannot be deleted: %w", match.RecordID, errs.ErrInvalidInput)
	}

	form := url.Values{}
	form.Set("hosted_dns_zoneid", zoneID)
	form.Set("hosted_dns_recordid", match.RecordID)
	form.Set("menu", "edit_zone")
	form.Set("hosted_dns_delconfirm", "delete")
	form.Set("hosted_dns_editzone", "1")
	form.Set("hosted_dns_delrecord", "1")

	_, err = s.client.PostForm(ctx, "/index.cgi", form, s.client.BaseURL().String()+ZonePagePath(zoneID))
	if err != nil {
		s.audit(ctx, "delete_record", &zoneID, "failed", err)
		return err
	}

	records, err = s.ListRecords(ctx, zoneID)
	if err != nil {
		s.audit(ctx, "delete_record", &zoneID, "failed", err)
		return err
	}
	if _, found := findExactRecord(records, normalized); found {
		err = fmt.Errorf("record still exists after delete: %w", errs.ErrRemote)
		s.audit(ctx, "delete_record", &zoneID, "failed", err)
		return err
	}
	s.audit(ctx, "delete_record", &zoneID, "ok", nil)
	return nil
}

func normalizeRecordInput(in RecordInput) (RecordInput, error) {
	in.Type = strings.ToUpper(strings.TrimSpace(in.Type))
	in.Name = strings.TrimSpace(in.Name)
	in.Value = strings.TrimSpace(in.Value)
	if in.Type == "" || in.Name == "" || in.Value == "" {
		return in, fmt.Errorf("type/name/value are required: %w", errs.ErrInvalidInput)
	}
	supported := map[string]bool{"A": true, "AAAA": true, "TXT": true, "CNAME": true, "MX": true}
	if !supported[in.Type] {
		return in, fmt.Errorf("unsupported type %q in MVP: %w", in.Type, errs.ErrInvalidInput)
	}
	if in.TTL <= 0 {
		in.TTL = 300
	}
	if in.Type == "MX" {
		if !in.HasPriority {
			in.Priority = 10
			in.HasPriority = true
		}
		if in.Priority < 0 {
			return in, fmt.Errorf("priority must be >= 0: %w", errs.ErrInvalidInput)
		}
	}
	return in, nil
}

func findExactRecord(records []model.Record, in RecordInput) (model.Record, bool) {
	for _, r := range records {
		if !strings.EqualFold(r.Type, in.Type) {
			continue
		}
		if !strings.EqualFold(strings.TrimSuffix(r.Name, "."), strings.TrimSuffix(in.Name, ".")) {
			continue
		}
		if strings.TrimSpace(r.Value) != strings.TrimSpace(in.Value) {
			continue
		}
		if in.Type == "MX" {
			if r.Priority == nil {
				continue
			}
			if *r.Priority != in.Priority {
				continue
			}
		}
		return r, true
	}
	return model.Record{}, false
}

func isDigits(v string) bool {
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return v != ""
}

func (s *Service) audit(ctx context.Context, action string, zoneID *string, status string, err error) {
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
	_ = s.auditRepo.Insert(ctx, entry)
}
