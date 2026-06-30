package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDraftRequiresToken(t *testing.T) {
	dir := t.TempDir()
	articlePath := filepath.Join(dir, "article.md")
	writeFile(t, articlePath, "---\ntitle: Token Required\n---\n\nBody.\n")
	configPath := filepath.Join(t.TempDir(), ".xurl")
	cmd := md2xCommand(t, "draft", articlePath, "--json", "--xurl-config", configPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatalf("md2x draft succeeded, want exit 3")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("error = %T %v, want *exec.ExitError", err, err)
	}
	if got := exitErr.ExitCode(); got != 3 {
		t.Fatalf("exit code = %d, want 3\nstdout:\n%s\nstderr:\n%s", got, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var got struct {
		Success       bool   `json:"success"`
		SchemaVersion string `json:"schema_version"`
		Status        string `json:"status"`
		Code          string `json:"code"`
	}
	if err := json.NewDecoder(bytes.NewReader(stdout.Bytes())).Decode(&got); err != nil {
		t.Fatalf("decode stdout json: %v\nstdout:\n%s", err, stdout.String())
	}
	if got.Success {
		t.Fatalf("success = true, want false")
	}
	if got.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want %q", got.SchemaVersion, "v1")
	}
	if got.Status != "failed" {
		t.Fatalf("status = %q, want %q", got.Status, "failed")
	}
	if got.Code != "AUTH_TOKEN_NOT_FOUND" {
		t.Fatalf("code = %q, want %q\nstderr:\n%s", got.Code, "AUTH_TOKEN_NOT_FOUND", stderr.String())
	}
}

func TestDraftCreatesArticleWithUploadedMedia(t *testing.T) {
	dir := t.TempDir()
	articlePath := filepath.Join(dir, "article.md")
	writeFile(t, articlePath, `---
title: Integrated Draft
cover: ./cover.png
---

Intro text.

![Diagram](./diagram.png)
`)
	writeFile(t, filepath.Join(dir, "cover.png"), dummyPNG)
	writeFile(t, filepath.Join(dir, "diagram.png"), dummyPNG)
	configPath := writeXurlConfig(t, dir)

	var mu sync.Mutex
	var calls []string
	var draftBody map[string]interface{}
	mediaSeq := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/2/media/upload/initialize":
			mediaSeq++
			mediaID := fmt.Sprintf("media-%d", mediaSeq)
			calls = append(calls, "initialize:"+mediaID)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"data":{"id":%q}}`, mediaID)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/2/media/upload/") && strings.HasSuffix(r.URL.Path, "/append"):
			calls = append(calls, "append:"+mediaIDFromPath(r.URL.Path))
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("parse append multipart: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/2/media/upload/") && strings.HasSuffix(r.URL.Path, "/finalize"):
			mediaID := mediaIDFromPath(r.URL.Path)
			calls = append(calls, "finalize:"+mediaID)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"data":{"id":%q}}`, mediaID)
		case r.Method == http.MethodPost && r.URL.Path == "/2/articles/draft":
			calls = append(calls, "draft")
			if err := json.NewDecoder(r.Body).Decode(&draftBody); err != nil {
				t.Errorf("decode draft body: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":"article-123","title":"Integrated Draft"}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cmd := md2xCommand(t, "draft", articlePath, "--json", "--xurl-config", configPath, "--app", "md2x-test", "--api-base-url", server.URL)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("md2x draft failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var got struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Data    struct {
			ArticleID string `json:"article_id"`
			Title     string `json:"title"`
			Media     []struct {
				Role          string `json:"role"`
				Source        string `json:"source"`
				AssetIndex    *int   `json:"asset_index,omitempty"`
				MediaID       string `json:"media_id"`
				MediaCategory string `json:"media_category"`
			} `json:"media"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout:\n%s", err, stdout.String())
	}
	if !got.Success || got.Code != "OK" {
		t.Fatalf("success/code = %v/%q, want true/OK\nstdout:\n%s", got.Success, got.Code, stdout.String())
	}
	if got.Data.ArticleID != "article-123" || got.Data.Title != "Integrated Draft" {
		t.Fatalf("draft data = %+v, want article id/title", got.Data)
	}
	if len(got.Data.Media) != 2 {
		t.Fatalf("media len = %d, want 2: %+v", len(got.Data.Media), got.Data.Media)
	}
	if got.Data.Media[0].Role != "cover" || got.Data.Media[0].Source != "./cover.png" || got.Data.Media[0].MediaID != "media-1" || got.Data.Media[0].MediaCategory != "tweet_image" || got.Data.Media[0].AssetIndex != nil {
		t.Fatalf("cover media = %+v, want cover media-1 tweet_image without asset index", got.Data.Media[0])
	}
	if got.Data.Media[1].Role != "body" || got.Data.Media[1].Source != "./diagram.png" || got.Data.Media[1].MediaID != "media-2" || got.Data.Media[1].MediaCategory != "tweet_image" || got.Data.Media[1].AssetIndex == nil || *got.Data.Media[1].AssetIndex != 0 {
		t.Fatalf("body media = %+v, want diagram media-2 tweet_image asset index 0", got.Data.Media[1])
	}

	wantCalls := []string{"initialize:media-1", "append:media-1", "finalize:media-1", "initialize:media-2", "append:media-2", "finalize:media-2", "draft"}
	if strings.Join(calls, ",") != strings.Join(wantCalls, ",") {
		t.Fatalf("calls = %v, want %v", calls, wantCalls)
	}
	assertDraftCoverMedia(t, draftBody, "media-1")
	assertDraftBodyImageEntity(t, draftBody, "media-2")
}

func TestDraftUsesEnvBearerTokenAndHumanOutput(t *testing.T) {
	dir := t.TempDir()
	articlePath := filepath.Join(dir, "article.md")
	writeFile(t, articlePath, `---
title: Env Token Draft
---

Body only.
`)

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/2/articles/draft" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"article-env-123","title":"Env Token Draft"}}`))
	}))
	defer server.Close()

	missingConfigPath := filepath.Join(t.TempDir(), ".xurl")
	cmd := md2xCommand(t, "draft", articlePath, "--xurl-config", missingConfigPath, "--api-base-url", server.URL)
	cmd.Env = append(cmd.Env, "X_BEARER_TOKEN=env-token-123")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("md2x draft with X_BEARER_TOKEN failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if gotAuth != "Bearer env-token-123" {
		t.Fatalf("Authorization = %q, want Bearer env-token-123", gotAuth)
	}
	if !strings.Contains(stdout.String(), "created X Article draft article-env-123") {
		t.Fatalf("stdout = %q, want human draft success line", stdout.String())
	}
	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err == nil {
		t.Fatalf("stdout decoded as JSON envelope unexpectedly: %#v", envelope)
	}
}

func TestDraftUsesLocalConfigTokenAndAPIBaseURL(t *testing.T) {
	dir := t.TempDir()
	articlePath := filepath.Join(dir, "article.md")
	writeFile(t, articlePath, `---
title: Config Token Draft
---

Body only.
`)

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/2/articles/draft" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"article-config-123","title":"Config Token Draft"}}`))
	}))
	defer server.Close()

	configPath := filepath.Join(dir, "config.yaml")
	writeFile(t, configPath, fmt.Sprintf(`version: 1
api:
  base_url: %s
auth:
  bearer_token: config-token-123
`, server.URL))

	cmd := md2xCommand(t, "--config", configPath, "draft", articlePath, "--json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("md2x draft with config token failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if gotAuth != "Bearer config-token-123" {
		t.Fatalf("Authorization = %q, want Bearer config-token-123", gotAuth)
	}
	var got struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Data    struct {
			ArticleID string `json:"article_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout:\n%s", err, stdout.String())
	}
	if !got.Success || got.Code != "OK" || got.Data.ArticleID != "article-config-123" {
		t.Fatalf("unexpected stdout:\n%s", stdout.String())
	}
}

func TestDraftUsesNativeOAuthTokenStore(t *testing.T) {
	dir := t.TempDir()
	stateHome := filepath.Join(dir, "state")
	articlePath := filepath.Join(dir, "article.md")
	writeFile(t, articlePath, `---
title: Native Token Draft
---

Body only.
`)

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/2/articles/draft" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"article-native-123","title":"Native Token Draft"}}`))
	}))
	defer server.Close()

	configPath := filepath.Join(dir, "config.yaml")
	writeFile(t, configPath, fmt.Sprintf(`version: 1
api:
  base_url: %s
auth:
  client_id: client-123
  profile: default
`, server.URL))
	tokenPath := filepath.Join(stateHome, "md2x", "auth", "default.json")
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, tokenPath, fmt.Sprintf(`{
  "access_token": "native-access-token",
  "refresh_token": "native-refresh-token",
  "token_type": "bearer",
  "expires_at": %q
}
`, time.Now().Add(time.Hour).UTC().Format(time.RFC3339)))

	cmd := md2xCommand(t, "--config", configPath, "draft", articlePath, "--json")
	cmd.Env = append(cmd.Env, "XDG_STATE_HOME="+stateHome)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("md2x draft with native token failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if gotAuth != "Bearer native-access-token" {
		t.Fatalf("Authorization = %q, want native token", gotAuth)
	}
}

func TestDraftLocalMediaValidationFailuresExit2(t *testing.T) {
	tests := []struct {
		name       string
		cover      string
		createFile bool
	}{
		{name: "missing file", cover: "./missing.png"},
		{name: "unsupported extension", cover: "./cover.gif", createFile: true},
		{name: "invalid image header", cover: "./cover.png", createFile: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			articlePath := filepath.Join(dir, "article.md")
			writeFile(t, articlePath, fmt.Sprintf(`---
title: Bad Media
cover: %s
---

Body.
`, tt.cover))
			if tt.createFile {
				writeFile(t, filepath.Join(dir, strings.TrimPrefix(tt.cover, "./")), "not really an image")
			}
			configPath := writeXurlConfig(t, dir)

			cmd := md2xCommand(t, "draft", articlePath, "--json", "--xurl-config", configPath, "--app", "md2x-test", "--api-base-url", "http://127.0.0.1:1")
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			if err == nil {
				t.Fatalf("md2x draft succeeded, want exit 2")
			}
			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("error = %T %v, want *exec.ExitError", err, err)
			}
			if got := exitErr.ExitCode(); got != 2 {
				t.Fatalf("exit code = %d, want 2\nstdout:\n%s\nstderr:\n%s", got, stdout.String(), stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}

			var got struct {
				Success bool   `json:"success"`
				Code    string `json:"code"`
				Error   struct {
					Diagnostics []struct {
						Code string `json:"code"`
					} `json:"diagnostics"`
				} `json:"error"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal stdout: %v\nstdout:\n%s", err, stdout.String())
			}
			if got.Success {
				t.Fatalf("success = true, want false")
			}
			if got.Code != "INPUT_VALIDATION_FAILED" {
				t.Fatalf("code = %q, want INPUT_VALIDATION_FAILED\nstdout:\n%s", got.Code, stdout.String())
			}
			if len(got.Error.Diagnostics) == 0 {
				t.Fatalf("diagnostics missing from stdout:\n%s", stdout.String())
			}
		})
	}
}

func writeXurlConfig(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, ".xurl")
	expiration := time.Now().Add(24 * time.Hour).Unix()
	writeFile(t, path, fmt.Sprintf(`default_app: md2x-test
apps:
  md2x-test:
    default_user: default
    oauth2_tokens:
      default:
        type: oauth2
        oauth2:
          access_token: test-token
          refresh_token: refresh-token
          expiration_time: %d
`, expiration))
	return path
}

func writeFile(t *testing.T, path string, content interface{}) {
	t.Helper()
	var data []byte
	switch value := content.(type) {
	case string:
		data = []byte(value)
	case []byte:
		data = value
	default:
		t.Fatalf("unsupported test file content type %T", content)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mediaIDFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		return ""
	}
	return parts[4]
}

func assertDraftBodyImageEntity(t *testing.T, body map[string]interface{}, wantMediaID string) {
	t.Helper()
	contentState, ok := body["content_state"].(map[string]interface{})
	if !ok {
		t.Fatalf("content_state = %T, want object: %#v", body["content_state"], body["content_state"])
	}
	blocks, ok := contentState["blocks"].([]interface{})
	if !ok {
		t.Fatalf("blocks = %T, want array", contentState["blocks"])
	}
	entities, ok := contentState["entities"].([]interface{})
	if !ok {
		t.Fatalf("entities = %T, want array", contentState["entities"])
	}
	for _, rawBlock := range blocks {
		block, ok := rawBlock.(map[string]interface{})
		if !ok || block["type"] != "atomic" {
			continue
		}
		if _, ok := block["data"]; ok {
			t.Fatalf("atomic image block leaked data: %#v", block["data"])
		}
		ranges, _ := block["entity_ranges"].([]interface{})
		if len(ranges) == 0 {
			t.Fatalf("atomic block has no entity_ranges: %#v", block)
		}
		entity, ok := entities[0].(map[string]interface{})
		if !ok {
			t.Fatalf("entity[0] = %T, want object", entities[0])
		}
		value, _ := entity["value"].(map[string]interface{})
		if value["type"] != "image" {
			t.Fatalf("entity type = %v, want image", value["type"])
		}
		data, _ := value["data"].(map[string]interface{})
		mediaItems, _ := data["media_items"].([]interface{})
		if len(mediaItems) != 1 {
			t.Fatalf("media_items = %#v, want one item", data["media_items"])
		}
		mediaItem, _ := mediaItems[0].(map[string]interface{})
		if mediaItem["media_id"] != wantMediaID {
			t.Fatalf("media_id = %v, want %s", mediaItem["media_id"], wantMediaID)
		}
		return
	}
	t.Fatalf("draft body has no atomic image block: %#v", contentState)
}

func assertDraftCoverMedia(t *testing.T, body map[string]interface{}, wantMediaID string) {
	t.Helper()
	coverMedia, ok := body["cover_media"].(map[string]interface{})
	if !ok {
		t.Fatalf("cover_media = %T, want object: %#v", body["cover_media"], body["cover_media"])
	}
	if coverMedia["media_category"] != "tweet_image" {
		t.Fatalf("cover_media.media_category = %v, want tweet_image", coverMedia["media_category"])
	}
	if coverMedia["media_id"] != wantMediaID {
		t.Fatalf("cover_media.media_id = %v, want %s", coverMedia["media_id"], wantMediaID)
	}
}

var dummyPNG = []byte{
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
