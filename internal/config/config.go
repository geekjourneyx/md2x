package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/geekjourneyx/md2x/internal/auth"
	"gopkg.in/yaml.v3"
)

const Version = 1
const DefaultAPIBaseURL = "https://api.x.com"
const DefaultAPITimeout = "120s"

type Config struct {
	Version int        `yaml:"version" json:"version"`
	API     APIConfig  `yaml:"api" json:"api"`
	Auth    AuthConfig `yaml:"auth" json:"auth"`
}

type APIConfig struct {
	BaseURL string `yaml:"base_url" json:"base_url"`
	Timeout string `yaml:"timeout" json:"timeout"`
}

type AuthConfig struct {
	BearerToken string   `yaml:"bearer_token,omitempty" json:"bearer_token,omitempty"`
	Mode        string   `yaml:"mode,omitempty" json:"mode,omitempty"`
	ClientID    string   `yaml:"client_id,omitempty" json:"client_id,omitempty"`
	RedirectURI string   `yaml:"redirect_uri,omitempty" json:"redirect_uri,omitempty"`
	Scopes      []string `yaml:"scopes,omitempty" json:"scopes,omitempty"`
	Profile     string   `yaml:"profile,omitempty" json:"profile,omitempty"`
	XurlConfig  string   `yaml:"xurl_config,omitempty" json:"xurl_config,omitempty"`
	App         string   `yaml:"app,omitempty" json:"app,omitempty"`
	Username    string   `yaml:"username,omitempty" json:"username,omitempty"`
}

type RedactedConfig struct {
	Version int                `yaml:"version" json:"version"`
	API     APIConfig          `yaml:"api" json:"api"`
	Auth    RedactedAuthConfig `yaml:"auth" json:"auth"`
}

type RedactedAuthConfig struct {
	BearerToken string   `yaml:"bearer_token,omitempty" json:"bearer_token,omitempty"`
	Mode        string   `yaml:"mode,omitempty" json:"mode,omitempty"`
	ClientID    string   `yaml:"client_id,omitempty" json:"client_id,omitempty"`
	RedirectURI string   `yaml:"redirect_uri,omitempty" json:"redirect_uri,omitempty"`
	Scopes      []string `yaml:"scopes,omitempty" json:"scopes,omitempty"`
	Profile     string   `yaml:"profile,omitempty" json:"profile,omitempty"`
	XurlConfig  string   `yaml:"xurl_config,omitempty" json:"xurl_config,omitempty"`
	App         string   `yaml:"app,omitempty" json:"app,omitempty"`
	Username    string   `yaml:"username,omitempty" json:"username,omitempty"`
}

func DefaultPath() (string, error) {
	configDir := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find home directory for config: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "md2x", "config.yaml"), nil
}

func Default() Config {
	return Config{
		Version: Version,
		API: APIConfig{
			BaseURL: DefaultAPIBaseURL,
			Timeout: DefaultAPITimeout,
		},
		Auth: AuthConfig{
			Mode:        "oauth2_pkce",
			RedirectURI: auth.DefaultRedirectURI,
			Scopes:      append([]string(nil), auth.DefaultScopes...),
			Profile:     auth.DefaultProfile,
		},
	}
}

func Load(path string) (Config, bool, error) {
	cfg := Default()
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return cfg, false, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, false, nil
		}
		return cfg, false, fmt.Errorf("read md2x config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, true, fmt.Errorf("parse md2x config %q: %w", path, err)
	}
	normalize(&cfg)
	return cfg, true, nil
}

func WriteInitial(path string, cfg Config, force bool) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	normalize(&cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config already exists at %s; pass --force to overwrite", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat config %q: %w", path, err)
		}
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

func ApplyEnv(cfg Config) Config {
	if value := strings.TrimSpace(os.Getenv("MD2X_API_BASE_URL")); value != "" {
		cfg.API.BaseURL = value
	}
	if value := strings.TrimSpace(os.Getenv("MD2X_HTTP_TIMEOUT")); value != "" {
		cfg.API.Timeout = value
	}
	if value := strings.TrimSpace(os.Getenv("X_BEARER_TOKEN")); value != "" {
		cfg.Auth.BearerToken = value
	}
	if value := strings.TrimSpace(os.Getenv("MD2X_XURL_CONFIG")); value != "" {
		cfg.Auth.XurlConfig = value
	}
	if value := strings.TrimSpace(os.Getenv("MD2X_APP")); value != "" {
		cfg.Auth.App = value
	}
	if value := strings.TrimSpace(os.Getenv("MD2X_USERNAME")); value != "" {
		cfg.Auth.Username = value
	}
	if value := strings.TrimSpace(os.Getenv("MD2X_CLIENT_ID")); value != "" {
		cfg.Auth.ClientID = value
	}
	if value := strings.TrimSpace(os.Getenv("MD2X_REDIRECT_URI")); value != "" {
		cfg.Auth.RedirectURI = value
	}
	if value := strings.TrimSpace(os.Getenv("MD2X_AUTH_PROFILE")); value != "" {
		cfg.Auth.Profile = value
	}
	return cfg
}

func Redact(cfg Config) RedactedConfig {
	redactedToken := ""
	if strings.TrimSpace(cfg.Auth.BearerToken) != "" {
		redactedToken = "<redacted>"
	}
	return RedactedConfig{
		Version: cfg.Version,
		API:     cfg.API,
		Auth: RedactedAuthConfig{
			BearerToken: redactedToken,
			Mode:        cfg.Auth.Mode,
			ClientID:    cfg.Auth.ClientID,
			RedirectURI: cfg.Auth.RedirectURI,
			Scopes:      append([]string(nil), cfg.Auth.Scopes...),
			Profile:     cfg.Auth.Profile,
			XurlConfig:  cfg.Auth.XurlConfig,
			App:         cfg.Auth.App,
			Username:    cfg.Auth.Username,
		},
	}
}

func normalize(cfg *Config) {
	if cfg.Version == 0 {
		cfg.Version = Version
	}
	if strings.TrimSpace(cfg.API.BaseURL) == "" {
		cfg.API.BaseURL = DefaultAPIBaseURL
	}
	if strings.TrimSpace(cfg.API.Timeout) == "" {
		cfg.API.Timeout = DefaultAPITimeout
	}
	if strings.TrimSpace(cfg.Auth.Mode) == "" {
		cfg.Auth.Mode = "oauth2_pkce"
	}
	if strings.TrimSpace(cfg.Auth.RedirectURI) == "" {
		cfg.Auth.RedirectURI = auth.DefaultRedirectURI
	}
	if len(cfg.Auth.Scopes) == 0 {
		cfg.Auth.Scopes = append([]string(nil), auth.DefaultScopes...)
	}
	if strings.TrimSpace(cfg.Auth.Profile) == "" {
		cfg.Auth.Profile = auth.DefaultProfile
	}
}

func APITimeout(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = DefaultAPITimeout
	}
	timeout, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse api.timeout %q: %w", value, err)
	}
	if timeout <= 0 {
		return 0, fmt.Errorf("api.timeout must be greater than 0, got %q", value)
	}
	return timeout, nil
}
