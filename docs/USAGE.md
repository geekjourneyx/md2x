# Usage

md2x has these primary commands:

- `inspect`: parse and validate Markdown.
- `render`: compile Markdown into deterministic output.
- `draft`: create an X Article draft.
- `auth`: login, refresh, inspect, or remove OAuth2 credentials.
- `config`: initialize and inspect local configuration.

Global flags:

```bash
md2x --config ~/.config/md2x/config.yaml config show
```

The default config path is `~/.config/md2x/config.yaml`.

## inspect

```bash
md2x inspect article.md --json
```

Use this command first. It reports the parsed article model, frontmatter, media references, warnings, and validation failures. Local media checks include existence, supported extension, upload size limit, and image header validation.

For cost-aware workflows, `inspect --json` also reports the number of unique media uploads and the estimated X request count for `draft`:

```json
{
  "unique_media_count": 1,
  "estimated_x_requests": {
    "media_upload": 1,
    "create_draft": 1,
    "total": 2
  }
}
```

## render

```bash
md2x render article.md --format draftjs --json
```

`render` consumes the same parsed article model as `inspect` and returns DraftJS `content_state`. It is offline and deterministic. For articles with local images, it is a pre-upload preview because media IDs are only available during `draft`.

See [Markdown Syntax](MARKDOWN.md) for the supported Markdown subset and [X API Contract](X-API.md) for the `content_state` mapping.

## draft

```bash
md2x draft article.md --json
```

`draft` uploads media, renders DraftJS content, and creates an X Article draft. V1 does not publish by default.

The JSON POST body is created from the same render path as `md2x render --format draftjs`, after media upload has attached final media IDs. Cover media is included only when frontmatter provides `cover`.

V1 uploads images with the single-step X media endpoint. Duplicate image contents within the same command are uploaded once and reused. md2x does not persist media IDs across commands because uploaded media can expire.

Auth and API defaults can come from flags, environment variables, or the local config file. See [Configuration](CONFIG.md).

## auth

```bash
md2x auth login
md2x auth status --json
md2x auth refresh
md2x auth logout
```

`auth login` performs OAuth2 PKCE against X and stores the resulting user-context token locally. Use `auth status --json` in agent workflows before `draft`.

## Exit Codes

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | Usage or argument error |
| 2 | Input file, Markdown, frontmatter, or media validation error |
| 3 | Authentication or token error |
| 4 | X API draft or media upload failure |

Agents should branch on exit code and read the JSON envelope before retrying.
