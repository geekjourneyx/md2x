# Configuration

md2x supports a local YAML config file for defaults that should survive shell sessions. The default path is:

```text
~/.config/md2x/config.yaml
```

You can override the path with either:

```bash
md2x --config /path/to/config.yaml config show
MD2X_CONFIG=/path/to/config.yaml md2x config show
```

## Initialize

Create a skeleton config:

```bash
md2x config init
```

Create one with common authentication defaults:

```bash
md2x config init --client-id YOUR_X_OAUTH2_CLIENT_ID
```

For a direct token workflow:

```bash
md2x config init --bearer-token YOUR_USER_CONTEXT_TOKEN
```

The config file is written with `0600` permissions. Do not commit it.

## Show Or List

```bash
md2x config show
md2x config list
md2x config path
```

`show` and `list` return the effective config. Sensitive values are redacted:

```yaml
auth:
  bearer_token: <redacted>
```

## File Format

```yaml
version: 1
api:
  base_url: https://api.x.com
auth:
  mode: oauth2_pkce
  client_id: ""
  redirect_uri: http://127.0.0.1:8765/callback
  scopes:
    - tweet.read
    - tweet.write
    - users.read
    - media.write
    - offline.access
  profile: default
  bearer_token: ""
  xurl_config: ""
  app: md2x
  username: ""
```

`client_id` is the OAuth2 Client ID from the X app's User authentication settings. `bearer_token` must be an X user-context access token and is mainly for smoke tests or controlled local automation.

## Priority

md2x resolves configuration in this order:

1. Command flags, such as `--api-base-url`, `--app`, `--username`, and `--xurl-config`.
2. Environment variables:
   - `X_BEARER_TOKEN`
   - `MD2X_API_BASE_URL`
   - `MD2X_CLIENT_ID`
   - `MD2X_REDIRECT_URI`
   - `MD2X_AUTH_PROFILE`
   - `MD2X_XURL_CONFIG`
   - `MD2X_APP`
   - `MD2X_USERNAME`
   - `MD2X_CONFIG`
3. YAML config file.
4. Built-in defaults.

`draft` first validates local input, then resolves auth. This lets agents get Markdown and media diagnostics before needing credentials.
