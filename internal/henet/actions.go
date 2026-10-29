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

	form := buildCreateForm(zoneID, normalized)
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
	zoneName := s.resolveZoneName(ctx, zoneID)
	normalized.Name = qualifyRecordName(normalized.Name, zoneName)
	if isApexNS(normalized, zoneName) {
		return fmt.Errorf("refusing to delete NS record at zone apex %s (it delegates the whole zone; use the he.net web UI if you really mean it): %w", normalized.Name, errs.ErrInvalidInput)
	}
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

// SupportedTypes lists the record types he.net's zone editor offers. All of
// them except MX and SRV use the same Name/Content/TTL formlet.
var SupportedTypes = []string{
	"A", "AAAA", "AFSDB", "ALIAS", "CAA", "CNAME", "HINFO", "HTTPS", "LOC",
	"MX", "NAPTR", "NS", "PTR", "RP", "SPF", "SRV", "SSHFP", "SVCB", "TXT",
}

// ValidTTLs are the only values he.net's TTL <select> offers.
var ValidTTLs = []int{300, 900, 1800, 3600, 7200, 14400, 28800, 43200, 86400, 172800}

// hostnameTypes hold a single hostname as their value; he.net lists them
// without a trailing dot and DNS names compare case-insensitively.
var hostnameTypes = map[string]bool{"CNAME": true, "MX": true, "NS": true, "ALIAS": true, "PTR": true}

func isSupportedType(t string) bool {
	for _, s := range SupportedTypes {
		if s == t {
			return true
		}
	}
	return false
}

func isValidTTL(ttl int) bool {
	for _, v := range ValidTTLs {
		if v == ttl {
			return true
		}
	}
	return false
}

func hasPriorityField(rrType string) bool {
	return rrType == "MX" || rrType == "SRV"
}

func normalizeRecordInput(in RecordInput) (RecordInput, error) {
	in.Type = strings.ToUpper(strings.TrimSpace(in.Type))
	in.Name = strings.TrimSpace(in.Name)
	in.Value = strings.TrimSpace(in.Value)
	if in.Type == "" || in.Name == "" || in.Value == "" {
		return in, fmt.Errorf("type/name/value are required: %w", errs.ErrInvalidInput)
	}
	if !isSupportedType(in.Type) {
		return in, fmt.Errorf("unsupported type %q (supported: %s): %w", in.Type, strings.Join(SupportedTypes, ", "), errs.ErrInvalidInput)
	}
	if in.TTL <= 0 {
		in.TTL = 300
	}
	if !isValidTTL(in.TTL) {
		return in, fmt.Errorf("ttl %d not offered by he.net (valid: %s): %w", in.TTL, joinInts(ValidTTLs), errs.ErrInvalidInput)
	}
	if hostnameTypes[in.Type] {
		in.Value = strings.TrimSuffix(in.Value, ".")
	}
	switch in.Type {
	case "MX":
		if !in.HasPriority {
			in.Priority = 10
			in.HasPriority = true
		}
	case "SRV":
		srv, err := normalizeSRV(in)
		if err != nil {
			return in, err
		}
		in = srv
	}
	if hasPriorityField(in.Type) && (in.Priority < 0 || in.Priority > 65535) {
		return in, fmt.Errorf("priority must be 0-65535: %w", errs.ErrInvalidInput)
	}
	return in, nil
}

// normalizeSRV accepts the value as "weight port target" (the form he.net
// lists, with priority in its own column) or as the zone-file form
// "priority weight port target" when no explicit priority was given.
// Priority defaults to 0.
func normalizeSRV(in RecordInput) (RecordInput, error) {
	fields := strings.Fields(in.Value)
	if len(fields) == 4 && !in.HasPriority {
		p, err := strconv.Atoi(fields[0])
		if err != nil {
			return in, fmt.Errorf("SRV priority %q is not a number: %w", fields[0], errs.ErrInvalidInput)
		}
		in.Priority = p
		in.HasPriority = true
		fields = fields[1:]
	}
	if len(fields) != 3 {
		return in, fmt.Errorf(`SRV value must be "weight port target" (or "priority weight port target" without --priority-set), got %q: %w`, in.Value, errs.ErrInvalidInput)
	}
	for i, label := range []string{"weight", "port"} {
		n, err := strconv.Atoi(fields[i])
		if err != nil || n < 0 || n > 65535 {
			return in, fmt.Errorf("SRV %s must be 0-65535, got %q: %w", label, fields[i], errs.ErrInvalidInput)
		}
	}
	fields[2] = strings.TrimSuffix(fields[2], ".")
	in.Value = strings.Join(fields, " ")
	if !in.HasPriority {
		in.Priority = 0
		in.HasPriority = true
	}
	return in, nil
}

// buildCreateForm mirrors the he.net formlet for the record type
// (/formlets/<type>.html). SRV is the only type that splits its value into
// separate Weight/Port/Target fields instead of a single Content field.
func buildCreateForm(zoneID string, in RecordInput) url.Values {
	form := url.Values{}
	form.Set("menu", "edit_zone")
	form.Set("Type", in.Type)
	form.Set("hosted_dns_zoneid", zoneID)
	form.Set("hosted_dns_recordid", "")
	form.Set("hosted_dns_editzone", "1")
	form.Set("Name", in.Name)
	form.Set("TTL", strconv.Itoa(in.TTL))
	if hasPriorityField(in.Type) {
		form.Set("Priority", strconv.Itoa(in.Priority))
	} else {
		form.Set("Priority", "")
	}
	if in.Type == "SRV" {
		fields := strings.Fields(in.Value)
		form.Set("Weight", fields[0])
		form.Set("Port", fields[1])
		form.Set("Target", fields[2])
	} else {
		form.Set("Content", in.Value)
	}
	form.Set("hosted_dns_editrecord", "Submit")
	return form
}

// isApexNS reports whether in is an NS record at the zone apex. Deleting
// those breaks delegation for the entire zone, and he.net does not lock them.
func isApexNS(in RecordInput, zoneName string) bool {
	if in.Type != "NS" {
		return false
	}
	name := strings.TrimSuffix(in.Name, ".")
	if name == "@" || name == "" {
		return true
	}
	return zoneName != "" && strings.EqualFold(name, strings.TrimSuffix(zoneName, "."))
}

func joinInts(vs []int) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ", ")
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
	switch {
	case rrType == "TXT" || rrType == "SPF":
		return normalizeTXTValue(a) == normalizeTXTValue(b)
	case hostnameTypes[rrType]:
		return strings.EqualFold(strings.TrimSuffix(strings.TrimSpace(a), "."), strings.TrimSuffix(strings.TrimSpace(b), "."))
	case rrType == "SRV":
		fa, fb := strings.Fields(a), strings.Fields(b)
		if len(fa) != 3 || len(fb) != 3 {
			return strings.Join(fa, " ") == strings.Join(fb, " ")
		}
		return fa[0] == fb[0] && fa[1] == fb[1] &&
			strings.EqualFold(strings.TrimSuffix(fa[2], "."), strings.TrimSuffix(fb[2], "."))
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
		if hasPriorityField(in.Type) {
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
