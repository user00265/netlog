package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadDefaultsAndOverrides(t *testing.T) {
	p := writeConfig(t, `
server:
  base_url: "https://net.example.org"
callbook:
  order: ["hamqth", "qrz"]
  qrz:
    username: "file-user"
    password: "file-pass"
`)
	t.Setenv("NETLOG_QRZ_PASSWORD", "env-pass")

	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Default merged in.
	if cfg.Server.Addr != ":8080" {
		t.Errorf("Addr default not applied: %q", cfg.Server.Addr)
	}
	// https base_url => secure cookies + derived redirect URL.
	if !cfg.Server.CookieSecure {
		t.Error("expected CookieSecure derived from https base_url")
	}
	// File value kept where no env override.
	if cfg.Callbook.QRZ.Username != "file-user" {
		t.Errorf("username = %q", cfg.Callbook.QRZ.Username)
	}
	// Env override wins.
	if cfg.Callbook.QRZ.Password != "env-pass" {
		t.Errorf("expected env override, got %q", cfg.Callbook.QRZ.Password)
	}
	// Order honored.
	if cfg.Callbook.Order[0] != "hamqth" {
		t.Errorf("order = %v", cfg.Callbook.Order)
	}
	if !cfg.Callbook.Enabled("qrz") {
		t.Error("expected qrz enabled with credentials")
	}
	if cfg.Callbook.Enabled("hamqth") {
		t.Error("expected hamqth disabled without credentials")
	}
}

func TestLoadMissingFileUsesDefaultsAndEnv(t *testing.T) {
	// The container image runs config-by-environment: no file mounted, all
	// settings supplied via NETLOG_*. A missing path must not be an error.
	missing := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	t.Setenv("NETLOG_DATA_DIR", "/data")
	t.Setenv("NETLOG_DB_PATH", "/data/netlog.sqlite")
	t.Setenv("NETLOG_ADDR", ":9999")

	cfg, err := Load(missing)
	if err != nil {
		t.Fatalf("Load with missing file: %v", err)
	}
	if cfg.Data.Dir != "/data" {
		t.Errorf("NETLOG_DATA_DIR not applied: %q", cfg.Data.Dir)
	}
	if cfg.Database.Path != "/data/netlog.sqlite" {
		t.Errorf("NETLOG_DB_PATH not applied: %q", cfg.Database.Path)
	}
	if cfg.Server.Addr != ":9999" {
		t.Errorf("NETLOG_ADDR not applied: %q", cfg.Server.Addr)
	}
	// Untouched defaults still present.
	if cfg.Callbook.CTYRefreshDays != 28 {
		t.Errorf("default CTYRefreshDays lost: %d", cfg.Callbook.CTYRefreshDays)
	}
}

func TestLoadOIDCValidation(t *testing.T) {
	// Enabled OIDC without client id/secret must fail validation.
	p := writeConfig(t, `
server:
  base_url: "http://localhost:8080"
oidc:
  enabled: true
  issuer: "https://idp.example.com"
callbook:
  order: ["qrz"]
`)
	if _, err := Load(p); err == nil {
		t.Error("expected validation error for incomplete OIDC config")
	}
}

func TestLoadRejectsBadProvider(t *testing.T) {
	p := writeConfig(t, `
server:
  base_url: "http://localhost:8080"
callbook:
  order: ["bogus"]
`)
	if _, err := Load(p); err == nil {
		t.Error("expected validation error for unknown callbook provider")
	}
}
