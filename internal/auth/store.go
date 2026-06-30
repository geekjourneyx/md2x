package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultProfile     = "default"
	tokenExpiryLeeway  = 60 * time.Second
	redactedSecretText = "<redacted>"
)

type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type TokenStatus struct {
	Authenticated bool   `json:"authenticated"`
	AccessToken   string `json:"access_token,omitempty"`
	RefreshToken  string `json:"refresh_token,omitempty"`
	TokenType     string `json:"token_type,omitempty"`
	Scope         string `json:"scope,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	Expired       bool   `json:"expired"`
}

type FileStore struct {
	Path string
}

func DefaultStorePath(profile string) (string, error) {
	profile = normalizeProfile(profile)
	stateHome := strings.TrimSpace(os.Getenv("XDG_STATE_HOME"))
	if stateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find home directory for auth store: %w", err)
		}
		stateHome = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateHome, "md2x", "auth", profile+".json"), nil
}

func NewDefaultStore(profile string) (FileStore, error) {
	path, err := DefaultStorePath(profile)
	if err != nil {
		return FileStore{}, err
	}
	return FileStore{Path: path}, nil
}

func (s FileStore) Load() (Token, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Token{}, ErrNotAuthenticated
		}
		return Token{}, fmt.Errorf("read auth token %q: %w", s.Path, err)
	}
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return Token{}, fmt.Errorf("parse auth token %q: %w", s.Path, err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return Token{}, fmt.Errorf("auth token %q has empty access_token", s.Path)
	}
	return token, nil
}

func (s FileStore) Save(token Token) error {
	if strings.TrimSpace(s.Path) == "" {
		return fmt.Errorf("auth token store path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return fmt.Errorf("create auth token directory: %w", err)
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("encode auth token: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(s.Path, data, 0o600); err != nil {
		return fmt.Errorf("write auth token %q: %w", s.Path, err)
	}
	return nil
}

func (s FileStore) Delete() error {
	if err := os.Remove(s.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete auth token %q: %w", s.Path, err)
	}
	return nil
}

func RedactTokenStatus(token Token) TokenStatus {
	status := TokenStatus{
		Authenticated: strings.TrimSpace(token.AccessToken) != "",
		TokenType:     token.TokenType,
		Scope:         token.Scope,
		Expired:       token.Expired(time.Now(), tokenExpiryLeeway),
	}
	if strings.TrimSpace(token.AccessToken) != "" {
		status.AccessToken = redactedSecretText
	}
	if strings.TrimSpace(token.RefreshToken) != "" {
		status.RefreshToken = redactedSecretText
	}
	if !token.ExpiresAt.IsZero() {
		status.ExpiresAt = token.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return status
}

func (t Token) Expired(now time.Time, leeway time.Duration) bool {
	if t.ExpiresAt.IsZero() {
		return true
	}
	return !t.ExpiresAt.After(now.Add(leeway))
}

func normalizeProfile(profile string) string {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return DefaultProfile
	}
	return profile
}
