package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/geekjourneyx/md2x/internal/article"
	"github.com/geekjourneyx/md2x/internal/auth"
	md2xconfig "github.com/geekjourneyx/md2x/internal/config"
	"github.com/geekjourneyx/md2x/internal/xapi"
	"github.com/spf13/cobra"
)

func executeCommand(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func writeArticle(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "article.md")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestVersionCommandWritesJSONEnvelope(t *testing.T) {
	cmd, _ := newRootCommand()
	out, err := executeCommand(t, cmd, "--json", "version")
	if err != nil {
		t.Fatal(err)
	}

	var env Envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	if !env.Success || env.Code != "OK" || env.Message != "md2x version" {
		t.Fatalf("unexpected envelope: %#v", env)
	}
}

func TestConfigInitAndShowRedactsToken(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	cmd, _ := newRootCommand()

	out, err := executeCommand(t, cmd, "--config", configPath, "--json", "config", "init", "--bearer-token", "secret-token", "--app", "md2x", "--client-id", "client-123", "--api-timeout", "75s")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "secret-token") {
		t.Fatalf("config init leaked token:\n%s", out)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "secret-token") {
		t.Fatalf("config file did not store token:\n%s", data)
	}

	cmd, _ = newRootCommand()
	out, err = executeCommand(t, cmd, "--config", configPath, "--json", "config", "show")
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data struct {
			Config struct {
				API struct {
					Timeout string `json:"timeout"`
				} `json:"api"`
				Auth struct {
					BearerToken string `json:"bearer_token"`
					ClientID    string `json:"client_id"`
					RedirectURI string `json:"redirect_uri"`
					Profile     string `json:"profile"`
				} `json:"auth"`
			} `json:"config"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("unmarshal config show: %v\n%s", err, out)
	}
	if strings.Contains(out, "secret-token") || envelope.Data.Config.Auth.BearerToken != "<redacted>" {
		t.Fatalf("config show redaction failed: %#v\n%s", envelope.Data.Config.Auth, out)
	}
	if envelope.Data.Config.API.Timeout != "75s" {
		t.Fatalf("api.timeout = %q, want 75s", envelope.Data.Config.API.Timeout)
	}
	if envelope.Data.Config.Auth.ClientID != "client-123" {
		t.Fatalf("client_id = %q, want client-123", envelope.Data.Config.Auth.ClientID)
	}
	if envelope.Data.Config.Auth.RedirectURI != auth.DefaultRedirectURI || envelope.Data.Config.Auth.Profile != auth.DefaultProfile {
		t.Fatalf("oauth defaults = %#v", envelope.Data.Config.Auth)
	}
}

func TestDraftRejectsInvalidAPITimeoutBeforeAuth(t *testing.T) {
	articlePath := writeArticle(t, "# Timeout Test\n\nBody")
	cmd, _ := newRootCommand()

	_, err := executeCommand(t, cmd, "--json", "draft", articlePath, "--api-timeout", "0s")
	if err == nil {
		t.Fatal("draft returned nil error, want timeout validation error")
	}
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("error = %T, want *ExitError", err)
	}
	if exitErr.Code != "API_TIMEOUT_INVALID" {
		t.Fatalf("code = %q, want API_TIMEOUT_INVALID", exitErr.Code)
	}
	if !strings.Contains(exitErr.Message, "api.timeout") {
		t.Fatalf("message = %q, want api.timeout detail", exitErr.Message)
	}
}

func TestAuthStatusReportsNativeTokenWithoutLeakingSecrets(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	store, err := auth.NewDefaultStore("default")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(auth.Token{
		AccessToken:  "access-secret",
		RefreshToken: "refresh-secret",
		TokenType:    "bearer",
		Scope:        "tweet.read tweet.write",
		ExpiresAt:    time.Now().Add(time.Hour).UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	cmd, _ := newRootCommand()
	out, err := executeCommand(t, cmd, "--json", "auth", "status")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "access-secret") || strings.Contains(out, "refresh-secret") {
		t.Fatalf("auth status leaked token:\n%s", out)
	}
	var envelope struct {
		Data struct {
			Profile string `json:"profile"`
			Status  struct {
				Authenticated bool   `json:"authenticated"`
				AccessToken   string `json:"access_token"`
				RefreshToken  string `json:"refresh_token"`
			} `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("unmarshal auth status: %v\n%s", err, out)
	}
	if envelope.Data.Profile != "default" || !envelope.Data.Status.Authenticated || envelope.Data.Status.AccessToken != "<redacted>" || envelope.Data.Status.RefreshToken != "<redacted>" {
		t.Fatalf("unexpected auth status: %#v", envelope.Data)
	}
}

func TestAuthStatusUnauthenticatedIsSuccessfulJSONStatus(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	cmd, _ := newRootCommand()

	out, err := executeCommand(t, cmd, "--json", "auth", "status")
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Status struct {
				Authenticated bool `json:"authenticated"`
			} `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("unmarshal auth status: %v\n%s", err, out)
	}
	if !envelope.Success || envelope.Data.Status.Authenticated {
		t.Fatalf("unexpected unauthenticated status:\n%s", out)
	}
}

func TestRunOAuthLoginCompletesPKCECallbackAndStoresToken(t *testing.T) {
	var gotForm string
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/2/oauth2/token" {
			t.Fatalf("request = %s %s, want POST /2/oauth2/token", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		gotForm = r.PostForm.Encode()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token_type":"bearer","access_token":"login-access","refresh_token":"login-refresh","expires_in":7200,"scope":"tweet.read tweet.write"}`))
	}))
	defer tokenServer.Close()

	redirectURI := freeLocalRedirectURI(t)
	store := auth.FileStore{Path: filepath.Join(t.TempDir(), "default.json")}
	writer := &authorizeURLWriter{ready: make(chan string, 1)}
	result := make(chan error, 1)
	var token auth.Token
	go func() {
		var err error
		token, err = runOAuthLogin(
			context.Background(),
			testOAuthConfig(tokenServer.URL, redirectURI),
			store,
			auth.DefaultAuthorizeURL,
			true,
			5*time.Second,
			writer,
		)
		result <- err
	}()

	authorizeURL := <-writer.ready
	state := queryValue(t, authorizeURL, "state")
	callbackURL := redirectURI + "?code=code-123&state=" + state
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback request: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "authorization complete") {
		t.Fatalf("callback page claims login is complete before token exchange: %s", body)
	}
	if !strings.Contains(string(body), "received the authorization callback") {
		t.Fatalf("callback page = %q, want callback received message", body)
	}

	if err := <-result; err != nil {
		t.Fatal(err)
	}
	writer.mu.Lock()
	progress := writer.buf.String()
	writer.mu.Unlock()
	if !strings.Contains(progress, "OAuth callback received; exchanging code for token") {
		t.Fatalf("progress output missing token exchange stage:\n%s", progress)
	}
	if !strings.Contains(progress, "OAuth token saved to ") {
		t.Fatalf("progress output missing token store stage:\n%s", progress)
	}
	if token.AccessToken != "login-access" || token.RefreshToken != "login-refresh" {
		t.Fatalf("token = %#v", token)
	}
	if !strings.Contains(gotForm, "grant_type=authorization_code") || !strings.Contains(gotForm, "code=code-123") {
		t.Fatalf("unexpected token exchange form: %s", gotForm)
	}
	saved, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.AccessToken != "login-access" {
		t.Fatalf("saved token = %#v", saved)
	}
}

func TestAuthRefreshCommandUpdatesStoredToken(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.PostForm.Get("grant_type") != "refresh_token" || r.PostForm.Get("refresh_token") != "old-refresh" {
			t.Fatalf("refresh form = %v", r.PostForm)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token_type":"bearer","access_token":"new-access","refresh_token":"new-refresh","expires_in":7200}`))
	}))
	defer tokenServer.Close()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(fmt.Sprintf("version: 1\napi:\n  base_url: %s\nauth:\n  client_id: client-123\n", tokenServer.URL)), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := auth.NewDefaultStore("default")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(auth.Token{AccessToken: "old-access", RefreshToken: "old-refresh", TokenType: "bearer", ExpiresAt: time.Now().Add(-time.Hour)}); err != nil {
		t.Fatal(err)
	}

	cmd, _ := newRootCommand()
	out, err := executeCommand(t, cmd, "--config", configPath, "--json", "auth", "refresh")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "new-access") || strings.Contains(out, "new-refresh") {
		t.Fatalf("auth refresh leaked token:\n%s", out)
	}
	saved, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.AccessToken != "new-access" || saved.RefreshToken != "new-refresh" {
		t.Fatalf("saved token = %#v", saved)
	}
}

func TestAuthLogoutDeletesStoredToken(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	store, err := auth.NewDefaultStore("default")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(auth.Token{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}

	cmd, _ := newRootCommand()
	if _, err := executeCommand(t, cmd, "--json", "auth", "logout"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, auth.ErrNotAuthenticated) {
		t.Fatalf("store.Load err = %v, want ErrNotAuthenticated", err)
	}
}

func TestConfigPathUsesEnvOverride(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("MD2X_CONFIG", configPath)
	cmd, _ := newRootCommand()

	out, err := executeCommand(t, cmd, "config", "path")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != configPath {
		t.Fatalf("config path = %q, want %q", strings.TrimSpace(out), configPath)
	}
}

func TestInspectCommandReadsArticle(t *testing.T) {
	path := writeArticle(t, "---\ntitle: Test Article\n---\n\n# Test Article\n\nBody.\n")
	cmd, _ := newRootCommand()

	out, err := executeCommand(t, cmd, "--json", "inspect", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"ready": true`) {
		t.Fatalf("inspect output did not mark article ready:\n%s", out)
	}
}

func TestInspectCommandEstimatesUniqueMediaRequests(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "shared.png")
	if err := os.WriteFile(imagePath, cliTestPNG, 0o600); err != nil {
		t.Fatal(err)
	}
	articlePath := filepath.Join(dir, "article.md")
	if err := os.WriteFile(articlePath, []byte("---\ntitle: Media Estimate\ncover: ./shared.png\n---\n\n![Again](./shared.png)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd, _ := newRootCommand()

	out, err := executeCommand(t, cmd, "--json", "inspect", articlePath)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data struct {
			UniqueMediaCount   int `json:"unique_media_count"`
			EstimatedXRequests struct {
				MediaUpload int `json:"media_upload"`
				CreateDraft int `json:"create_draft"`
				Total       int `json:"total"`
			} `json:"estimated_x_requests"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("unmarshal inspect output: %v\n%s", err, out)
	}
	if envelope.Data.UniqueMediaCount != 1 {
		t.Fatalf("unique_media_count = %d, want 1\n%s", envelope.Data.UniqueMediaCount, out)
	}
	if envelope.Data.EstimatedXRequests.MediaUpload != 1 || envelope.Data.EstimatedXRequests.CreateDraft != 1 || envelope.Data.EstimatedXRequests.Total != 2 {
		t.Fatalf("estimated_x_requests = %#v, want 1 upload + 1 draft", envelope.Data.EstimatedXRequests)
	}
}

func TestRenderCommandRejectsUnsupportedFormat(t *testing.T) {
	path := writeArticle(t, "---\ntitle: Test Article\n---\n\n# Test Article\n")
	cmd, _ := newRootCommand()

	_, err := executeCommand(t, cmd, "render", path, "--format", "html")
	if err == nil {
		t.Fatal("expected unsupported format error")
	}
	exitErr := &ExitError{}
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.Code != "UNSUPPORTED_FORMAT" || ExitCode(err) != 2 {
		t.Fatalf("unexpected error: %#v", exitErr)
	}
}

func TestDraftCommandFailsWithoutToken(t *testing.T) {
	t.Setenv("X_BEARER_TOKEN", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	path := writeArticle(t, "---\ntitle: Test Article\n---\n\n# Test Article\n")
	cmd, _ := newRootCommand()

	_, err := executeCommand(t, cmd, "draft", path, "--app", "md2x", "--xurl-config", filepath.Join(t.TempDir(), ".xurl"))
	if err == nil {
		t.Fatal("expected missing token error")
	}
	exitErr := &ExitError{}
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.Code != "AUTH_TOKEN_NOT_FOUND" || ExitCode(err) != 3 {
		t.Fatalf("unexpected error: %#v", exitErr)
	}
}

func TestDraftCommandValidatesInputBeforeToken(t *testing.T) {
	cmd, _ := newRootCommand()

	_, err := executeCommand(t, cmd, "draft", filepath.Join(t.TempDir(), "missing.md"), "--app", "md2x", "--xurl-config", filepath.Join(t.TempDir(), ".xurl"))
	if err == nil {
		t.Fatal("expected missing input error")
	}
	exitErr := &ExitError{}
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.Code != "INPUT_READ_FAILED" || ExitCode(err) != 2 {
		t.Fatalf("unexpected error: %#v", exitErr)
	}
}

func TestDraftCommandRejectsInvalidImageBeforeToken(t *testing.T) {
	dir := t.TempDir()
	articlePath := filepath.Join(dir, "article.md")
	if err := os.WriteFile(articlePath, []byte("---\ntitle: Bad Image\ncover: ./cover.png\n---\n\nBody.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cover.png"), []byte("not really png"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd, _ := newRootCommand()

	_, err := executeCommand(t, cmd, "draft", articlePath, "--app", "md2x", "--xurl-config", filepath.Join(t.TempDir(), ".xurl"))
	if err == nil {
		t.Fatal("expected invalid image error")
	}
	exitErr := &ExitError{}
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.Code != "INPUT_VALIDATION_FAILED" || ExitCode(err) != 2 {
		t.Fatalf("unexpected error: %#v", exitErr)
	}
	if len(exitErr.Diagnostics) == 0 || exitErr.Diagnostics[0].Code != "INVALID_IMAGE_FILE" {
		t.Fatalf("diagnostics = %#v, want INVALID_IMAGE_FILE", exitErr.Diagnostics)
	}
}

func TestInspectCommandMarksMissingMediaNotReady(t *testing.T) {
	path := writeArticle(t, "---\ntitle: Bad Media\ncover: ./missing.png\n---\n\nBody.\n")
	cmd, _ := newRootCommand()

	out, err := executeCommand(t, cmd, "--json", "inspect", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"ready": false`) {
		t.Fatalf("inspect output did not mark article not ready:\n%s", out)
	}
	if !strings.Contains(out, `"code": "MEDIA_NOT_FOUND"`) {
		t.Fatalf("inspect output did not include media diagnostic:\n%s", out)
	}
}

func TestJSONFlagRequestedStopsAtDoubleDash(t *testing.T) {
	if !jsonFlagRequested([]string{"inspect", "--json=true"}) {
		t.Fatal("expected --json=true to request JSON")
	}
	if jsonFlagRequested([]string{"inspect", "--", "--json"}) {
		t.Fatal("did not expect args after -- to request JSON")
	}
	if jsonFlagRequested([]string{"inspect", "--json=false"}) {
		t.Fatal("did not expect --json=false to request JSON")
	}
}

func TestFailureEnvelopeDefaultsCodeAndMessage(t *testing.T) {
	env := failureEnvelope(&ExitError{Err: errors.New("boom")})
	if env.Code != "ERROR" || env.Message != "boom" || env.Status != "failed" {
		t.Fatalf("unexpected failure envelope: %#v", env)
	}
}

func TestFailureEnvelopeIncludesDiagnostics(t *testing.T) {
	env := failureEnvelope(&ExitError{
		Code:    "INPUT_VALIDATION_FAILED",
		Message: "bad input",
		Diagnostics: []article.Diagnostic{
			{Severity: "error", Code: "INVALID_IMAGE_FILE", Message: "invalid image"},
		},
	})

	errorData, ok := env.Error.(map[string]interface{})
	if !ok {
		t.Fatalf("Error = %T, want map", env.Error)
	}
	if errorData["diagnostics"] == nil {
		t.Fatalf("diagnostics missing from %#v", errorData)
	}
}

func TestFailureEnvelopeIncludesDetails(t *testing.T) {
	env := failureEnvelope(&ExitError{
		Code:    "X_DRAFT_FAILED",
		Message: "rate limited",
		Details: map[string]interface{}{
			"x_api": map[string]interface{}{"status_code": 429, "retryable": true},
		},
	})

	errorData, ok := env.Error.(map[string]interface{})
	if !ok {
		t.Fatalf("Error = %T, want map", env.Error)
	}
	if errorData["x_api"] == nil {
		t.Fatalf("x_api details missing from %#v", errorData)
	}
}

func TestXAPIErrorDetailsMarksRateLimitRetryable(t *testing.T) {
	details := xAPIErrorDetails(&xapi.APIError{
		Operation:  "create draft",
		Status:     "429 Too Many Requests",
		StatusCode: 429,
		RateLimit:  &xapi.RateLimitInfo{Limit: 10, Remaining: 0, ResetUnix: 1893456000},
	})

	xapiDetails, ok := details["x_api"].(map[string]interface{})
	if !ok {
		t.Fatalf("details = %#v, want x_api map", details)
	}
	if xapiDetails["retryable"] != true {
		t.Fatalf("retryable = %#v, want true", xapiDetails["retryable"])
	}
	if xapiDetails["rate_limit"] == nil {
		t.Fatalf("rate_limit missing from %#v", xapiDetails)
	}
}

func TestAttachUploadedMediaMutatesMatchingImageBlock(t *testing.T) {
	doc := &article.Document{
		Blocks: []article.Block{
			{Type: "paragraph"},
			{Type: "image", Data: map[string]string{"asset_index": "2"}},
		},
	}

	ok := attachUploadedMedia(doc, 2, &xapi.UploadMediaResult{
		MediaID:       "media-1",
		MediaCategory: "tweet_image",
	})
	if !ok {
		t.Fatal("expected media to attach")
	}
	if got := doc.Blocks[1].Data["media_id"]; got != "media-1" {
		t.Fatalf("media_id = %q", got)
	}
	if attachUploadedMedia(doc, 3, &xapi.UploadMediaResult{}) {
		t.Fatal("did not expect unmatched media to attach")
	}
}

func TestResolveArticlePath(t *testing.T) {
	absolute := filepath.Join(t.TempDir(), "image.png")
	if got := resolveArticlePath("/base", absolute); got != absolute {
		t.Fatalf("absolute path changed: %q", got)
	}
	if got := resolveArticlePath("/base", "image.png"); got != filepath.Join("/base", "image.png") {
		t.Fatalf("relative path = %q", got)
	}
}

type authorizeURLWriter struct {
	mu       sync.Mutex
	buf      strings.Builder
	ready    chan string
	signaled bool
}

func (w *authorizeURLWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.buf.Write(data)
	if !w.signaled {
		text := w.buf.String()
		for _, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "https://") || strings.HasPrefix(line, "http://") {
				w.signaled = true
				w.ready <- line
				break
			}
		}
	}
	return n, err
}

func freeLocalRedirectURI(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return "http://" + addr + "/callback"
}

func queryValue(t *testing.T, rawURL, key string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	value := parsed.Query().Get(key)
	if value == "" {
		t.Fatalf("%s missing from %s", key, rawURL)
	}
	return value
}

func testOAuthConfig(apiBaseURL, redirectURI string) md2xconfig.Config {
	cfg := md2xconfig.Default()
	cfg.API.BaseURL = apiBaseURL
	cfg.Auth.ClientID = "client-123"
	cfg.Auth.RedirectURI = redirectURI
	cfg.Auth.Scopes = []string{"tweet.read", "tweet.write", "offline.access"}
	return cfg
}

var cliTestPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
	0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
	0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
	0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xdd, 0x8d,
	0xb0, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
	0x44, 0xae, 0x42, 0x60, 0x82,
}
