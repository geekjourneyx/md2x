<div align="center">

# md2x

**Agent Native Go CLI for turning Markdown into X Articles drafts.**

<img src="assets/banner.webp" alt="md2x converts Markdown into X Articles drafts" width="100%" />

[Chinese](README_ZH.md) · [Quickstart](docs/QUICKSTART.md) · [Install](docs/INSTALL.md) · [Usage](docs/USAGE.md) · [Config](docs/CONFIG.md) · [Markdown](docs/MARKDOWN.md) · [Agent Guide](docs/AGENT-GUIDE.md)

</div>

## What It Does

md2x is an Agent Native CLI for Markdown-to-X Articles publishing workflows. It does four things:

- `inspect` checks Markdown readiness before you authenticate.
- `render` produces deterministic DraftJS `content_state`.
- `auth` manages X OAuth2 PKCE login for user-context API access.
- `draft` creates X Article drafts through the official X API.

V1 is draft-first. It creates drafts and does not publish by default, so terminal users and agents can stop at a reviewable state before anything goes live.

## Quick Start

```bash
npm install -g @geekjourneyx/md2x
md2x inspect article.md --json
md2x render article.md --format draftjs --json
md2x config init --client-id YOUR_X_OAUTH2_CLIENT_ID
md2x auth login
md2x draft article.md --json
```

For legacy `xurl` token stores, `md2x draft article.md --app md2x --json` remains supported.

## Why md2x Exists

X Articles are not plain Markdown. The API expects DraftJS content state, uploaded media IDs, and user-context authentication. md2x turns that into a small Agent Native CLI contract that is easy for humans to run and easy for agents to reason about.

The source file stays Markdown. The compiler path stays inspectable. The live API call is isolated to draft creation.

## Documentation

- [Quickstart](docs/QUICKSTART.md)
- [Install](docs/INSTALL.md)
- [Authentication](docs/AUTHENTICATION.md)
- [OAuth2 PKCE Tutorial](docs/OAUTH2-PKCE.md)
- [Configuration](docs/CONFIG.md)
- [Usage](docs/USAGE.md)
- [Markdown Syntax](docs/MARKDOWN.md)
- [X API Contract](docs/X-API.md)
- [Agent Guide](docs/AGENT-GUIDE.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)

## Contributing

Contributing? Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a PR.

## License

AGPL-3.0-only. Commercial licenses are available; see [COMMERCIAL.md](COMMERCIAL.md).
