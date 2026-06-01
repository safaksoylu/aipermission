package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Host           string   `json:"host"`
	Port           string   `json:"port"`
	FrontendPort   string   `json:"frontend_port"`
	DataPath       string   `json:"data_path"`
	GatewaySecret  string   `json:"-"`
	AllowedOrigins []string `json:"-"`
}

func Load() (Config, error) {
	frontendPort := env("AIPERMISSION_FRONTEND_PORT", "3210")
	cfg := Config{
		Host:         env("AIPERMISSION_BACKEND_HOST", "127.0.0.1"),
		Port:         env("AIPERMISSION_BACKEND_PORT", "8080"),
		FrontendPort: frontendPort,
		DataPath:     env("AIPERMISSION_DATA_PATH", "./data/aipermission.db"),
		GatewaySecret: env(
			"AIPERMISSION_GATEWAY_SECRET",
			"dev-only-change-me",
		),
		AllowedOrigins: parseOrigins(env(
			"AIPERMISSION_ALLOWED_ORIGINS",
			fmt.Sprintf("http://localhost:%s,http://127.0.0.1:%s", frontendPort, frontendPort),
		)),
	}

	// docker-compose uses a visible development placeholder by default so the
	// sample file stays copy/paste friendly. The runtime never keeps that
	// placeholder: it is replaced with a generated high-entropy local secret.
	if isDefaultSecret(cfg.GatewaySecret) {
		secret, err := LoadOrCreateGatewaySecret(cfg.DataPath)
		if err != nil {
			return Config{}, err
		}
		cfg.GatewaySecret = secret
	} else if err := validateGatewaySecret(cfg.GatewaySecret); err != nil {
		return Config{}, err
	}
	if err := validateLocalBind(cfg.Host); err != nil {
		return Config{}, err
	}
	if err := validateAllowedOrigins(cfg.AllowedOrigins); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func (c Config) PublicStatus() map[string]any {
	return c.PublicStatusWithDataPath(c.DataPath)
}

func (c Config) PublicStatusWithDataPath(dataPath string) map[string]any {
	return map[string]any{
		"host":             c.Host,
		"port":             c.Port,
		"frontend_port":    c.FrontendPort,
		"data_path":        dataPath,
		"gateway_secret":   secretState(c.GatewaySecret),
		"mcp_api_url_hint": fmt.Sprintf("http://localhost:%s", c.FrontendPort),
	}
}

func (c Config) PublicStatusMinimal() map[string]any {
	return map[string]any{
		"host":             c.Host,
		"port":             c.Port,
		"frontend_port":    c.FrontendPort,
		"gateway_secret":   secretState(c.GatewaySecret),
		"mcp_api_url_hint": fmt.Sprintf("http://localhost:%s", c.FrontendPort),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func parseOrigins(value string) []string {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			origins = append(origins, part)
		}
	}
	return origins
}

func isLoopbackBind(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	return host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func validateLocalBind(host string) error {
	if isLoopbackBind(host) {
		return nil
	}
	return fmt.Errorf("AIPermission is local-only and refuses to bind to %q; set AIPERMISSION_BACKEND_HOST=127.0.0.1 and keep Docker ports bound to 127.0.0.1", host)
}

func validateAllowedOrigins(origins []string) error {
	for _, origin := range origins {
		if err := validateAllowedOrigin(origin); err != nil {
			return err
		}
	}
	return nil
}

func validateAllowedOrigin(origin string) error {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("AIPERMISSION_ALLOWED_ORIGINS entry %q must be an origin such as http://localhost:3210", origin)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("AIPERMISSION_ALLOWED_ORIGINS entry %q must use http or https", origin)
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("AIPERMISSION_ALLOWED_ORIGINS entry %q is not loopback; AIPermission only accepts localhost, 127.0.0.1, or [::1] origins", origin)
}

func secretState(value string) string {
	if isDefaultSecret(value) {
		return "development-default"
	}
	return "configured"
}

func isDefaultSecret(value string) bool {
	return value == "" || value == "dev-only-change-me" || value == "change-me-for-local-dev"
}

func validateGatewaySecret(value string) error {
	if len(strings.TrimSpace(value)) < 32 {
		return fmt.Errorf("AIPERMISSION_GATEWAY_SECRET must be at least 32 characters or omitted for auto-generation")
	}
	return nil
}

func LoadOrCreateGatewaySecret(dataPath string) (string, error) {
	path := GatewaySecretPath(dataPath)
	value, err := os.ReadFile(path)
	if err == nil && strings.TrimSpace(string(value)) != "" {
		return strings.TrimSpace(string(value)), nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read gateway secret: %w", err)
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate gateway secret: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(secret)
	if err := SaveGatewaySecret(dataPath, encoded); err != nil {
		return "", err
	}
	return encoded, nil
}

func SaveGatewaySecret(dataPath string, secret string) error {
	path := GatewaySecretPath(dataPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create gateway secret directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(secret)+"\n"), 0o600); err != nil {
		return fmt.Errorf("write gateway secret: %w", err)
	}
	return nil
}

func GatewaySecretPath(dataPath string) string {
	return filepath.Join(filepath.Dir(dataPath), "gateway.secret")
}
