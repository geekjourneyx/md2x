package xapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/geekjourneyx/md2x/internal/mediafile"
)

const tweetImageCategory = "tweet_image"
const maxMediaProcessingStatusAttempts = 5

type UploadMediaResult struct {
	MediaID       string `json:"media_id"`
	MediaCategory string `json:"media_category"`
}

type MediaValidationError struct {
	Path string
	Err  error
}

func (e *MediaValidationError) Error() string {
	if e.Path == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("validate image %q: %v", e.Path, e.Err)
}

func (e *MediaValidationError) Unwrap() error {
	return e.Err
}

func (c *Client) UploadImage(filePath string) (*UploadMediaResult, error) {
	mediaType, _, err := mediafile.ValidateImageFile(filePath)
	if err != nil {
		return nil, &MediaValidationError{Path: filePath, Err: err}
	}
	mediaID, err := c.uploadMedia(filePath, mediaType)
	if err != nil {
		return nil, err
	}

	return &UploadMediaResult{
		MediaID:       mediaID,
		MediaCategory: tweetImageCategory,
	}, nil
}

func (c *Client) uploadMedia(filePath, mediaType string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("media_category", tweetImageCategory); err != nil {
		return "", fmt.Errorf("write media_category field: %w", err)
	}
	if err := writer.WriteField("media_type", mediaType); err != nil {
		return "", fmt.Errorf("write media_type field: %w", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open image %q: %w", filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("create media form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("write media form file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/2/media/upload", &body)
	if err != nil {
		return "", fmt.Errorf("create media upload request: %w", err)
	}
	c.authorize(req)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", c.requestError("upload media", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if !isSuccess(resp.StatusCode) {
		return "", apiError("upload media", resp)
	}

	var decoded struct {
		Data struct {
			ID             string               `json:"id"`
			ProcessingInfo *mediaProcessingInfo `json:"processing_info"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode media upload response: %w", err)
	}
	if decoded.Data.ID == "" {
		return "", fmt.Errorf("media upload response missing data.id")
	}
	if err := c.waitForMediaProcessing(decoded.Data.ID, decoded.Data.ProcessingInfo); err != nil {
		return "", err
	}
	return decoded.Data.ID, nil
}

func (c *Client) initializeMediaUpload(totalBytes int64, mediaType string) (string, error) {
	body, err := json.Marshal(struct {
		TotalBytes    int64  `json:"total_bytes"`
		MediaType     string `json:"media_type"`
		MediaCategory string `json:"media_category"`
	}{
		TotalBytes:    totalBytes,
		MediaType:     mediaType,
		MediaCategory: tweetImageCategory,
	})
	if err != nil {
		return "", fmt.Errorf("encode initialize media request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/2/media/upload/initialize", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create initialize media request: %w", err)
	}
	c.authorize(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("initialize media upload: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if !isSuccess(resp.StatusCode) {
		return "", apiError("initialize media upload", resp)
	}

	var decoded struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode initialize media response: %w", err)
	}
	if decoded.Data.ID == "" {
		return "", fmt.Errorf("initialize media response missing data.id")
	}
	return decoded.Data.ID, nil
}

func (c *Client) appendMediaUpload(mediaID, filePath string) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("segment_index", "0"); err != nil {
		return fmt.Errorf("write segment_index field: %w", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open image %q: %w", filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("create media form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("write media form file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/2/media/upload/"+mediaID+"/append", &body)
	if err != nil {
		return fmt.Errorf("create append media request: %w", err)
	}
	c.authorize(req)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("append media upload: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if !isSuccess(resp.StatusCode) {
		return apiError("append media upload", resp)
	}
	return nil
}

func (c *Client) finalizeMediaUpload(mediaID string) error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/2/media/upload/"+mediaID+"/finalize", nil)
	if err != nil {
		return fmt.Errorf("create finalize media request: %w", err)
	}
	c.authorize(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("finalize media upload: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if !isSuccess(resp.StatusCode) {
		return apiError("finalize media upload", resp)
	}

	var decoded struct {
		Data struct {
			ID             string               `json:"id"`
			ProcessingInfo *mediaProcessingInfo `json:"processing_info"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return fmt.Errorf("decode finalize media response: %w", err)
	}
	if decoded.Data.ID == "" {
		return fmt.Errorf("finalize media response missing data.id")
	}
	return c.waitForMediaProcessing(mediaID, decoded.Data.ProcessingInfo)
}

type mediaProcessingInfo struct {
	State           string `json:"state"`
	CheckAfterSecs  int    `json:"check_after_secs"`
	ProgressPercent int    `json:"progress_percent"`
}

func (c *Client) waitForMediaProcessing(mediaID string, info *mediaProcessingInfo) error {
	if mediaProcessingComplete(info) {
		return nil
	}
	if mediaProcessingFailed(info) {
		return fmt.Errorf("media upload %s processing failed", mediaID)
	}

	for attempt := 0; attempt < maxMediaProcessingStatusAttempts; attempt++ {
		if info != nil && info.CheckAfterSecs > 0 {
			time.Sleep(time.Duration(info.CheckAfterSecs) * time.Second)
		}

		nextInfo, err := c.mediaProcessingStatus(mediaID)
		if err != nil {
			return err
		}
		if mediaProcessingComplete(nextInfo) {
			return nil
		}
		if mediaProcessingFailed(nextInfo) {
			return fmt.Errorf("media upload %s processing failed", mediaID)
		}
		info = nextInfo
	}

	return fmt.Errorf("media upload %s is still processing after %d status attempts; retry later", mediaID, maxMediaProcessingStatusAttempts)
}

func (c *Client) mediaProcessingStatus(mediaID string) (*mediaProcessingInfo, error) {
	statusURL, err := url.Parse(c.baseURL + "/2/media/upload")
	if err != nil {
		return nil, fmt.Errorf("create media status URL: %w", err)
	}
	query := statusURL.Query()
	query.Set("media_id", mediaID)
	query.Set("command", "STATUS")
	statusURL.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, statusURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create media status request: %w", err)
	}
	c.authorize(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get media upload status: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if !isSuccess(resp.StatusCode) {
		return nil, apiError("media upload status", resp)
	}

	var decoded struct {
		Data struct {
			ID             string               `json:"id"`
			ProcessingInfo *mediaProcessingInfo `json:"processing_info"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode media status response: %w", err)
	}
	if decoded.Data.ID == "" {
		return nil, fmt.Errorf("media status response missing data.id")
	}
	return decoded.Data.ProcessingInfo, nil
}

func mediaProcessingComplete(info *mediaProcessingInfo) bool {
	if info == nil {
		return true
	}
	switch strings.ToLower(info.State) {
	case "succeeded", "complete", "completed":
		return true
	default:
		return false
	}
}

func mediaProcessingFailed(info *mediaProcessingInfo) bool {
	return info != nil && strings.ToLower(info.State) == "failed"
}

func imageMediaType(filePath string) (string, error) {
	return mediafile.ImageMediaType(filePath)
}

func isSuccess(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}
