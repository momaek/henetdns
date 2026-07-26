package store

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/momaek/henetdns/internal/model"
)

// cacheData is the on-disk shape of <dir>/cache.json. Zones and records share
// one file, so both repos read-modify-write the whole document under st.mu.
type cacheData struct {
	Zones   []model.Zone              `json:"zones"`
	Records map[string][]model.Record `json:"records"`
}

func (s *Store) loadCache() (*cacheData, error) {
	data := &cacheData{Records: map[string][]model.Record{}}
	if _, err := readJSON(s.cachePath(), data); err != nil {
		return nil, err
	}
	if data.Records == nil {
		data.Records = map[string][]model.Record{}
	}
	return data, nil
}

// ZoneRepo caches the zone list (cached-first lookups).
type ZoneRepo struct {
	st *Store
}

func NewZoneRepo(st *Store) *ZoneRepo {
	return &ZoneRepo{st: st}
}

func (r *ZoneRepo) ReplaceAll(ctx context.Context, zones []model.Zone, syncedAt time.Time) error {
	r.st.mu.Lock()
	defer r.st.mu.Unlock()

	data, err := r.st.loadCache()
	if err != nil {
		return err
	}
	synced := syncedAt.UTC()
	stamped := make([]model.Zone, len(zones))
	for i, z := range zones {
		z.LastSyncedAt = synced
		stamped[i] = z
	}
	data.Zones = stamped
	return writeJSON(r.st.cachePath(), data)
}

func (r *ZoneRepo) List(ctx context.Context) ([]model.Zone, error) {
	data, err := r.st.loadCache()
	if err != nil {
		return nil, err
	}
	zones := append([]model.Zone(nil), data.Zones...)
	sort.Slice(zones, func(i, j int) bool {
		ni, nj := strings.ToLower(zones[i].Name), strings.ToLower(zones[j].Name)
		if ni != nj {
			return ni < nj
		}
		return zones[i].ID < zones[j].ID
	})
	return zones, nil
}

func (r *ZoneRepo) FindIDByName(ctx context.Context, zoneName string) (string, bool, error) {
	zoneName = strings.TrimSpace(zoneName)
	if zoneName == "" {
		return "", false, nil
	}
	data, err := r.st.loadCache()
	if err != nil {
		return "", false, err
	}
	for _, z := range data.Zones {
		if strings.EqualFold(z.Name, zoneName) {
			return z.ID, true, nil
		}
	}
	return "", false, nil
}

func (r *ZoneRepo) FindNameByID(ctx context.Context, zoneID string) (string, bool, error) {
	zoneID = strings.TrimSpace(zoneID)
	if zoneID == "" {
		return "", false, nil
	}
	data, err := r.st.loadCache()
	if err != nil {
		return "", false, err
	}
	for _, z := range data.Zones {
		if z.ID == zoneID {
			return z.Name, true, nil
		}
	}
	return "", false, nil
}

// RecordRepo caches records per zone (cached-first lookups).
type RecordRepo struct {
	st *Store
}

func NewRecordRepo(st *Store) *RecordRepo {
	return &RecordRepo{st: st}
}

func (r *RecordRepo) ReplaceAllForZone(ctx context.Context, zoneID string, records []model.Record, syncedAt time.Time) error {
	r.st.mu.Lock()
	defer r.st.mu.Unlock()

	data, err := r.st.loadCache()
	if err != nil {
		return err
	}
	synced := syncedAt.UTC()
	stamped := make([]model.Record, len(records))
	for i, rec := range records {
		rec.ZoneID = zoneID
		rec.LastSyncedAt = synced
		stamped[i] = rec
	}
	data.Records[zoneID] = stamped
	return writeJSON(r.st.cachePath(), data)
}

func (r *RecordRepo) ListByZone(ctx context.Context, zoneID string) ([]model.Record, error) {
	data, err := r.st.loadCache()
	if err != nil {
		return nil, err
	}
	records := append([]model.Record(nil), data.Records[zoneID]...)
	sort.Slice(records, func(i, j int) bool {
		ni, nj := strings.ToLower(records[i].Name), strings.ToLower(records[j].Name)
		if ni != nj {
			return ni < nj
		}
		ti, tj := strings.ToLower(records[i].Type), strings.ToLower(records[j].Type)
		if ti != tj {
			return ti < tj
		}
		return records[i].RecordID < records[j].RecordID
	})
	return records, nil
}
