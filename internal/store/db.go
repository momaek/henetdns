package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/momaek/henetdns/internal/errs"
)

// Store is a small file-backed store rooted at a data directory. It replaces
// the previous SQLite database: a CLI is single-user and single-host, so a
// couple of JSON files are simpler and dependency-free.
//
//	<dir>/session.json  saved login sessions (cookie jars)
//	<dir>/cache.json    zones + records cache (cached-first optimization)
type Store struct {
	dir string
	mu  sync.Mutex // guards read-modify-write of cache.json
}

// Open ensures the data directory exists and returns a Store rooted at it.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w: %w", err, errs.ErrStore)
	}
	return &Store{dir: dir}, nil
}

// Close is a no-op; it exists so callers can treat Store like the old DB handle.
func (s *Store) Close() error { return nil }

func (s *Store) sessionPath() string { return filepath.Join(s.dir, "session.json") }
func (s *Store) cachePath() string   { return filepath.Join(s.dir, "cache.json") }

// readJSON loads a JSON file into v. A missing file is not an error: v is left
// untouched and (false, nil) is returned.
func readJSON(path string, v any) (bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s: %w: %w", filepath.Base(path), err, errs.ErrStore)
	}
	if len(data) == 0 {
		return false, nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return false, fmt.Errorf("parse %s: %w: %w", filepath.Base(path), err, errs.ErrStore)
	}
	return true, nil
}

// writeJSON atomically writes v as indented JSON to path (temp file + rename).
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w: %w", filepath.Base(path), err, errs.ErrStore)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w: %w", filepath.Base(path), err, errs.ErrStore)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("commit %s: %w: %w", filepath.Base(path), err, errs.ErrStore)
	}
	return nil
}
