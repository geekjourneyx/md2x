# Troubleshooting

## AUTH_TOKEN_NOT_FOUND

md2x could not find an X user-context bearer token.

Remediation:

```bash
md2x config init --client-id YOUR_X_OAUTH2_CLIENT_ID
md2x auth login
md2x draft article.md --json
```

Legacy xurl stores are still supported with `xurl auth oauth2 --app md2x`.

Confirm the token has these scopes:

- `tweet.read`
- `tweet.write`
- `users.read`
- `media.write`
- `offline.access`

## Browser Callback Succeeded But CLI Is Still Running

The callback page means md2x received the OAuth `code`. The CLI still needs to exchange that code with X's token endpoint and write the local token store.

Keep the terminal open until it prints that the OAuth token was saved. If it does not finish, check:

```bash
md2x config show
md2x auth login --timeout 30s
```

Make sure `api.base_url` is `https://api.x.com` unless you are intentionally testing against a mock endpoint.

## X_DRAFT_FAILED

X rejected the draft creation request or media upload.

Remediation:

1. Run `md2x inspect article.md --json` and fix validation warnings.
2. Run `md2x render article.md --format draftjs --json` and confirm DraftJS output is produced.
3. Refresh the OAuth2 token with `md2x auth refresh`.
4. Retry `md2x draft article.md --json`.

If the failure is rate-limit related, wait before retrying. If it is a media error, verify file paths, file sizes, and supported image formats.

For `429 Too Many Requests`, inspect the JSON error details:

```json
{
  "x_api": {
    "status_code": 429,
    "retryable": true,
    "rate_limit": {
      "limit": 1,
      "remaining": 0,
      "reset_at": "2030-01-01T00:00:00Z",
      "retry_after_seconds": 600
    }
  }
}
```

If `remaining` is `0`, wait until `reset_at` before retrying. If the first request for a new app returns 429, verify the X developer console package/environment, regenerate OAuth2 credentials for the Production app, and run `md2x auth logout && md2x auth login` so the stored token matches the active app.

## Draft Was Not Published

This is expected in V1. md2x creates X Article drafts and does not publish by default.
