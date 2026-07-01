package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultPathUsesUnixConfigHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "md2x", "config.yaml")
	if path != want {
		t.Fatalf("DefaultPath() = %q, want %q", path, want)
	}
	if strings.Contains(path, "Application Support") {
		t.Fatalf("DefaultPath() used platform app config directory: %q", path)
	}
}

func TestDefaultPathUsesXDGConfigHomeOverride(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(configHome, "md2x", "config.yaml")
	if path != want {
		t.Fatalf("DefaultPath() = %q, want %q", path, want)
	}
}

func TestLoadMissingConfigReturnsDefaults(t *testing.T) {
	cfg, found, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if found {
		t.Fatal("found = true, want false")
	}
	if cfg.API.BaseURL != DefaultAPIBaseURL {
		t.Fatalf("BaseURL = %q, want default", cfg.API.BaseURL)
	}
	if cfg.API.Timeout != "120s" {
		t.Fatalf("Timeout = %q, want 120s", cfg.API.Timeout)
	}
	timeout, err := APITimeout(cfg.API.Timeout)
	if err != nil {
		t.Fatalf("APITimeout returned error: %v", err)
	}
	if timeout != 120*time.Second {
		t.Fatalf("APITimeout = %s, want 120s", timeout)
	}
}

func TestWriteInitialAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Default()
	cfg.Auth.BearerToken = "secret-token"
	cfg.Auth.App = "md2x"

	if err := WriteInitial(path, cfg, false); err != nil {
		t.Fatalf("WriteInitial returned error: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}

	got, found, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if got.Auth.BearerToken != "secret-token" || got.Auth.App != "md2x" {
		t.Fatalf("loaded config = %#v", got)
	}
}

func TestApplyEnvOverridesConfig(t *testing.T) {
	t.Setenv("MD2X_API_BASE_URL", "https://api.example.test")
	t.Setenv("MD2X_HTTP_TIMEOUT", "45s")
	t.Setenv("X_BEARER_TOKEN", "env-token")
	t.Setenv("MD2X_APP", "env-app")
	t.Setenv("MD2X_USERNAME", "env-user")
	t.Setenv("MD2X_XURL_CONFIG", "/tmp/xurl.yaml")
	t.Setenv("MD2X_CLIENT_ID", "client-env")
	t.Setenv("MD2X_REDIRECT_URI", "http://127.0.0.1:9999/callback")
	t.Setenv("MD2X_AUTH_PROFILE", "work")

	cfg := Default()
	cfg.Auth.BearerToken = "file-token"
	cfg.Auth.App = "file-app"

	got := ApplyEnv(cfg)
	if got.API.BaseURL != "https://api.example.test" {
		t.Fatalf("BaseURL = %q", got.API.BaseURL)
	}
	if got.API.Timeout != "45s" {
		t.Fatalf("Timeout = %q, want 45s", got.API.Timeout)
	}
	if got.Auth.BearerToken != "env-token" || got.Auth.App != "env-app" || got.Auth.Username != "env-user" || got.Auth.XurlConfig != "/tmp/xurl.yaml" {
		t.Fatalf("env config = %#v", got.Auth)
	}
	if got.Auth.ClientID != "client-env" || got.Auth.RedirectURI != "http://127.0.0.1:9999/callback" || got.Auth.Profile != "work" {
		t.Fatalf("env oauth config = %#v", got.Auth)
	}
}

func TestAPITimeoutRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"0s", "-1s", "not-a-duration"} {
		t.Run(value, func(t *testing.T) {
			if _, err := APITimeout(value); err == nil {
				t.Fatal("APITimeout returned nil error, want error")
			}
		})
	}
}

func TestRedactHidesBearerToken(t *testing.T) {
	cfg := Default()
	cfg.Auth.BearerToken = "secret-token"
	cfg.Auth.ClientID = "client-123"

	got := Redact(cfg)
	if got.Auth.BearerToken != "<redacted>" {
		t.Fatalf("BearerToken = %q, want redacted", got.Auth.BearerToken)
	}
	if got.Auth.ClientID != "client-123" {
		t.Fatalf("ClientID = %q, want visible client id", got.Auth.ClientID)
	}
}

func TestDefaultAuthUsesOAuth2PKCE(t *testing.T) {
	cfg := Default()
	if cfg.Auth.Mode != "oauth2_pkce" {
		t.Fatalf("mode = %q, want oauth2_pkce", cfg.Auth.Mode)
	}
	if cfg.Auth.RedirectURI != "http://127.0.0.1:8765/callback" {
		t.Fatalf("redirect_uri = %q", cfg.Auth.RedirectURI)
	}
	if cfg.Auth.Profile != "default" {
		t.Fatalf("profile = %q, want default", cfg.Auth.Profile)
	}
	if len(cfg.Auth.Scopes) == 0 {
		t.Fatal("default scopes are empty")
	}
}
