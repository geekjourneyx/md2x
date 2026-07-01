package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/geekjourneyx/md2x/internal/article"
	md2xauth "github.com/geekjourneyx/md2x/internal/auth"
	md2xconfig "github.com/geekjourneyx/md2x/internal/config"
	"github.com/geekjourneyx/md2x/internal/draftjs"
	"github.com/geekjourneyx/md2x/internal/markdown"
	"github.com/geekjourneyx/md2x/internal/xapi"
	"github.com/geekjourneyx/md2x/internal/xauth"
	"github.com/spf13/cobra"
)

type draftData struct {
	ArticleID string           `json:"article_id"`
	Title     string           `json:"title"`
	Media     []draftMediaData `json:"media"`
}

type draftMediaData struct {
	Role          string `json:"role"`
	Source        string `json:"source"`
	AssetIndex    *int   `json:"asset_index,omitempty"`
	MediaID       string `json:"media_id"`
	MediaCategory string `json:"media_category"`
}

func newDraftCommand(opts *rootOptions) *cobra.Command {
	var xurlConfig string
	var appName string
	var username string
	var apiBaseURL string
	var apiTimeout string
	var clientID string
	var authProfile string

	cmd := &cobra.Command{
		Use:   "draft <article.md>",
		Short: "Create an X Article draft",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourcePath := args[0]
			content, err := os.ReadFile(sourcePath)
			if err != nil {
				return &ExitError{Code: "INPUT_READ_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}

			doc, err := markdown.Parse(sourcePath, content)
			if err != nil {
				return &ExitError{Code: "MARKDOWN_PARSE_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}
			diagnostics := append(doc.Warnings, article.ValidateLocalInputs(doc)...)
			if article.BlockingDiagnostics(diagnostics) {
				return &ExitError{
					Code:        "INPUT_VALIDATION_FAILED",
					Message:     firstDiagnosticMessage(diagnostics),
					Exit:        2,
					Diagnostics: diagnostics,
				}
			}

			md2xConfigPath, err := resolveConfigPath(opts.configPath)
			if err != nil {
				return &ExitError{Code: "CONFIG_PATH_FAILED", Message: err.Error(), Exit: 3, Err: err}
			}
			cfg, _, err := md2xconfig.Load(md2xConfigPath)
			if err != nil {
				return &ExitError{Code: "CONFIG_READ_FAILED", Message: err.Error(), Exit: 3, Err: err}
			}
			cfg = md2xconfig.ApplyEnv(cfg)
			if cmd.Flags().Changed("api-base-url") {
				cfg.API.BaseURL = apiBaseURL
			}
			if cmd.Flags().Changed("api-timeout") {
				cfg.API.Timeout = apiTimeout
			}
			if cmd.Flags().Changed("xurl-config") {
				cfg.Auth.XurlConfig = xurlConfig
			}
			if cmd.Flags().Changed("app") {
				cfg.Auth.App = appName
			}
			if cmd.Flags().Changed("username") {
				cfg.Auth.Username = username
			}
			if cmd.Flags().Changed("client-id") {
				cfg.Auth.ClientID = clientID
			}
			if cmd.Flags().Changed("auth-profile") {
				cfg.Auth.Profile = authProfile
			}
			timeout, err := md2xconfig.APITimeout(cfg.API.Timeout)
			if err != nil {
				return &ExitError{Code: "API_TIMEOUT_INVALID", Message: err.Error(), Exit: 3, Err: err}
			}

			accessToken, err := resolveDraftAccessToken(cmd, cfg)
			if err != nil {
				return err
			}

			client := xapi.NewClientWithTimeout(cfg.API.BaseURL, accessToken, nil, timeout)
			coverMedia, uploadedMedia, err := uploadDraftMedia(client, doc)
			if err != nil {
				var mediaValidationErr *xapi.MediaValidationError
				if errors.As(err, &mediaValidationErr) {
					return &ExitError{Code: "MEDIA_VALIDATION_FAILED", Message: err.Error(), Exit: 2, Err: err}
				}
				return &ExitError{Code: "X_MEDIA_UPLOAD_FAILED", Message: err.Error(), Exit: 4, Err: err, Details: xAPIErrorDetails(err)}
			}

			contentState, err := draftjs.Render(doc)
			if err != nil {
				return &ExitError{Code: "DRAFTJS_RENDER_FAILED", Message: err.Error(), Exit: 2, Err: err}
			}

			draft, err := client.CreateDraft(xapi.CreateDraftRequest{
				Title:        doc.Title,
				ContentState: contentState,
				CoverMedia:   coverMedia,
			})
			if err != nil {
				return &ExitError{Code: "X_DRAFT_FAILED", Message: err.Error(), Exit: 4, Err: err, Details: xAPIErrorDetails(err)}
			}

			if opts.json {
				return writeJSON(cmd.OutOrStdout(), Envelope{
					Success:       true,
					SchemaVersion: schemaVersion,
					Status:        "completed",
					Code:          "OK",
					Message:       "created X Article draft",
					Data: draftData{
						ArticleID: draft.ID,
						Title:     draft.Title,
						Media:     uploadedMedia,
					},
				})
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "created X Article draft %s\n", draft.ID)
			return err
		},
	}

	cmd.Flags().StringVar(&xurlConfig, "xurl-config", "", "path to xurl config")
	cmd.Flags().StringVar(&appName, "app", "", "xurl app name")
	cmd.Flags().StringVar(&username, "username", "", "xurl username")
	cmd.Flags().StringVar(&clientID, "client-id", "", "X OAuth2 client ID for native token refresh")
	cmd.Flags().StringVar(&authProfile, "auth-profile", "", "local OAuth2 token profile")
	cmd.Flags().StringVar(&apiBaseURL, "api-base-url", md2xconfig.DefaultAPIBaseURL, "X API base URL")
	cmd.Flags().StringVar(&apiTimeout, "api-timeout", md2xconfig.DefaultAPITimeout, "X API HTTP timeout")
	return cmd
}

func xAPIErrorDetails(err error) map[string]interface{} {
	var apiErr *xapi.APIError
	if !errors.As(err, &apiErr) {
		return nil
	}
	apiDetails := map[string]interface{}{
		"operation":   apiErr.Operation,
		"status":      apiErr.Status,
		"status_code": apiErr.StatusCode,
		"retryable":   retryableStatus(apiErr.StatusCode),
	}
	if apiErr.RateLimit != nil {
		apiDetails["rate_limit"] = apiErr.RateLimit
	}
	return map[string]interface{}{"x_api": apiDetails}
}

func retryableStatus(statusCode int) bool {
	switch statusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

func resolveDraftAccessToken(cmd *cobra.Command, cfg md2xconfig.Config) (string, error) {
	nativeStore, err := md2xauth.NewDefaultStore(cfg.Auth.Profile)
	if err != nil {
		return "", &ExitError{Code: "AUTH_STORE_FAILED", Message: err.Error(), Exit: 3, Err: err}
	}
	accessToken, err := md2xauth.ResolveAccessToken(cmd.Context(), md2xauth.ResolveOptions{
		BearerToken: cfg.Auth.BearerToken,
		ClientID:    cfg.Auth.ClientID,
		Store:       nativeStore,
		Client:      md2xauth.Client{APIBaseURL: cfg.API.BaseURL},
	})
	if err == nil {
		return accessToken, nil
	}
	if !errors.Is(err, md2xauth.ErrNotAuthenticated) {
		return "", &ExitError{Code: "AUTH_TOKEN_NOT_FOUND", Message: err.Error(), Exit: 3, Err: err}
	}

	configPath, err := resolveXurlConfigPath(cfg.Auth.XurlConfig)
	if err != nil {
		return "", &ExitError{Code: "AUTH_CONFIG_FAILED", Message: err.Error(), Exit: 3, Err: err}
	}
	token, err := xauth.LoadXurlOAuth2Token(configPath, cfg.Auth.App, cfg.Auth.Username)
	if err != nil {
		return "", &ExitError{Code: "AUTH_TOKEN_NOT_FOUND", Message: err.Error(), Exit: 3, Err: err}
	}
	return strings.TrimSpace(token.AccessToken), nil
}

func firstDiagnosticMessage(diagnostics []article.Diagnostic) string {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			return diagnostic.Message
		}
	}
	if len(diagnostics) > 0 {
		return diagnostics[0].Message
	}
	return "input validation failed"
}

func resolveXurlConfigPath(configPath string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory for xurl config: %w", err)
	}
	return filepath.Join(home, ".xurl"), nil
}

func uploadDraftMedia(client *xapi.Client, doc *article.Document) (*xapi.MediaRef, []draftMediaData, error) {
	baseDir := filepath.Dir(doc.SourcePath)
	var coverMedia *xapi.MediaRef
	var media []draftMediaData
	uploadCache := newMediaUploadCache()

	if doc.Cover != "" {
		result, err := uploadCache.uploadImage(client, resolveArticlePath(baseDir, doc.Cover))
		if err != nil {
			return nil, nil, err
		}
		coverMedia = &xapi.MediaRef{
			MediaCategory: result.MediaCategory,
			MediaID:       result.MediaID,
		}
		media = append(media, draftMediaData{
			Role:          "cover",
			Source:        doc.Cover,
			MediaID:       result.MediaID,
			MediaCategory: result.MediaCategory,
		})
	}

	for _, asset := range doc.Assets {
		if asset.Role != "body" {
			continue
		}
		result, err := uploadCache.uploadImage(client, resolveArticlePath(baseDir, asset.Source))
		if err != nil {
			return nil, nil, err
		}
		if !attachUploadedMedia(doc, asset.Index, result) {
			return nil, nil, fmt.Errorf("image block for asset_index %d not found", asset.Index)
		}
		assetIndex := asset.Index
		media = append(media, draftMediaData{
			Role:          asset.Role,
			Source:        asset.Source,
			AssetIndex:    &assetIndex,
			MediaID:       result.MediaID,
			MediaCategory: result.MediaCategory,
		})
	}

	return coverMedia, media, nil
}

func resolveArticlePath(baseDir, assetPath string) string {
	if filepath.IsAbs(assetPath) {
		return assetPath
	}
	return filepath.Join(baseDir, assetPath)
}

func attachUploadedMedia(doc *article.Document, assetIndex int, media *xapi.UploadMediaResult) bool {
	for i := range doc.Blocks {
		block := &doc.Blocks[i]
		if block.Type != "image" {
			continue
		}
		if block.Data == nil {
			continue
		}
		if block.Data["asset_index"] != strconv.Itoa(assetIndex) {
			continue
		}
		block.Data["media_id"] = media.MediaID
		block.Data["media_category"] = media.MediaCategory
		return true
	}
	return false
}
