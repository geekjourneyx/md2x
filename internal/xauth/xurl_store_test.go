package xauth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadXurlOAuth2TokenUsesDefaultAppAndUser(t *testing.T) {
	configPath := writeXurlConfig(t, `
apps:
  md2x-test:
    client_id: abc
    client_secret: def
    default_user: alice
    oauth2_tokens:
      alice:
        type: oauth2
        oauth2:
          access_token: token-123
          refresh_token: refresh-123
          expiration_time: 9999999999
default_app: md2x-test
`)

	token, err := LoadXurlOAuth2Token(configPath, "", "")
	if err != nil {
		t.Fatalf("LoadXurlOAuth2Token returned error: %v", err)
	}
	if token.AccessToken != "token-123" {
		t.Fatalf("AccessToken = %q, want token-123", token.AccessToken)
	}
	if token.RefreshToken != "refresh-123" {
		t.Fatalf("RefreshToken = %q, want refresh-123", token.RefreshToken)
	}
	if token.ExpirationTime != 9999999999 {
		t.Fatalf("ExpirationTime = %d, want 9999999999", token.ExpirationTime)
	}
}

func TestLoadXurlOAuth2TokenFallsBackToOnlyTokenUser(t *testing.T) {
	configPath := writeXurlConfig(t, fmt.Sprintf(`
apps:
  md2x-test:
    oauth2_tokens:
      alice:
        oauth2:
          access_token: token-123
          expiration_time: %d
default_app: md2x-test
`, time.Now().Add(time.Hour).Unix()))

	token, err := LoadXurlOAuth2Token(configPath, "", "")
	if err != nil {
		t.Fatalf("LoadXurlOAuth2Token returned error: %v", err)
	}
	if token.AccessToken != "token-123" {
		t.Fatalf("AccessToken = %q, want token-123", token.AccessToken)
	}
}

func TestLoadXurlOAuth2TokenErrorsForExpiredToken(t *testing.T) {
	configPath := writeXurlConfig(t, fmt.Sprintf(`
apps:
  md2x-test:
    default_user: alice
    oauth2_tokens:
      alice:
        oauth2:
          access_token: token-123
          expiration_time: %d
default_app: md2x-test
`, time.Now().Add(-time.Minute).Unix()))

	_, err := LoadXurlOAuth2Token(configPath, "", "")
	if err == nil {
		t.Fatal("LoadXurlOAuth2Token returned nil error")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("error = %q, want expired context", err.Error())
	}
	if !strings.Contains(err.Error(), "xurl auth oauth2 --app md2x-test") {
		t.Fatalf("error = %q, want refresh command with app", err.Error())
	}
}

func TestLoadXurlOAuth2TokenErrorsForNearExpiredToken(t *testing.T) {
	configPath := writeXurlConfig(t, fmt.Sprintf(`
apps:
  md2x-test:
    default_user: alice
    oauth2_tokens:
      alice:
        oauth2:
          access_token: token-123
          expiration_time: %d
default_app: md2x-test
`, time.Now().Add(30*time.Second).Unix()))

	_, err := LoadXurlOAuth2Token(configPath, "", "")
	if err == nil {
		t.Fatal("LoadXurlOAuth2Token returned nil error")
	}
	if !strings.Contains(err.Error(), "expires too soon") && !strings.Contains(err.Error(), "expired") {
		t.Fatalf("error = %q, want near-expiration context", err.Error())
	}
	if !strings.Contains(err.Error(), "xurl auth oauth2 --app md2x-test") {
		t.Fatalf("error = %q, want refresh command with app", err.Error())
	}
}

func TestLoadXurlOAuth2TokenErrorsForZeroExpirationTime(t *testing.T) {
	configPath := writeXurlConfig(t, `
apps:
  md2x-test:
    default_user: alice
    oauth2_tokens:
      alice:
        oauth2:
          access_token: token-123
          expiration_time: 0
default_app: md2x-test
`)

	_, err := LoadXurlOAuth2Token(configPath, "", "")
	if err == nil {
		t.Fatal("LoadXurlOAuth2Token returned nil error")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("error = %q, want expiration context", err.Error())
	}
	if !strings.Contains(err.Error(), "xurl auth oauth2 --app md2x-test") {
		t.Fatalf("error = %q, want refresh command with app", err.Error())
	}
}

func TestLoadXurlOAuth2TokenErrorsForMissingApp(t *testing.T) {
	configPath := writeXurlConfig(t, `
apps:
  md2x-test:
    default_user: alice
default_app: md2x-test
`)

	_, err := LoadXurlOAuth2Token(configPath, "missing", "")
	if err == nil {
		t.Fatal("LoadXurlOAuth2Token returned nil error")
	}
	if !strings.Contains(err.Error(), `app "missing"`) {
		t.Fatalf("error = %q, want missing app context", err.Error())
	}
}

func TestLoadXurlOAuth2TokenErrorsForMissingUser(t *testing.T) {
	configPath := writeXurlConfig(t, `
apps:
  md2x-test:
    oauth2_tokens:
      alice:
        oauth2:
          access_token: token-123
default_app: md2x-test
`)

	_, err := LoadXurlOAuth2Token(configPath, "", "bob")
	if err == nil {
		t.Fatal("LoadXurlOAuth2Token returned nil error")
	}
	if !strings.Contains(err.Error(), `user "bob"`) {
		t.Fatalf("error = %q, want missing user context", err.Error())
	}
}

func TestLoadXurlOAuth2TokenErrorsForEmptyAccessToken(t *testing.T) {
	configPath := writeXurlConfig(t, `
apps:
  md2x-test:
    default_user: alice
    oauth2_tokens:
      alice:
        oauth2:
          refresh_token: refresh-123
default_app: md2x-test
`)

	_, err := LoadXurlOAuth2Token(configPath, "", "")
	if err == nil {
		t.Fatal("LoadXurlOAuth2Token returned nil error")
	}
	if !strings.Contains(err.Error(), "access token") {
		t.Fatalf("error = %q, want access token context", err.Error())
	}
}

func writeXurlConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".xurl")
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatalf("write .xurl: %v", err)
	}
	return path
}
