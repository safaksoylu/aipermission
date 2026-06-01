package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesAndReusesGatewaySecretWhenDefaultConfigured(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "aipermission.db")
	t.Setenv("AIPERMISSION_DATA_PATH", dataPath)
	t.Setenv("AIPERMISSION_GATEWAY_SECRET", "dev-only-change-me")
	t.Setenv("AIPERMISSION_FRONTEND_PORT", "3333")
	t.Setenv("AIPERMISSION_ALLOWED_ORIGINS", "")

	first, err := Load()
	if err != nil {
		t.Fatalf("load first config: %v", err)
	}
	if first.GatewaySecret == "" || first.GatewaySecret == "dev-only-change-me" {
		t.Fatalf("expected generated gateway secret, got %q", first.GatewaySecret)
	}
	if first.Address() != "127.0.0.1:8080" {
		t.Fatalf("unexpected address: %s", first.Address())
	}
	if got := first.AllowedOrigins; len(got) != 2 || got[0] != "http://localhost:3333" || got[1] != "http://127.0.0.1:3333" {
		t.Fatalf("unexpected origins from frontend port: %#v", got)
	}
	if first.PublicStatus()["gateway_secret"] != "configured" {
		t.Fatalf("public status should not expose secret: %#v", first.PublicStatus())
	}
	if _, ok := first.PublicStatus()["remote_access"]; ok {
		t.Fatalf("public status should not expose remote access mode: %#v", first.PublicStatus())
	}

	second, err := Load()
	if err != nil {
		t.Fatalf("load second config: %v", err)
	}
	if second.GatewaySecret != first.GatewaySecret {
		t.Fatalf("gateway secret should be reused")
	}
	info, err := os.Stat(GatewaySecretPath(dataPath))
	if err != nil {
		t.Fatalf("stat gateway secret: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("gateway secret permissions = %o", info.Mode().Perm())
	}
}

func TestLoadDefaultsCORSOriginsToFrontendPort3210(t *testing.T) {
	t.Setenv("AIPERMISSION_DATA_PATH", filepath.Join(t.TempDir(), "aipermission.db"))
	t.Setenv("AIPERMISSION_GATEWAY_SECRET", "real-secret-with-at-least-32-characters")
	t.Setenv("AIPERMISSION_ALLOWED_ORIGINS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.AllowedOrigins; len(got) != 2 || got[0] != "http://localhost:3210" || got[1] != "http://127.0.0.1:3210" {
		t.Fatalf("unexpected default origins: %#v", got)
	}
}

func TestLoadHonorsExplicitEnv(t *testing.T) {
	t.Setenv("AIPERMISSION_BACKEND_HOST", "localhost")
	t.Setenv("AIPERMISSION_BACKEND_PORT", "9000")
	t.Setenv("AIPERMISSION_DATA_PATH", filepath.Join(t.TempDir(), "custom.db"))
	t.Setenv("AIPERMISSION_GATEWAY_SECRET", "real-secret-with-at-least-32-characters")
	t.Setenv("AIPERMISSION_ALLOWED_ORIGINS", " http://localhost:9001, ,http://127.0.0.1:9001,https://[::1]:9001 ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Address() != "localhost:9000" {
		t.Fatalf("unexpected address: %s", cfg.Address())
	}
	if got := cfg.AllowedOrigins; len(got) != 3 || got[0] != "http://localhost:9001" || got[1] != "http://127.0.0.1:9001" || got[2] != "https://[::1]:9001" {
		t.Fatalf("unexpected origins: %#v", got)
	}
	if _, err := os.Stat(GatewaySecretPath(cfg.DataPath)); !os.IsNotExist(err) {
		t.Fatalf("explicit gateway secret should not create gateway.secret, err=%v", err)
	}
}

func TestLoadRejectsNonLoopbackAllowedOrigin(t *testing.T) {
	t.Setenv("AIPERMISSION_DATA_PATH", filepath.Join(t.TempDir(), "custom.db"))
	t.Setenv("AIPERMISSION_GATEWAY_SECRET", "real-secret-with-at-least-32-characters")
	t.Setenv("AIPERMISSION_ALLOWED_ORIGINS", "https://example.com")

	if _, err := Load(); err == nil {
		t.Fatalf("expected non-loopback allowed origin to fail")
	}
}

func TestLoadRejectsNonLocalBind(t *testing.T) {
	t.Setenv("AIPERMISSION_BACKEND_HOST", "0.0.0.0")
	t.Setenv("AIPERMISSION_DATA_PATH", filepath.Join(t.TempDir(), "custom.db"))
	t.Setenv("AIPERMISSION_GATEWAY_SECRET", "real-secret-with-at-least-32-characters")

	if _, err := Load(); err == nil {
		t.Fatalf("expected non-local bind to fail")
	}
}

func TestLoadRejectsWeakExplicitGatewaySecret(t *testing.T) {
	t.Setenv("AIPERMISSION_DATA_PATH", filepath.Join(t.TempDir(), "custom.db"))
	t.Setenv("AIPERMISSION_GATEWAY_SECRET", "short")

	if _, err := Load(); err == nil {
		t.Fatalf("expected weak explicit gateway secret to fail")
	}
}
