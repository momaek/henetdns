package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/momaek/henetdns/internal/errs"
)

type Config struct {
	BaseURL  string
	DataDir  string
	Username string
	Email    string // deprecated alias for Username
	Password string
	Timeout  time.Duration
}

func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./henetdns-data"
	}
	return filepath.Join(home, ".config", "henetdns")
}

func ApplyEnv(cfg *Config) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = strings.TrimSpace(os.Getenv("HENETDNS_BASE_URL"))
	}
	if cfg.DataDir == "" {
		cfg.DataDir = strings.TrimSpace(os.Getenv("HENETDNS_DATA_DIR"))
	}
	if cfg.Password == "" {
		cfg.Password = os.Getenv("HE_PASS")
	}
	if cfg.Timeout == 0 {
		if raw := strings.TrimSpace(os.Getenv("HENETDNS_TIMEOUT")); raw != "" {
			if d, err := time.ParseDuration(raw); err == nil {
				cfg.Timeout = d
			}
		}
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://dns.he.net"
	}
	if cfg.DataDir == "" {
		cfg.DataDir = DefaultDataDir()
	}
	if cfg.Username == "" {
		cfg.Username = strings.TrimSpace(cfg.Email)
	}
	if cfg.Username == "" {
		cfg.Username = strings.TrimSpace(os.Getenv("HE_USERNAME"))
	}
	if cfg.Username == "" {
		cfg.Username = strings.TrimSpace(os.Getenv("HE_EMAIL"))
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 20 * time.Second
	}
}

func ValidateCommon(cfg Config) error {
	if cfg.BaseURL == "" {
		return fmt.Errorf("base url is required: %w", errs.ErrInvalidInput)
	}
	if cfg.DataDir == "" {
		return fmt.Errorf("data dir is required: %w", errs.ErrInvalidInput)
	}
	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be > 0: %w", errs.ErrInvalidInput)
	}
	return nil
}
