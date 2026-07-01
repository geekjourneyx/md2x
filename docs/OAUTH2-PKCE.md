# OAuth2 PKCE Tutorial

This guide walks a first-time user from the X Developer Portal to a working `md2x draft` command.

Official X documentation:

- OAuth2 user access token: https://docs.x.com/fundamentals/authentication/oauth-2-0/user-access-token
- Create draft article: https://docs.x.com/x-api/articles/create-draft-article#create-draft-article
- Publish article: https://docs.x.com/x-api/articles/publish-article

md2x uses OAuth2 Authorization Code Flow with PKCE because X Articles require a user-context token. The App-Only Bearer Token shown under App-Only Authentication is not enough.

## 1. Configure The X App

Open X Developer Portal, choose your project and app, then open User authentication settings.

Use these values for the first local setup:

```text
App permissions: Read and write
Type of App: Native App or public client with PKCE
Callback URL: http://127.0.0.1:8765/callback
Website URL: any valid URL you control if required
```

Save the settings and copy the OAuth2 Client ID. Do not use the App-Only Bearer Token for md2x Articles.

## 2. Initialize md2x Config

```bash
md2x config init --client-id YOUR_X_OAUTH2_CLIENT_ID
```

If a config already exists:

```bash
md2x config init --client-id YOUR_X_OAUTH2_CLIENT_ID --force
```

Check the effective config:

```bash
md2x config show
```

Sensitive values are redacted. The config file is:

```text
~/.config/md2x/config.yaml
```

## 3. Login

```bash
md2x auth login
```

md2x will:

1. Generate a PKCE verifier and challenge.
2. Start a local callback server at `http://127.0.0.1:8765/callback`.
3. Open the X authorization URL in your browser.
4. Exchange the callback `code` for an access token and refresh token.
5. Save the token locally with `0600` permissions.

The browser callback page only means md2x received the authorization callback. Keep the terminal open until it prints that the OAuth token was saved. If the terminal is still running after the callback, md2x is usually exchanging the code with X's token endpoint.

For servers or agent sessions without a browser:

```bash
md2x auth login --no-open
```

Open the printed URL manually, approve access, and let the browser redirect to the callback URL on the same machine.

## 4. Verify Authentication

```bash
md2x auth status
md2x auth status --json
```

Expected JSON shape:

```json
{
  "success": true,
  "code": "OK",
  "data": {
    "profile": "default",
    "status": {
      "authenticated": true,
      "access_token": "<redacted>",
      "refresh_token": "<redacted>"
    }
  }
}
```

Tokens are stored at:

```text
${XDG_STATE_HOME:-~/.local/state}/md2x/auth/default.json
```

Do not commit this file.

## 5. Create A Draft

Run offline checks first:

```bash
md2x inspect article.md --json
md2x render article.md --format draftjs --json
```

Then create the X Article draft:

```bash
md2x draft article.md --json
```

md2x uploads local images, converts Markdown into DraftJS `content_state`, and calls X's create draft article endpoint. It does not publish by default.

## 6. Refresh Or Logout

Refresh manually:

```bash
md2x auth refresh
```

`md2x draft` also refreshes an expired native token automatically when a refresh token is available.

Remove local credentials:

```bash
md2x auth logout
```

## Agent Notes

Use `md2x auth status --json` before live operations. Treat `data.status.authenticated == true` as the readiness signal.

For CI, prefer an injected short-lived secret:

```bash
X_BEARER_TOKEN=... md2x draft article.md --json
```

`X_BEARER_TOKEN` has the highest priority and is never written to disk by md2x.
