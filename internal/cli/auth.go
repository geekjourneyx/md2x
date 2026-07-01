package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	md2xauth "github.com/geekjourneyx/md2x/internal/auth"
	md2xconfig "github.com/geekjourneyx/md2x/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type authStatusData struct {
	Profile string                `json:"profile"`
	Path    string                `json:"path"`
	Status  md2xauth.TokenStatus  `json:"status"`
	Config  md2xconfig.AuthConfig `json:"config,omitempty"`
}

func newAuthCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage X OAuth2 authentication",
	}
	cmd.AddCommand(
		newAuthLoginCommand(opts),
		newAuthStatusCommand(opts),
		newAuthRefreshCommand(opts),
		newAuthLogoutCommand(opts),
	)
	return cmd
}

func newAuthStatusCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show local OAuth2 authentication status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadEffectiveConfig(opts)
			if err != nil {
				return err
			}
			store, err := md2xauth.NewDefaultStore(cfg.Auth.Profile)
			if err != nil {
				return &ExitError{Code: "AUTH_STORE_FAILED", Message: err.Error(), Exit: 3, Err: err}
			}
			token, err := store.Load()
			status := md2xauth.TokenStatus{Authenticated: false}
			if err == nil {
				status = md2xauth.RedactTokenStatus(token)
			} else if !errors.Is(err, md2xauth.ErrNotAuthenticated) {
				return &ExitError{Code: "AUTH_STATUS_FAILED", Message: err.Error(), Exit: 3, Err: err}
			}
			return writeAuthStatus(cmd, opts, "showed auth status", authStatusData{
				Profile: cfg.Auth.Profile,
				Path:    store.Path,
				Status:  status,
				Config:  redactedAuthConfigAsPlain(md2xconfig.Redact(cfg).Auth),
			})
		},
	}
}

func newAuthRefreshCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the local OAuth2 access token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadEffectiveConfig(opts)
			if err != nil {
				return err
			}
			if strings.TrimSpace(cfg.Auth.ClientID) == "" {
				return &ExitError{Code: "AUTH_CONFIG_FAILED", Message: "auth.client_id is required; run md2x config init --client-id YOUR_CLIENT_ID --force", Exit: 3}
			}
			store, err := md2xauth.NewDefaultStore(cfg.Auth.Profile)
			if err != nil {
				return &ExitError{Code: "AUTH_STORE_FAILED", Message: err.Error(), Exit: 3, Err: err}
			}
			token, err := store.Load()
			if err != nil {
				return &ExitError{Code: "AUTH_TOKEN_NOT_FOUND", Message: err.Error(), Exit: 3, Err: err}
			}
			if strings.TrimSpace(token.RefreshToken) == "" {
				return &ExitError{Code: "AUTH_REFRESH_FAILED", Message: "stored token has no refresh_token; run md2x auth login", Exit: 3}
			}
			client := md2xauth.Client{APIBaseURL: cfg.API.BaseURL}
			refreshed, err := client.Refresh(cmd.Context(), md2xauth.RefreshRequest{
				ClientID:     cfg.Auth.ClientID,
				RefreshToken: token.RefreshToken,
			})
			if err != nil {
				return &ExitError{Code: "AUTH_REFRESH_FAILED", Message: err.Error(), Exit: 4, Err: err}
			}
			if refreshed.RefreshToken == "" {
				refreshed.RefreshToken = token.RefreshToken
			}
			if err := store.Save(refreshed); err != nil {
				return &ExitError{Code: "AUTH_STORE_FAILED", Message: err.Error(), Exit: 3, Err: err}
			}
			return writeAuthStatus(cmd, opts, "refreshed auth token", authStatusData{
				Profile: cfg.Auth.Profile,
				Path:    store.Path,
				Status:  md2xauth.RedactTokenStatus(refreshed),
			})
		},
	}
}

func newAuthLogoutCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Delete the local OAuth2 token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadEffectiveConfig(opts)
			if err != nil {
				return err
			}
			store, err := md2xauth.NewDefaultStore(cfg.Auth.Profile)
			if err != nil {
				return &ExitError{Code: "AUTH_STORE_FAILED", Message: err.Error(), Exit: 3, Err: err}
			}
			if err := store.Delete(); err != nil {
				return &ExitError{Code: "AUTH_LOGOUT_FAILED", Message: err.Error(), Exit: 3, Err: err}
			}
			data := authStatusData{
				Profile: cfg.Auth.Profile,
				Path:    store.Path,
				Status:  md2xauth.TokenStatus{Authenticated: false},
			}
			return writeAuthStatus(cmd, opts, "logged out", data)
		},
	}
}

func newAuthLoginCommand(opts *rootOptions) *cobra.Command {
	var clientID string
	var redirectURI string
	var profile string
	var apiBaseURL string
	var authorizeURL string
	var noOpen bool
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authorize md2x with X OAuth2 PKCE",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadEffectiveConfig(opts)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("client-id") {
				cfg.Auth.ClientID = clientID
			}
			if cmd.Flags().Changed("redirect-uri") {
				cfg.Auth.RedirectURI = redirectURI
			}
			if cmd.Flags().Changed("auth-profile") {
				cfg.Auth.Profile = profile
			}
			if cmd.Flags().Changed("api-base-url") {
				cfg.API.BaseURL = apiBaseURL
			}
			if strings.TrimSpace(cfg.Auth.ClientID) == "" {
				return &ExitError{Code: "AUTH_CONFIG_FAILED", Message: "missing X OAuth2 client_id; run md2x config init --client-id YOUR_CLIENT_ID --force", Exit: 3}
			}
			store, err := md2xauth.NewDefaultStore(cfg.Auth.Profile)
			if err != nil {
				return &ExitError{Code: "AUTH_STORE_FAILED", Message: err.Error(), Exit: 3, Err: err}
			}
			progressOut := cmd.OutOrStdout()
			if opts.json {
				progressOut = io.Discard
			}
			token, err := runOAuthLogin(cmd.Context(), cfg, store, authorizeURL, noOpen, timeout, progressOut)
			if err != nil {
				return err
			}
			return writeAuthStatus(cmd, opts, "logged in", authStatusData{
				Profile: cfg.Auth.Profile,
				Path:    store.Path,
				Status:  md2xauth.RedactTokenStatus(token),
			})
		},
	}
	cmd.Flags().StringVar(&clientID, "client-id", "", "X OAuth2 client ID")
	cmd.Flags().StringVar(&redirectURI, "redirect-uri", md2xauth.DefaultRedirectURI, "OAuth2 callback URL")
	cmd.Flags().StringVar(&profile, "auth-profile", md2xauth.DefaultProfile, "local OAuth2 token profile")
	cmd.Flags().StringVar(&apiBaseURL, "api-base-url", md2xconfig.DefaultAPIBaseURL, "X API base URL")
	cmd.Flags().StringVar(&authorizeURL, "authorize-url", md2xauth.DefaultAuthorizeURL, "X OAuth2 authorize URL")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "print authorize URL instead of opening a browser")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "OAuth2 login timeout")
	return cmd
}

func runOAuthLogin(ctx context.Context, cfg md2xconfig.Config, store md2xauth.FileStore, authorizeBaseURL string, noOpen bool, timeout time.Duration, out interface{ Write([]byte) (int, error) }) (md2xauth.Token, error) {
	verifier, err := md2xauth.GenerateCodeVerifier()
	if err != nil {
		return md2xauth.Token{}, &ExitError{Code: "AUTH_PKCE_FAILED", Message: err.Error(), Exit: 3, Err: err}
	}
	state, err := md2xauth.GenerateState()
	if err != nil {
		return md2xauth.Token{}, &ExitError{Code: "AUTH_PKCE_FAILED", Message: err.Error(), Exit: 3, Err: err}
	}
	authURL, err := md2xauth.AuthorizeURL(authorizeBaseURL, cfg.Auth.ClientID, cfg.Auth.RedirectURI, cfg.Auth.Scopes, state, md2xauth.CodeChallengeS256(verifier))
	if err != nil {
		return md2xauth.Token{}, &ExitError{Code: "AUTH_CONFIG_FAILED", Message: err.Error(), Exit: 3, Err: err}
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	server, err := startCallbackServer(cfg.Auth.RedirectURI, state, codeCh, errCh)
	if err != nil {
		return md2xauth.Token{}, &ExitError{Code: "AUTH_CALLBACK_FAILED", Message: err.Error(), Exit: 3, Err: err}
	}
	defer func() { _ = server.Shutdown(context.Background()) }()

	_, _ = fmt.Fprintf(out, "Open this URL to authorize md2x:\n%s\n", authURL)
	if !noOpen {
		_ = openBrowser(authURL)
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return md2xauth.Token{}, &ExitError{Code: "AUTH_CALLBACK_FAILED", Message: err.Error(), Exit: 3, Err: err}
	case <-waitCtx.Done():
		return md2xauth.Token{}, &ExitError{Code: "AUTH_TIMEOUT", Message: "timed out waiting for OAuth2 callback", Exit: 3, Err: waitCtx.Err()}
	}
	_, _ = fmt.Fprintln(out, "OAuth callback received; exchanging code for token...")
	client := md2xauth.Client{APIBaseURL: cfg.API.BaseURL}
	token, err := client.ExchangeCode(waitCtx, md2xauth.ExchangeCodeRequest{
		ClientID:     cfg.Auth.ClientID,
		Code:         code,
		CodeVerifier: verifier,
		RedirectURI:  cfg.Auth.RedirectURI,
	})
	if err != nil {
		return md2xauth.Token{}, &ExitError{Code: "AUTH_TOKEN_EXCHANGE_FAILED", Message: err.Error(), Exit: 4, Err: err}
	}
	if err := store.Save(token); err != nil {
		return md2xauth.Token{}, &ExitError{Code: "AUTH_STORE_FAILED", Message: err.Error(), Exit: 3, Err: err}
	}
	_, _ = fmt.Fprintf(out, "OAuth token saved to %s\n", store.Path)
	return token, nil
}

func startCallbackServer(redirectURI, expectedState string, codeCh chan<- string, errCh chan<- error) (*http.Server, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return nil, fmt.Errorf("parse redirect_uri: %w", err)
	}
	if u.Scheme != "http" || strings.TrimSpace(u.Host) == "" {
		return nil, fmt.Errorf("redirect_uri must be an http localhost URL")
	}
	listener, err := net.Listen("tcp", u.Host)
	if err != nil {
		return nil, fmt.Errorf("listen on OAuth callback %s: %w", u.Host, err)
	}
	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}
	mux.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if oauthErr := query.Get("error"); oauthErr != "" {
			errCh <- fmt.Errorf("x authorization failed: %s", oauthErr)
			http.Error(w, "authorization failed", http.StatusBadRequest)
			return
		}
		if query.Get("state") != expectedState {
			errCh <- fmt.Errorf("OAuth state mismatch")
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		code := query.Get("code")
		if code == "" {
			errCh <- fmt.Errorf("OAuth callback missing code")
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		codeCh <- code
		_, _ = fmt.Fprintln(w, "md2x received the authorization callback. Return to the terminal to finish login.")
	})
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	return server, nil
}

func openBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}

func loadEffectiveConfig(opts *rootOptions) (md2xconfig.Config, error) {
	path, err := resolveConfigPath(opts.configPath)
	if err != nil {
		return md2xconfig.Config{}, &ExitError{Code: "CONFIG_PATH_FAILED", Message: err.Error(), Exit: 3, Err: err}
	}
	cfg, _, err := md2xconfig.Load(path)
	if err != nil {
		return md2xconfig.Config{}, &ExitError{Code: "CONFIG_READ_FAILED", Message: err.Error(), Exit: 3, Err: err}
	}
	return md2xconfig.ApplyEnv(cfg), nil
}

func writeAuthStatus(cmd *cobra.Command, opts *rootOptions, message string, data authStatusData) error {
	if opts.json {
		return writeJSON(cmd.OutOrStdout(), Envelope{
			Success:       true,
			SchemaVersion: schemaVersion,
			Status:        "completed",
			Code:          "OK",
			Message:       message,
			Data:          data,
		})
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), string(out))
	return err
}

func redactedAuthConfigAsPlain(in md2xconfig.RedactedAuthConfig) md2xconfig.AuthConfig {
	return md2xconfig.AuthConfig{
		BearerToken: in.BearerToken,
		Mode:        in.Mode,
		ClientID:    in.ClientID,
		RedirectURI: in.RedirectURI,
		Scopes:      append([]string(nil), in.Scopes...),
		Profile:     in.Profile,
		XurlConfig:  in.XurlConfig,
		App:         in.App,
		Username:    in.Username,
	}
}
