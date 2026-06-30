package cli

import (
	"fmt"
	"os"
	"strings"

	md2xconfig "github.com/geekjourneyx/md2x/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type configData struct {
	Path   string                    `json:"path"`
	Found  bool                      `json:"found"`
	Config md2xconfig.RedactedConfig `json:"config"`
}

func newConfigCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage md2x configuration",
	}
	cmd.AddCommand(
		newConfigPathCommand(opts),
		newConfigShowCommand(opts),
		newConfigInitCommand(opts),
	)
	return cmd
}

func newConfigPathCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the md2x config path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveConfigPath(opts.configPath)
			if err != nil {
				return &ExitError{Code: "CONFIG_PATH_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}
			if opts.json {
				return writeJSON(cmd.OutOrStdout(), Envelope{
					Success:       true,
					SchemaVersion: schemaVersion,
					Status:        "completed",
					Code:          "OK",
					Message:       "resolved md2x config path",
					Data: map[string]string{
						"path": path,
					},
				})
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), path)
			return err
		},
	}
}

func newConfigShowCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show",
		Aliases: []string{"list"},
		Short:   "Show effective md2x configuration",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveConfigPath(opts.configPath)
			if err != nil {
				return &ExitError{Code: "CONFIG_PATH_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}
			cfg, found, err := md2xconfig.Load(path)
			if err != nil {
				return &ExitError{Code: "CONFIG_READ_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}
			cfg = md2xconfig.ApplyEnv(cfg)
			data := configData{Path: path, Found: found, Config: md2xconfig.Redact(cfg)}
			if opts.json {
				return writeJSON(cmd.OutOrStdout(), Envelope{
					Success:       true,
					SchemaVersion: schemaVersion,
					Status:        "completed",
					Code:          "OK",
					Message:       "showed md2x config",
					Data:          data,
				})
			}
			out, err := yaml.Marshal(data)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), string(out))
			return err
		},
	}
	return cmd
}

func newConfigInitCommand(opts *rootOptions) *cobra.Command {
	var force bool
	var bearerToken string
	var xurlConfig string
	var appName string
	var username string
	var apiBaseURL string
	var clientID string
	var redirectURI string
	var profile string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create an md2x config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveConfigPath(opts.configPath)
			if err != nil {
				return &ExitError{Code: "CONFIG_PATH_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}
			cfg := md2xconfig.Default()
			cfg.API.BaseURL = apiBaseURL
			if strings.TrimSpace(bearerToken) != "" {
				cfg.Auth.BearerToken = bearerToken
			}
			if strings.TrimSpace(xurlConfig) != "" {
				cfg.Auth.XurlConfig = xurlConfig
			}
			if strings.TrimSpace(appName) != "" {
				cfg.Auth.App = appName
			}
			if strings.TrimSpace(username) != "" {
				cfg.Auth.Username = username
			}
			if strings.TrimSpace(clientID) != "" {
				cfg.Auth.ClientID = clientID
			}
			if strings.TrimSpace(redirectURI) != "" {
				cfg.Auth.RedirectURI = redirectURI
			}
			if strings.TrimSpace(profile) != "" {
				cfg.Auth.Profile = profile
			}
			if err := md2xconfig.WriteInitial(path, cfg, force); err != nil {
				return &ExitError{Code: "CONFIG_INIT_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}
			data := configData{Path: path, Found: true, Config: md2xconfig.Redact(cfg)}
			if opts.json {
				return writeJSON(cmd.OutOrStdout(), Envelope{
					Success:       true,
					SchemaVersion: schemaVersion,
					Status:        "completed",
					Code:          "OK",
					Message:       "initialized md2x config",
					Data:          data,
				})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "created md2x config %s\n", path)
			return err
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing config")
	cmd.Flags().StringVar(&bearerToken, "bearer-token", "", "X user-context bearer token")
	cmd.Flags().StringVar(&xurlConfig, "xurl-config", "", "path to xurl config")
	cmd.Flags().StringVar(&appName, "app", "", "xurl app name")
	cmd.Flags().StringVar(&username, "username", "", "xurl username")
	cmd.Flags().StringVar(&clientID, "client-id", "", "X OAuth2 client ID")
	cmd.Flags().StringVar(&redirectURI, "redirect-uri", "", "OAuth2 callback URL")
	cmd.Flags().StringVar(&profile, "auth-profile", "", "local OAuth2 token profile")
	cmd.Flags().StringVar(&apiBaseURL, "api-base-url", md2xconfig.DefaultAPIBaseURL, "X API base URL")
	return cmd
}

func resolveConfigPath(path string) (string, error) {
	if path != "" {
		return path, nil
	}
	if envPath := strings.TrimSpace(os.Getenv("MD2X_CONFIG")); envPath != "" {
		return envPath, nil
	}
	return md2xconfig.DefaultPath()
}
