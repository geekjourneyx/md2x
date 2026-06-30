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

## X_DRAFT_FAILED

X rejected the draft creation request or media upload.

Remediation:

1. Run `md2x inspect article.md --json` and fix validation warnings.
2. Run `md2x render article.md --format draftjs --json` and confirm DraftJS output is produced.
3. Refresh the OAuth2 token with `md2x auth refresh`.
4. Retry `md2x draft article.md --json`.

If the failure is rate-limit related, wait before retrying. If it is a media error, verify file paths, file sizes, and supported image formats.

## Draft Was Not Published

This is expected in V1. md2x creates X Article drafts and does not publish by default.
