package xapi

import (
	"net/http"
	"strings"
	"time"
)

const defaultHTTPTimeout = 30 * time.Second

type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

func NewClient(baseURL, accessToken string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = "https://api.x.com"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		accessToken: accessToken,
		httpClient:  httpClient,
	}
}

func (c *Client) authorize(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
}
