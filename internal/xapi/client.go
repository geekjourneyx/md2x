package xapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultHTTPTimeout = 120 * time.Second

type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
	timeout     time.Duration
}

func NewClient(baseURL, accessToken string, httpClient *http.Client) *Client {
	return NewClientWithTimeout(baseURL, accessToken, httpClient, defaultHTTPTimeout)
}

func NewClientWithTimeout(baseURL, accessToken string, httpClient *http.Client, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = "https://api.x.com"
	}
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		accessToken: accessToken,
		httpClient:  httpClient,
		timeout:     timeout,
	}
}

func (c *Client) authorize(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
}

func (c *Client) requestError(operation string, err error) error {
	if c.timeout > 0 {
		return fmt.Errorf("%s (timeout %s): %w", operation, c.timeout, err)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
