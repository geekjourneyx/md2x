package xapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/geekjourneyx/md2x/internal/draftjs"
)

type CreateDraftRequest struct {
	Title        string                `json:"title"`
	ContentState *draftjs.ContentState `json:"content_state"`
	CoverMedia   *MediaRef             `json:"cover_media,omitempty"`
}

type MediaRef struct {
	MediaCategory string `json:"media_category"`
	MediaID       string `json:"media_id"`
}

type DraftResult struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (c *Client) CreateDraft(input CreateDraftRequest) (*DraftResult, error) {
	input = normalizeCreateDraftRequest(input)

	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encode draft request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/2/articles/draft", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create draft request: %w", err)
	}
	c.authorize(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create draft: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		return nil, apiError("create draft", resp)
	}

	var decoded struct {
		Data struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode create draft response: %w", err)
	}
	if decoded.Data.ID == "" {
		return nil, fmt.Errorf("create draft response missing data.id")
	}

	return &DraftResult{
		ID:    decoded.Data.ID,
		Title: decoded.Data.Title,
	}, nil
}

func normalizeCreateDraftRequest(input CreateDraftRequest) CreateDraftRequest {
	if input.ContentState == nil {
		return input
	}
	contentState := *input.ContentState
	if contentState.Blocks == nil {
		contentState.Blocks = []draftjs.Block{}
	}
	if contentState.Entities == nil {
		contentState.Entities = []draftjs.Entity{}
	}
	input.ContentState = &contentState
	return input
}

func readErrorBody(r io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(r, 4096))
	if err != nil || len(data) == 0 {
		return ""
	}
	return string(data)
}
