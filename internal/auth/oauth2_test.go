package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuthorizeURLIncludesPKCEParametersAndDefaultScopes(t *testing.T) {
	rawURL, err := AuthorizeURL("", "client-123", "", nil, "state-123", "challenge-123")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	if parsed.Scheme != "https" || parsed.Host != "x.com" || parsed.Path != "/i/oauth2/authorize" {
		t.Fatalf("authorize URL = %s", rawURL)
	}
	if q.Get("response_type") != "code" || q.Get("client_id") != "client-123" || q.Get("redirect_uri") != DefaultRedirectURI || q.Get("state") != "state-123" || q.Get("code_challenge") != "challenge-123" || q.Get("code_challenge_method") != "S256" {
		t.Fatalf("authorize query = %v", q)
	}
	if !strings.Contains(q.Get("scope"), "offline.access") {
		t.Fatalf("scope = %q, want offline.access", q.Get("scope"))
	}
}

func TestExchangeCodePostsPKCEFormAndReturnsToken(t *testing.T) {
	var form url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/2/oauth2/token" {
			t.Fatalf("request = %s %s, want POST /2/oauth2/token", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		form = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"token_type":    "bearer",
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"expires_in":    7200,
			"scope":         "tweet.read tweet.write",
		})
	}))
	defer server.Close()

	client := Client{
		APIBaseURL:  server.URL,
		HTTPClient:  server.Client(),
		Now:         func() time.Time { return time.Unix(1000, 0).UTC() },
		TokenLeeway: time.Minute,
	}
	token, err := client.ExchangeCode(context.Background(), ExchangeCodeRequest{
		ClientID:     "client-123",
		Code:         "code-123",
		CodeVerifier: "verifier-123",
		RedirectURI:  "http://127.0.0.1:8765/callback",
	})
	if err != nil {
		t.Fatal(err)
	}

	if form.Get("grant_type") != "authorization_code" || form.Get("client_id") != "client-123" || form.Get("code") != "code-123" || form.Get("code_verifier") != "verifier-123" {
		t.Fatalf("unexpected token exchange form: %v", form)
	}
	if token.AccessToken != "access-token" || token.RefreshToken != "refresh-token" || token.TokenType != "bearer" || token.Scope != "tweet.read tweet.write" {
		t.Fatalf("token = %#v", token)
	}
	if want := time.Unix(1000+7200, 0).UTC(); !token.ExpiresAt.Equal(want) {
		t.Fatalf("expires_at = %s, want %s", token.ExpiresAt, want)
	}
}

func TestResolveAccessTokenReturnsBearerBeforeStore(t *testing.T) {
	accessToken, err := ResolveAccessToken(context.Background(), ResolveOptions{
		BearerToken: " direct-token ",
		Store:       FileStore{Path: filepath.Join(t.TempDir(), "missing.json")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if accessToken != "direct-token" {
		t.Fatalf("access token = %q, want direct-token", accessToken)
	}
}

func TestRefreshReportsHTTPErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad refresh", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := Client{APIBaseURL: server.URL, HTTPClient: server.Client()}
	_, err := client.Refresh(context.Background(), RefreshRequest{
		ClientID:     "client-123",
		RefreshToken: "refresh-token",
	})
	if err == nil {
		t.Fatal("expected refresh error")
	}
	if !strings.Contains(err.Error(), "401 Unauthorized") || !strings.Contains(err.Error(), "bad refresh") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveAccessTokenRefreshesExpiredNativeToken(t *testing.T) {
	var form url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		form = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"token_type":    "bearer",
			"access_token":  "fresh-access",
			"refresh_token": "fresh-refresh",
			"expires_in":    7200,
		})
	}))
	defer server.Close()

	store := FileStore{Path: filepath.Join(t.TempDir(), "default.json")}
	if err := store.Save(Token{
		AccessToken:  "stale-access",
		RefreshToken: "stale-refresh",
		TokenType:    "bearer",
		ExpiresAt:    time.Unix(900, 0).UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	client := Client{
		APIBaseURL:  server.URL,
		HTTPClient:  server.Client(),
		Now:         func() time.Time { return time.Unix(1000, 0).UTC() },
		TokenLeeway: time.Minute,
	}
	accessToken, err := ResolveAccessToken(context.Background(), ResolveOptions{
		BearerToken: "",
		ClientID:    "client-123",
		Store:       store,
		Client:      client,
	})
	if err != nil {
		t.Fatal(err)
	}

	if accessToken != "fresh-access" {
		t.Fatalf("access token = %q, want fresh-access", accessToken)
	}
	if form.Get("grant_type") != "refresh_token" || form.Get("refresh_token") != "stale-refresh" || form.Get("client_id") != "client-123" {
		t.Fatalf("unexpected refresh form: %v", form)
	}
	saved, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.AccessToken != "fresh-access" || saved.RefreshToken != "fresh-refresh" {
		t.Fatalf("saved token = %#v", saved)
	}
}
