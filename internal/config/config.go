// Package config loads and validates the YAML configuration for NetLog.
//
// Secrets (callbook and OIDC credentials, etc.) may be supplied in the file or
// overridden via environment variables so they need not be written to disk.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// Config is the root configuration.
type Config struct {
	Server   Server   `yaml:"server"`
	Database Database `yaml:"database"`
	Log      Log      `yaml:"log"`
	Data     Data     `yaml:"data"`
	OIDC     OIDC     `yaml:"oidc"`
	Callbook Callbook `yaml:"callbook"`
}

// Server holds HTTP server settings.
type Server struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string `yaml:"addr" validate:"required"`
	// BaseURL is the externally reachable base URL, used to build the OIDC
	// redirect URL and secure-cookie decisions, e.g. "https://net.example.org".
	BaseURL string `yaml:"base_url" validate:"required,url"`
	// CookieSecure forces the Secure flag on session cookies. Defaults to true
	// when BaseURL is https.
	CookieSecure bool `yaml:"cookie_secure"`
	// ReadTimeout / WriteTimeout bound request handling.
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// Database holds SQLite settings.
type Database struct {
	// Path is the SQLite file path, e.g. "data/netlog.sqlite".
	Path string `yaml:"path" validate:"required"`
}

// Log holds logger settings.
type Log struct {
	// Level: trace, debug, info, warn, error.
	Level string `yaml:"level" validate:"required,oneof=trace debug info warn error"`
	// Format: text or json.
	Format string `yaml:"format" validate:"required,oneof=text json"`
}

// Data holds on-disk runtime data settings.
type Data struct {
	// Dir is where cached datasets (cty.xml) live.
	Dir string `yaml:"dir" validate:"required"`
}

// OIDC holds optional single-provider OIDC settings. When Enabled is false the
// other fields are ignored.
type OIDC struct {
	Enabled      bool     `yaml:"enabled"`
	Issuer       string   `yaml:"issuer" validate:"required_if=Enabled true,omitempty,url"`
	ClientID     string   `yaml:"client_id" validate:"required_if=Enabled true"`
	ClientSecret string   `yaml:"client_secret" validate:"required_if=Enabled true"`
	RedirectURL  string   `yaml:"redirect_url" validate:"omitempty,url"`
	Scopes       []string `yaml:"scopes"`
}

// Callbook configures the QRZ + HamQTH lookups. Order defines primary→fallback.
type Callbook struct {
	// Order lists provider keys in priority order, e.g. ["qrz", "hamqth"].
	Order []string `yaml:"order" validate:"required,min=1,dive,oneof=qrz hamqth"`
	// HTTPTimeout bounds each callbook HTTP request.
	HTTPTimeout time.Duration `yaml:"http_timeout"`
	// CTYRefreshDays is how often the cty.xml DXCC dataset is refreshed.
	CTYRefreshDays int    `yaml:"cty_refresh_days" validate:"min=1"`
	QRZ            QRZ    `yaml:"qrz"`
	HamQTH         HamQTH `yaml:"hamqth"`
}

// QRZ holds QRZ XML callbook credentials.
type QRZ struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	// Agent identifies this client to QRZ, e.g. "NetLog/0.1".
	Agent string `yaml:"agent"`
}

// HamQTH holds HamQTH callbook credentials.
type HamQTH struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	// Program identifies this client to HamQTH, e.g. "NetLog".
	Program string `yaml:"program"`
}

// Enabled reports whether a given provider key has usable credentials.
func (c Callbook) Enabled(provider string) bool {
	switch provider {
	case "qrz":
		return c.QRZ.Username != "" && c.QRZ.Password != ""
	case "hamqth":
		return c.HamQTH.Username != "" && c.HamQTH.Password != ""
	default:
		return false
	}
}

// Defaults returns a Config populated with sensible defaults. Load merges the
// YAML file on top of these.
func Defaults() Config {
	return Config{
		Server: Server{
			Addr:         ":8080",
			BaseURL:      "http://localhost:8080",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Database: Database{Path: "data/netlog.sqlite"},
		Log:      Log{Level: "info", Format: "text"},
		Data:     Data{Dir: "data"},
		OIDC:     OIDC{Scopes: []string{"openid", "profile", "email"}},
		Callbook: Callbook{
			Order:          []string{"qrz", "hamqth"},
			HTTPTimeout:    15 * time.Second,
			CTYRefreshDays: 28,
			QRZ:            QRZ{Agent: "NetLog/0.1"},
			HamQTH:         HamQTH{Program: "NetLog"},
		},
	}
}

// Load reads the YAML config at path on top of Defaults, applies environment
// overrides for secrets, derives dependent defaults, and validates the result.
//
// A missing file at path is not an error: NetLog falls back to the built-in
// defaults overridden by NETLOG_* environment variables. This is the
// configuration-by-environment path used by the container image, where the
// operator may run with env vars alone and never mount a config file.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	switch raw, err := os.ReadFile(path); {
	case err == nil:
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("parse config %q: %w", path, err)
		}
	case errors.Is(err, fs.ErrNotExist):
		// No file: defaults + environment overrides only.
	default:
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	applyEnvOverrides(&cfg)
	deriveDefaults(&cfg)

	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

// applyEnvOverrides lets secrets be injected without touching the file. Empty
// values leave the file/default untouched.
func applyEnvOverrides(cfg *Config) {
	set := func(dst *string, env string) {
		if v, ok := os.LookupEnv(env); ok && v != "" {
			*dst = v
		}
	}
	setBool := func(dst *bool, env string) {
		if v, ok := os.LookupEnv(env); ok && v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				*dst = b
			}
		}
	}
	set(&cfg.Database.Path, "NETLOG_DB_PATH")
	set(&cfg.Data.Dir, "NETLOG_DATA_DIR")
	set(&cfg.Server.Addr, "NETLOG_ADDR")
	set(&cfg.Server.BaseURL, "NETLOG_BASE_URL")
	set(&cfg.Log.Level, "NETLOG_LOG_LEVEL")
	set(&cfg.Log.Format, "NETLOG_LOG_FORMAT")
	setBool(&cfg.OIDC.Enabled, "NETLOG_OIDC_ENABLED")
	set(&cfg.OIDC.ClientID, "NETLOG_OIDC_CLIENT_ID")
	set(&cfg.OIDC.ClientSecret, "NETLOG_OIDC_CLIENT_SECRET")
	set(&cfg.OIDC.Issuer, "NETLOG_OIDC_ISSUER")
	set(&cfg.Callbook.QRZ.Username, "NETLOG_QRZ_USERNAME")
	set(&cfg.Callbook.QRZ.Password, "NETLOG_QRZ_PASSWORD")
	set(&cfg.Callbook.HamQTH.Username, "NETLOG_HAMQTH_USERNAME")
	set(&cfg.Callbook.HamQTH.Password, "NETLOG_HAMQTH_PASSWORD")
}

// deriveDefaults fills in values that depend on other fields.
func deriveDefaults(cfg *Config) {
	if cfg.OIDC.Enabled && cfg.OIDC.RedirectURL == "" {
		cfg.OIDC.RedirectURL = strings.TrimRight(cfg.Server.BaseURL, "/") + "/api/auth/oidc/callback"
	}
	if strings.HasPrefix(cfg.Server.BaseURL, "https://") {
		cfg.Server.CookieSecure = true
	}
}
