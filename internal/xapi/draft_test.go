package xapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/geekjourneyx/md2x/internal/draftjs"
)

func TestNewClientDefaultsAndTrimsBaseURL(t *testing.T) {
	client := NewClient("", "token-123", nil)

	if client.baseURL != "https://api.x.com" {
		t.Fatalf("baseURL = %q, want https://api.x.com", client.baseURL)
	}
	if client.accessToken != "token-123" {
		t.Fatalf("accessToken = %q, want token-123", client.accessToken)
	}
	if client.httpClient == http.DefaultClient {
		t.Fatalf("httpClient = %#v, want package-owned default client", client.httpClient)
	}
	if client.httpClient.Timeout == 0 {
		t.Fatal("default httpClient timeout = 0, want finite timeout")
	}
	if client.httpClient.Timeout != 120*time.Second {
		t.Fatalf("default httpClient timeout = %s, want 120s", client.httpClient.Timeout)
	}

	customHTTPClient := &http.Client{}
	client = NewClient("https://example.test///", "token-456", customHTTPClient)
	if client.baseURL != "https://example.test" {
		t.Fatalf("baseURL = %q, want https://example.test", client.baseURL)
	}
	if client.httpClient != customHTTPClient {
		t.Fatalf("httpClient = %#v, want custom client", client.httpClient)
	}
}

func TestNewClientWithTimeoutUsesConfiguredTimeout(t *testing.T) {
	client := NewClientWithTimeout("", "token-123", nil, 45*time.Second)

	if client.httpClient.Timeout != 45*time.Second {
		t.Fatalf("httpClient timeout = %s, want 45s", client.httpClient.Timeout)
	}
}

func TestUploadImageRequestErrorIncludesTimeout(t *testing.T) {
	imagePath := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(imagePath, testPNG, 0o600); err != nil {
		t.Fatal(err)
	}

	client := NewClientWithTimeout("https://api.example.test", "token-123", &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		}),
	}, 2*time.Minute)

	_, err := client.UploadImage(imagePath)
	if err == nil {
		t.Fatal("UploadImage returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "timeout 2m0s") {
		t.Fatalf("error = %q, want configured timeout", err.Error())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestCreateDraftSendsArticlePayload(t *testing.T) {
	var gotTitle string
	var gotContentState draftjs.ContentState
	var gotCoverMedia *MediaRef

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2/articles/draft" {
			t.Fatalf("path = %q, want /2/articles/draft", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer token-123" {
			t.Fatalf("Authorization = %q, want Bearer token-123", r.Header.Get("Authorization"))
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			t.Fatalf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		var body struct {
			Title        string               `json:"title"`
			ContentState draftjs.ContentState `json:"content_state"`
			CoverMedia   *MediaRef            `json:"cover_media,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		gotTitle = body.Title
		gotContentState = body.ContentState
		gotCoverMedia = body.CoverMedia

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"1146654567674912769","title":"Hello"}}`))
	}))
	defer server.Close()

	contentState := &draftjs.ContentState{
		Blocks: []draftjs.Block{{Text: "Body", Type: "unstyled"}},
	}
	client := NewClient(server.URL, "token-123", server.Client())

	result, err := client.CreateDraft(CreateDraftRequest{
		Title:        "Hello",
		ContentState: contentState,
		CoverMedia:   &MediaRef{MediaCategory: "tweet_image", MediaID: "1146654567674912000"},
	})
	if err != nil {
		t.Fatalf("CreateDraft returned error: %v", err)
	}
	if result.ID != "1146654567674912769" {
		t.Fatalf("ID = %q, want 1146654567674912769", result.ID)
	}
	if result.Title != "Hello" {
		t.Fatalf("Title = %q, want Hello", result.Title)
	}
	if gotTitle != "Hello" {
		t.Fatalf("request title = %q, want Hello", gotTitle)
	}
	if len(gotContentState.Blocks) != 1 || gotContentState.Blocks[0].Text != "Body" {
		t.Fatalf("request content_state = %#v, want Body block", gotContentState)
	}
	if gotCoverMedia == nil {
		t.Fatalf("request cover_media = nil, want cover media")
	}
	if gotCoverMedia.MediaCategory != "tweet_image" || gotCoverMedia.MediaID != "1146654567674912000" {
		t.Fatalf("request cover_media = %#v, want tweet_image media id", gotCoverMedia)
	}
}

func TestCreateDraftNormalizesNilContentStateSlices(t *testing.T) {
	var rawBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		rawBody = string(data)

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"1146654567674912769","title":"Hello"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token-123", server.Client())
	_, err := client.CreateDraft(CreateDraftRequest{
		Title:        "Hello",
		ContentState: &draftjs.ContentState{},
	})
	if err != nil {
		t.Fatalf("CreateDraft returned error: %v", err)
	}
	if !strings.Contains(rawBody, `"blocks":[]`) {
		t.Fatalf("request body = %s, want blocks empty array", rawBody)
	}
	if !strings.Contains(rawBody, `"entities":[]`) {
		t.Fatalf("request body = %s, want entities empty array", rawBody)
	}
	if strings.Contains(rawBody, `null`) {
		t.Fatalf("request body = %s, want no null arrays", rawBody)
	}
}

func TestCreateDraftNonCreatedReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad draft"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token-123", server.Client())

	_, err := client.CreateDraft(CreateDraftRequest{
		Title:        "Hello",
		ContentState: &draftjs.ContentState{},
	})
	if err == nil {
		t.Fatal("CreateDraft returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "create draft returned 400 Bad Request") {
		t.Fatalf("error = %q, want status context", err.Error())
	}
}

func TestCreateDraftRateLimitErrorIncludesHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-rate-limit-limit", "10")
		w.Header().Set("x-rate-limit-remaining", "0")
		w.Header().Set("x-rate-limit-reset", "1893456000")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"title":"Too Many Requests","status":429}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token-123", server.Client())
	_, err := client.CreateDraft(CreateDraftRequest{
		Title:        "Hello",
		ContentState: &draftjs.ContentState{},
	})
	if err == nil {
		t.Fatal("CreateDraft returned nil error, want rate limit error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T, want APIError", err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("StatusCode = %d, want 429", apiErr.StatusCode)
	}
	if apiErr.RateLimit == nil {
		t.Fatal("RateLimit = nil, want parsed headers")
	}
	if apiErr.RateLimit.Limit != 10 || apiErr.RateLimit.Remaining != 0 || apiErr.RateLimit.ResetUnix != 1893456000 {
		t.Fatalf("RateLimit = %#v, want parsed limit headers", apiErr.RateLimit)
	}
	if apiErr.RateLimit.ResetAt != "2030-01-01T00:00:00Z" {
		t.Fatalf("ResetAt = %q, want 2030-01-01T00:00:00Z", apiErr.RateLimit.ResetAt)
	}
}

func TestCreateDraftMissingDataIDReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"title":"Hello"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token-123", server.Client())

	_, err := client.CreateDraft(CreateDraftRequest{
		Title:        "Hello",
		ContentState: &draftjs.ContentState{},
	})
	if err == nil {
		t.Fatal("CreateDraft returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "missing data.id") {
		t.Fatalf("error = %q, want missing data.id context", err.Error())
	}
}

func TestDraftResultMarshalUsesJSONTags(t *testing.T) {
	data, err := json.Marshal(DraftResult{
		ID:    "draft-123",
		Title: "Hello",
	})
	if err != nil {
		t.Fatalf("Marshal DraftResult: %v", err)
	}

	if got, want := string(data), `{"id":"draft-123","title":"Hello"}`; got != want {
		t.Fatalf("JSON = %s, want %s", got, want)
	}
}

func TestUploadImageUsesSingleStepMediaUpload(t *testing.T) {
	imagePath := filepath.Join(t.TempDir(), "cover.png")
	if err := os.WriteFile(imagePath, testPNG, 0600); err != nil {
		t.Fatalf("write image: %v", err)
	}

	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token-123" {
			t.Fatalf("Authorization = %q, want Bearer token-123", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/2/media/upload" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		calls = append(calls, "upload")
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if r.FormValue("media_category") != "tweet_image" {
			t.Fatalf("media_category = %q, want tweet_image", r.FormValue("media_category"))
		}
		if r.FormValue("media_type") != "image/png" {
			t.Fatalf("media_type = %q, want image/png", r.FormValue("media_type"))
		}
		file, header, err := r.FormFile("media")
		if err != nil {
			t.Fatalf("FormFile(media): %v", err)
		}
		_ = file.Close()
		if header.Filename != "cover.png" {
			t.Fatalf("filename = %q, want cover.png", header.Filename)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"media-123"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token-123", server.Client())

	result, err := client.UploadImage(imagePath)
	if err != nil {
		t.Fatalf("UploadImage returned error: %v", err)
	}
	if result.MediaID != "media-123" {
		t.Fatalf("MediaID = %q, want media-123", result.MediaID)
	}
	if result.MediaCategory != "tweet_image" {
		t.Fatalf("MediaCategory = %q, want tweet_image", result.MediaCategory)
	}
	if got, want := strings.Join(calls, ","), "upload"; got != want {
		t.Fatalf("calls = %q, want %q", got, want)
	}
}

func TestUploadImageSingleStepErrorReturnsAPIError(t *testing.T) {
	imagePath := writeTestImage(t)

	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/2/media/upload":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"upload failed"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token-123", server.Client())

	_, err := client.UploadImage(imagePath)
	if err == nil {
		t.Fatal("UploadImage returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "upload media returned 500 Internal Server Error") {
		t.Fatalf("error = %q, want upload status context", err.Error())
	}
	if got, want := strings.Join(calls, ","), "/2/media/upload"; got != want {
		t.Fatalf("calls = %q, want %q", got, want)
	}
}

func TestChunkedMediaUploadFunctionsRunInitializeAppendFinalize(t *testing.T) {
	imagePath := filepath.Join(t.TempDir(), "cover.png")
	if err := os.WriteFile(imagePath, testPNG, 0600); err != nil {
		t.Fatalf("write image: %v", err)
	}

	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token-123" {
			t.Fatalf("Authorization = %q, want Bearer token-123", r.Header.Get("Authorization"))
		}

		switch r.URL.Path {
		case "/2/media/upload/initialize":
			calls = append(calls, "initialize")
			if r.Method != http.MethodPost {
				t.Fatalf("initialize method = %q, want POST", r.Method)
			}
			var body struct {
				TotalBytes    int64  `json:"total_bytes"`
				MediaType     string `json:"media_type"`
				MediaCategory string `json:"media_category"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode initialize body: %v", err)
			}
			if body.TotalBytes != int64(len(testPNG)) {
				t.Fatalf("total_bytes = %d, want %d", body.TotalBytes, len(testPNG))
			}
			if body.MediaType != "image/png" {
				t.Fatalf("media_type = %q, want image/png", body.MediaType)
			}
			if body.MediaCategory != "tweet_image" {
				t.Fatalf("media_category = %q, want tweet_image", body.MediaCategory)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":"media-123"}}`))
		case "/2/media/upload/media-123/append":
			calls = append(calls, "append")
			if r.Method != http.MethodPost {
				t.Fatalf("append method = %q, want POST", r.Method)
			}
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("ParseMultipartForm: %v", err)
			}
			if r.FormValue("segment_index") != "0" {
				t.Fatalf("segment_index = %q, want 0", r.FormValue("segment_index"))
			}
			file, header, err := r.FormFile("media")
			if err != nil {
				t.Fatalf("FormFile(media): %v", err)
			}
			_ = file.Close()
			if header.Filename != "cover.png" {
				t.Fatalf("filename = %q, want cover.png", header.Filename)
			}
			w.WriteHeader(http.StatusNoContent)
		case "/2/media/upload/media-123/finalize":
			calls = append(calls, "finalize")
			if r.Method != http.MethodPost {
				t.Fatalf("finalize method = %q, want POST", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"id":"media-123"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token-123", server.Client())
	mediaID, err := client.initializeMediaUpload(int64(len(testPNG)), "image/png")
	if err != nil {
		t.Fatalf("initializeMediaUpload returned error: %v", err)
	}
	if mediaID != "media-123" {
		t.Fatalf("mediaID = %q, want media-123", mediaID)
	}
	if err := client.appendMediaUpload(mediaID, imagePath); err != nil {
		t.Fatalf("appendMediaUpload returned error: %v", err)
	}
	if err := client.finalizeMediaUpload(mediaID); err != nil {
		t.Fatalf("finalizeMediaUpload returned error: %v", err)
	}
	if got, want := strings.Join(calls, ","), "initialize,append,finalize"; got != want {
		t.Fatalf("calls = %q, want %q", got, want)
	}
}

func TestUploadImageFinalizeProcessingPollsStatusUntilSuccess(t *testing.T) {
	imagePath := writeTestImage(t)

	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.String())
		switch r.URL.Path {
		case "/2/media/upload":
			if r.Method == http.MethodGet {
				if r.URL.Query().Get("media_id") != "media-123" {
					t.Fatalf("media_id = %q, want media-123", r.URL.Query().Get("media_id"))
				}
				if r.URL.Query().Get("command") != "STATUS" {
					t.Fatalf("command = %q, want STATUS", r.URL.Query().Get("command"))
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data":{"id":"media-123","processing_info":{"state":"succeeded","check_after_secs":0,"progress_percent":100}}}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"id":"media-123","processing_info":{"state":"pending","check_after_secs":0,"progress_percent":10}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token-123", server.Client())

	result, err := client.UploadImage(imagePath)
	if err != nil {
		t.Fatalf("UploadImage returned error: %v", err)
	}
	if result.MediaID != "media-123" {
		t.Fatalf("MediaID = %q, want media-123", result.MediaID)
	}
	if got, want := strings.Join(calls, ","), "/2/media/upload,/2/media/upload?command=STATUS&media_id=media-123"; got != want {
		t.Fatalf("calls = %q, want %q", got, want)
	}
}

func TestUploadImageStatusFailureReturnsError(t *testing.T) {
	imagePath := writeTestImage(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2/media/upload":
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data":{"id":"media-123","processing_info":{"state":"in_progress","check_after_secs":0}}}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"id":"media-123","processing_info":{"state":"failed","check_after_secs":0,"progress_percent":40}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token-123", server.Client())

	_, err := client.UploadImage(imagePath)
	if err == nil {
		t.Fatal("UploadImage returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Fatalf("error = %q, want failed processing context", err.Error())
	}
}

func TestUploadImageStatusPendingEventuallyReturnsError(t *testing.T) {
	imagePath := writeTestImage(t)

	statusCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2/media/upload":
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data":{"id":"media-123","processing_info":{"state":"pending","check_after_secs":0}}}`))
				return
			}
			statusCalls++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"id":"media-123","processing_info":{"state":"in_progress","check_after_secs":0,"progress_percent":50}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token-123", server.Client())

	_, err := client.UploadImage(imagePath)
	if err == nil {
		t.Fatal("UploadImage returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "still processing") && !strings.Contains(err.Error(), "pending") {
		t.Fatalf("error = %q, want retryable pending context", err.Error())
	}
	if statusCalls == 0 {
		t.Fatal("statusCalls = 0, want polling")
	}
}

func TestUploadMediaResultMarshalUsesJSONTags(t *testing.T) {
	data, err := json.Marshal(UploadMediaResult{
		MediaID:       "media-123",
		MediaCategory: tweetImageCategory,
	})
	if err != nil {
		t.Fatalf("Marshal UploadMediaResult: %v", err)
	}

	if got, want := string(data), `{"media_id":"media-123","media_category":"tweet_image"}`; got != want {
		t.Fatalf("JSON = %s, want %s", got, want)
	}
}

func TestImageMediaType(t *testing.T) {
	tests := []struct {
		filePath string
		want     string
		wantErr  bool
	}{
		{filePath: "cover.jpg", want: "image/jpeg"},
		{filePath: "cover.jpeg", want: "image/jpeg"},
		{filePath: "cover.webp", want: "image/webp"},
		{filePath: "cover.png", want: "image/png"},
		{filePath: "cover.gif", wantErr: true},
		{filePath: "cover", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s maps to %s", tt.filePath, tt.want), func(t *testing.T) {
			got, err := imageMediaType(tt.filePath)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("imageMediaType(%q) returned nil error", tt.filePath)
				}
				if !strings.Contains(err.Error(), "unsupported image type") {
					t.Fatalf("error = %q, want unsupported image type context", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("imageMediaType(%q) returned error: %v", tt.filePath, err)
			}
			if got != tt.want {
				t.Fatalf("imageMediaType(%q) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestUploadImageLocalValidationErrorsAreTyped(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "missing file", path: filepath.Join(t.TempDir(), "missing.png")},
		{name: "unsupported extension", path: writeTestFile(t, "cover.gif", "gif bytes")},
		{name: "invalid image header", path: writeTestFile(t, "cover.png", "not really png")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("http://127.0.0.1:1", "token-123", nil)
			_, err := client.UploadImage(tt.path)
			if err == nil {
				t.Fatal("UploadImage returned nil error, want validation error")
			}
			var validationErr *MediaValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("error = %T %v, want MediaValidationError", err, err)
			}
			if validationErr.Path != tt.path {
				t.Fatalf("Path = %q, want %q", validationErr.Path, tt.path)
			}
		})
	}
}

func writeTestImage(t *testing.T) string {
	t.Helper()

	return writeTestFile(t, "cover.png", testPNG)
}

func writeTestFile(t *testing.T, name string, content interface{}) string {
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
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}

var testPNG = []byte{
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
