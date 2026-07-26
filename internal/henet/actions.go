package henet

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/momaek/henetdns/internal/errs"
	"github.com/momaek/henetdns/internal/httpclient"
	"github.com/momaek/henetdns/internal/model"
	"github.com/momaek/henetdns/internal/store"
)

type Service struct {
	client     *httpclient.Client
	zoneRepo   *store.ZoneRepo
	recordRepo *store.RecordRepo
}

type RecordInput struct {
	Type        string
	Name        string
	Value       string
	TTL         int
	Priority    int
	HasPriority bool
}

func NewService(client *httpclient.Client, zoneRepo *store.ZoneRepo, recordRepo *store.RecordRepo) *Service {
	return &Service{client: client, zoneRepo: zoneRepo, recordRepo: recordRepo}
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

// ResolveZoneIDCachedFirst resolves a zone name to its ID using the local
// cache first. Falls back to a remote fetch (which also refreshes the cache)
// if the zone is not found locally.
func (s *Service) ResolveZoneIDCachedFirst(ctx context.Context, zoneOrID string) (string, error) {
	id, err := s.ResolveZoneIDFromCache(ctx, zoneOrID)
	if err == nil {
		return id, nil
	}
	return s.ResolveZoneID(ctx, zoneOrID)
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

// resolveZoneName maps a zone ID back to its zone name, cache-first with a
// remote fallback. Returns "" when the zone cannot be resolved; callers treat
// that as "leave the record name as given".
func (s *Service) resolveZoneName(ctx context.Context, zoneID string) string {
	if s.zoneRepo != nil {
		if name, found, err := s.zoneRepo.FindNameByID(ctx, zoneID); err == nil && found {
			return name
		}
	}
	zones, err := s.ListZones(ctx)
	if err != nil {
		return ""
	}
	for _, z := range zones {
		if z.ID == zoneID {
			return z.Name
		}
	}
	return ""
}

func (s *Service) UpsertRecord(ctx context.Context, zoneID string, in RecordInput) error {
	normalized, err := normalizeRecordInput(in)
	if err != nil {
		return err
	}
	normalized.Name = qualifyRecordName(normalized.Name, s.resolveZoneName(ctx, zoneID))
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
		return err
	}

	records, err = s.ListRecords(ctx, zoneID)
	if err != nil {
		return err
	}
	if _, found := findExactRecord(records, normalized); !found {
		return fmt.Errorf("record not found after upsert: %w", errs.ErrRemote)
	}
	return nil
}

func (s *Service) DeleteRecord(ctx context.Context, zoneID string, in RecordInput) error {
	normalized, err := normalizeRecordInput(in)
	if err != nil {
		return err
	}
	normalized.Name = qualifyRecordName(normalized.Name, s.resolveZoneName(ctx, zoneID))
	records, err := s.ListRecords(ctx, zoneID)
	if err != nil {
		return err
	}
	match, found := findExactRecord(records, normalized)
	if !found {
		msg := fmt.Sprintf("record not found for delete (type=%s name=%s value=%s)", normalized.Type, normalized.Name, normalized.Value)
		if hints := nearMissHints(records, normalized); hints != "" {
			msg += "; close matches: " + hints
		}
		return fmt.Errorf("%s: %w", msg, errs.ErrInvalidInput)
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
		return err
	}

	records, err = s.ListRecords(ctx, zoneID)
	if err != nil {
		return err
	}
	if _, found := findExactRecord(records, normalized); found {
		return fmt.Errorf("record still exists after delete: %w", errs.ErrRemote)
	}
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

// qualifyRecordName expands a short record name to its fully-qualified form
// within zone, so `www` and `www.example.com` address the same record. `@` and
// an empty name mean the zone apex. Unknown zone leaves the name untouched.
func qualifyRecordName(name, zone string) string {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".")
	zone = strings.TrimSuffix(strings.TrimSpace(zone), ".")
	if zone == "" {
		return name
	}
	if name == "" || name == "@" {
		return zone
	}
	if strings.EqualFold(name, zone) || strings.HasSuffix(strings.ToLower(name), "."+strings.ToLower(zone)) {
		return name
	}
	return name + "." + zone
}

// normalizeTXTValue strips one pair of surrounding double quotes so the bare
// token matches the quoted form he.net returns in record listings.
func normalizeTXTValue(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`) {
		v = v[1 : len(v)-1]
	}
	return v
}

func recordValueEqual(rrType, a, b string) bool {
	if rrType == "TXT" {
		return normalizeTXTValue(a) == normalizeTXTValue(b)
	}
	return strings.TrimSpace(a) == strings.TrimSpace(b)
}

func findExactRecord(records []model.Record, in RecordInput) (model.Record, bool) {
	for _, r := range records {
		if !strings.EqualFold(r.Type, in.Type) {
			continue
		}
		if !strings.EqualFold(strings.TrimSuffix(r.Name, "."), strings.TrimSuffix(in.Name, ".")) {
			continue
		}
		if !recordValueEqual(in.Type, r.Value, in.Value) {
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

// nearMissHints lists records sharing the requested name (any type/value), so
// a failed exact match points at the mismatching field instead of a dead end.
func nearMissHints(records []model.Record, in RecordInput) string {
	const maxHints = 5
	var hints []string
	for _, r := range records {
		if !strings.EqualFold(strings.TrimSuffix(r.Name, "."), strings.TrimSuffix(in.Name, ".")) {
			continue
		}
		hints = append(hints, fmt.Sprintf("[%s %s %s]", r.Type, r.Name, r.Value))
		if len(hints) == maxHints {
			break
		}
	}
	return strings.Join(hints, " ")
}

func isDigits(v string) bool {
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return v != ""
}
