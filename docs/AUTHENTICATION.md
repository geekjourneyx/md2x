# Authentication

md2x creates X Article drafts through the official X API. The relevant Articles endpoints are X's create draft article endpoint and publish article endpoint, but V1 only calls draft creation by default.

- https://docs.x.com/x-api/articles/create-draft-article#create-draft-article
- https://docs.x.com/x-api/articles/publish-article

Use an official OAuth2 user-context token. The recommended md2x path is native OAuth2 PKCE:

```bash
md2x config init --client-id YOUR_X_OAUTH2_CLIENT_ID
md2x auth login
md2x auth status
```

Full setup is in [OAuth2 PKCE Tutorial](OAUTH2-PKCE.md).

## Required X App Settings

In the X Developer Portal, open your app and configure User authentication:

- App permissions: Read and write.
- Type of App: Native App or another public-client setting that supports PKCE.
- Callback URL: `http://127.0.0.1:8765/callback`.
- Website URL: any valid URL you own or control if the console requires one.

Use the OAuth2 Client ID from this screen. The App-Only Bearer Token is not enough for X Articles.

## Compatibility: xurl

```bash
xurl auth apps add md2x --client-id YOUR_CLIENT_ID --client-secret YOUR_CLIENT_SECRET --redirect-uri http://localhost:8080/callback
xurl auth oauth2 --app md2x
```

The token should belong to the X user that will own the draft. xurl remains supported, but it is not required for the normal md2x flow.

## Required Scopes

- `tweet.read`
- `tweet.write`
- `users.read`
- `media.write`
- `offline.access`

`offline.access` lets automation refresh credentials without an interactive browser step.

## Token Lookup

`md2x draft` resolves credentials in this order:

1. `X_BEARER_TOKEN`.
2. Native md2x OAuth2 token store from `md2x auth login`.
3. `auth.bearer_token` in `~/.config/md2x/config.yaml`.
4. `xurl` OAuth2 token store.

The native token store lives under:

```text
${XDG_STATE_HOME:-~/.local/state}/md2x/auth/default.json
```

For CI or custom token management, `X_BEARER_TOKEN` can override token lookup:

```bash
export X_BEARER_TOKEN="..."
```

For local defaults, initialize `~/.config/md2x/config.yaml`:

```bash
md2x config init --client-id YOUR_X_OAUTH2_CLIENT_ID
md2x config show
```

Configuration priority is flags, then environment variables, then the YAML config file, then built-in defaults. See [Configuration](CONFIG.md).

Do not commit tokens. Agents should pass credentials through the execution environment and redact them from logs.

## Offline Commands

These commands do not require authentication:

```bash
md2x inspect article.md --json
md2x render article.md --format draftjs --json
```

Only `draft` needs live X credentials.
