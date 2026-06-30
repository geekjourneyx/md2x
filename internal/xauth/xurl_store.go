package xauth

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const oauth2ExpirationSkew = 60 * time.Second

type OAuth2Token struct {
	AccessToken    string `yaml:"access_token"`
	RefreshToken   string `yaml:"refresh_token"`
	ExpirationTime uint64 `yaml:"expiration_time"`
}

type xurlConfig struct {
	Apps       map[string]xurlApp `yaml:"apps"`
	DefaultApp string             `yaml:"default_app"`
}

type xurlApp struct {
	ClientID     string                    `yaml:"client_id"`
	ClientSecret string                    `yaml:"client_secret"`
	DefaultUser  string                    `yaml:"default_user"`
	OAuth2Tokens map[string]xurlTokenEntry `yaml:"oauth2_tokens"`
}

type xurlTokenEntry struct {
	Type   string      `yaml:"type"`
	OAuth2 OAuth2Token `yaml:"oauth2"`
}

func LoadXurlOAuth2Token(path, appName, username string) (*OAuth2Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read xurl config %q: %w", path, err)
	}

	var cfg xurlConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse xurl config %q: %w", path, err)
	}

	if appName == "" {
		appName = cfg.DefaultApp
	}
	if appName == "" {
		return nil, fmt.Errorf("xurl config %q has no default_app and no app was requested", path)
	}

	app, ok := cfg.Apps[appName]
	if !ok {
		return nil, fmt.Errorf("xurl app %q not found in %q", appName, path)
	}

	if username == "" {
		username = app.DefaultUser
	}
	if username == "" && len(app.OAuth2Tokens) == 1 {
		for user := range app.OAuth2Tokens {
			username = user
		}
	}
	if username == "" {
		return nil, fmt.Errorf("xurl app %q has no default_user and no user was requested", appName)
	}

	entry, ok := app.OAuth2Tokens[username]
	if !ok {
		return nil, fmt.Errorf("xurl oauth2 token for app %q user %q not found", appName, username)
	}
	if entry.OAuth2.AccessToken == "" {
		return nil, fmt.Errorf("xurl oauth2 token for app %q user %q has empty access token", appName, username)
	}
	if tokenExpiredOrNearExpiry(entry.OAuth2.ExpirationTime) {
		return nil, fmt.Errorf("xurl oauth2 token for app %q user %q is expired or expires too soon; refresh with: xurl auth oauth2 --app %s", appName, username, appName)
	}

	token := entry.OAuth2
	return &token, nil
}

func tokenExpiredOrNearExpiry(expirationTime uint64) bool {
	if expirationTime == 0 {
		return true
	}
	return expirationTime <= uint64(time.Now().Add(oauth2ExpirationSkew).Unix())
}
