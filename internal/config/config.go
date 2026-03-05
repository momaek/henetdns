package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wentx/henetdns/internal/errs"
)

type Config struct {
	BaseURL  string
	DBPath   string
	Username string
	Email    string // deprecated alias for Username
	Password string
	Timeout  time.Duration
}

func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./henetdns-client.db"
	}
	return filepath.Join(home, ".config", "henetdns", "client.db")
}

func ApplyEnv(cfg *Config) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = strings.TrimSpace(os.Getenv("HENETDNS_BASE_URL"))
	}
	if cfg.DBPath == "" {
		cfg.DBPath = strings.TrimSpace(os.Getenv("HENETDNS_DB_PATH"))
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
	if cfg.DBPath == "" {
		cfg.DBPath = DefaultDBPath()
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
	if cfg.DBPath == "" {
		return fmt.Errorf("db path is required: %w", errs.ErrInvalidInput)
	}
	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be > 0: %w", errs.ErrInvalidInput)
	}
	return nil
}
