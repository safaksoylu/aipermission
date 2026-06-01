package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aipermission/aipermission/backend/internal/config"
)

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	server := NewLockedServer(config.Config{
		Host:           "127.0.0.1",
		Port:           "8080",
		DataPath:       t.TempDir() + "/aipermission.db",
		GatewaySecret:  "test-secret",
		AllowedOrigins: []string{"http://localhost:3001"},
	})

	request := httptest.NewRequest(http.MethodOptions, "/api/status", nil)
	request.Host = "localhost:8080"
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("Origin", "http://localhost:3001")
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3001" {
		t.Fatalf("unexpected allow origin header: %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("unexpected allow credentials header: %q", got)
	}
}

func TestCORSRejectsUnexpectedOrigin(t *testing.T) {
	server := NewLockedServer(config.Config{
		Host:           "127.0.0.1",
		Port:           "8080",
		DataPath:       t.TempDir() + "/aipermission.db",
		GatewaySecret:  "test-secret",
		AllowedOrigins: []string{"http://localhost:3001"},
	})

	request := httptest.NewRequest(http.MethodOptions, "/api/status", nil)
	request.Host = "localhost:8080"
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("Origin", "https://example.com")
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected allow origin header: %q", got)
	}
}

func TestCORSRejectsWildcardOriginConfiguration(t *testing.T) {
	server := NewLockedServer(config.Config{
		Host:           "127.0.0.1",
		Port:           "8080",
		DataPath:       t.TempDir() + "/aipermission.db",
		GatewaySecret:  "test-secret",
		AllowedOrigins: []string{"*"},
	})

	request := httptest.NewRequest(http.MethodOptions, "/api/status", nil)
	request.Host = "localhost:8080"
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("Origin", "https://example.com")
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("wildcard origin should not be accepted, got %d", response.Code)
	}
}

func TestCORSAllowsNonBrowserRequestsWithoutOrigin(t *testing.T) {
	server := NewLockedServer(config.Config{
		Host:           "127.0.0.1",
		Port:           "8080",
		DataPath:       t.TempDir() + "/aipermission.db",
		GatewaySecret:  "test-secret",
		AllowedOrigins: []string{"http://localhost:3001"},
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Host = "localhost:8080"
	request.RemoteAddr = "127.0.0.1:12345"
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, response.Code)
	}
}

func TestRemoteHostHeaderAndRemoteClientAreAlwaysRejected(t *testing.T) {
	server := NewLockedServer(config.Config{
		Host:           "0.0.0.0",
		Port:           "8080",
		DataPath:       t.TempDir() + "/aipermission.db",
		GatewaySecret:  "test-secret",
		AllowedOrigins: []string{"http://localhost:3001"},
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Host = "192.0.2.10:8080"
	request.RemoteAddr = "192.0.2.20:12345"
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected remote host header to be rejected, got %d", response.Code)
	}

	response = httptest.NewRecorder()
	request.Host = "localhost:8080"
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected remote client with localhost host header to be rejected, got %d", response.Code)
	}
}
