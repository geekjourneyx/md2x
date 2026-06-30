package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultAuthorizeURL = "https://x.com/i/oauth2/authorize"
	DefaultRedirectURI  = "http://127.0.0.1:8765/callback"
)

var DefaultScopes = []string{"tweet.read", "tweet.write", "users.read", "media.write", "offline.access"}

var ErrNotAuthenticated = errors.New("not authenticated; run md2x auth login")

type Client struct {
	APIBaseURL  string
	HTTPClient  *http.Client
	Now         func() time.Time
	TokenLeeway time.Duration
}

type ExchangeCodeRequest struct {
	ClientID     string
	Code         string
	CodeVerifier string
	RedirectURI  string
}

type RefreshRequest struct {
	ClientID     string
	RefreshToken string
}

type tokenResponse struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
}

func AuthorizeURL(authorizeBaseURL, clientID, redirectURI string, scopes []string, state, codeChallenge string) (string, error) {
	if strings.TrimSpace(authorizeBaseURL) == "" {
		authorizeBaseURL = DefaultAuthorizeURL
	}
	if strings.TrimSpace(clientID) == "" {
		return "", fmt.Errorf("client_id is required")
	}
	if strings.TrimSpace(redirectURI) == "" {
		redirectURI = DefaultRedirectURI
	}
	if len(scopes) == 0 {
		scopes = DefaultScopes
	}
	u, err := url.Parse(authorizeBaseURL)
	if err != nil {
		return "", fmt.Errorf("parse authorize URL: %w", err)
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c Client) ExchangeCode(ctx context.Context, req ExchangeCodeRequest) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", req.ClientID)
	form.Set("code", req.Code)
	form.Set("code_verifier", req.CodeVerifier)
	form.Set("redirect_uri", req.RedirectURI)
	return c.postTokenForm(ctx, form)
}

func (c Client) Refresh(ctx context.Context, req RefreshRequest) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", req.ClientID)
	form.Set("refresh_token", req.RefreshToken)
	return c.postTokenForm(ctx, form)
}

func (c Client) postTokenForm(ctx context.Context, form url.Values) (Token, error) {
	endpoint := strings.TrimRight(c.apiBaseURL(), "/") + "/2/oauth2/token"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, fmt.Errorf("create oauth token request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient().Do(httpReq)
	if err != nil {
		return Token{}, fmt.Errorf("request oauth token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Token{}, fmt.Errorf("oauth token endpoint returned %s: %s", resp.Status, readSmallBody(resp.Body))
	}

	var payload tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Token{}, fmt.Errorf("decode oauth token response: %w", err)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return Token{}, fmt.Errorf("oauth token response missing access_token")
	}
	expiresAt := time.Time{}
	if payload.ExpiresIn > 0 {
		expiresAt = c.now().Add(time.Duration(payload.ExpiresIn) * time.Second).UTC()
	}
	return Token{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		TokenType:    payload.TokenType,
		Scope:        payload.Scope,
		ExpiresAt:    expiresAt,
	}, nil
}

func (c Client) apiBaseURL() string {
	if strings.TrimSpace(c.APIBaseURL) == "" {
		return "https://api.x.com"
	}
	return c.APIBaseURL
}

func (c Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c Client) leeway() time.Duration {
	if c.TokenLeeway > 0 {
		return c.TokenLeeway
	}
	return tokenExpiryLeeway
}

func readSmallBody(r io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(r, 4096))
	if err != nil {
		return "<read error>"
	}
	return strings.TrimSpace(string(data))
}

type ResolveOptions struct {
	BearerToken string
	ClientID    string
	Store       FileStore
	Client      Client
}

func ResolveAccessToken(ctx context.Context, opts ResolveOptions) (string, error) {
	if token := strings.TrimSpace(opts.BearerToken); token != "" {
		return token, nil
	}
	token, err := opts.Store.Load()
	if err != nil {
		return "", err
	}
	if !token.Expired(opts.Client.now(), opts.Client.leeway()) {
		return token.AccessToken, nil
	}
	if strings.TrimSpace(token.RefreshToken) == "" {
		return "", fmt.Errorf("auth token expired and has no refresh token; run md2x auth login")
	}
	if strings.TrimSpace(opts.ClientID) == "" {
		return "", fmt.Errorf("auth token expired and auth.client_id is not configured; run md2x config init --client-id YOUR_CLIENT_ID --force")
	}
	refreshed, err := opts.Client.Refresh(ctx, RefreshRequest{
		ClientID:     opts.ClientID,
		RefreshToken: token.RefreshToken,
	})
	if err != nil {
		return "", err
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = token.RefreshToken
	}
	if err := opts.Store.Save(refreshed); err != nil {
		return "", err
	}
	return refreshed.AccessToken, nil
}
