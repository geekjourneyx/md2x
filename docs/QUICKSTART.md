# Quickstart

md2x turns Markdown into X Articles drafts. V1 is draft-first: it creates drafts and does not publish by default.

## 1. Install

```bash
npm install -g @geekjourneyx/md2x
md2x version
```

For a source checkout:

```bash
make build
./bin/md2x version --json
```

## 2. Create an Article

Create `article.md`:

```markdown
---
title: Shipping Markdown to X Articles
cover: ./cover.png
---

# Shipping Markdown to X Articles

Markdown is the source of truth.

![Architecture sketch](./diagram.png)

The CLI compiles this file into the DraftJS content state expected by X Articles.
```

Frontmatter is the article contract. `title` names the draft. `cover` points to the cover image that should be uploaded before draft creation.

Create the referenced image files before running a live draft command:

```bash
test -f cover.png
test -f diagram.png
```

For a text-only first run, remove the `cover` field and the Markdown image.

## 3. Inspect

Use `inspect` before any live API call:

```bash
md2x inspect article.md --json
```

`inspect` parses the Markdown, validates frontmatter, discovers local media, and returns machine-readable JSON.

## 4. Render

Render the deterministic DraftJS payload offline:

```bash
md2x render article.md --format draftjs --json
```

This command does not authenticate and does not contact X. It is the safest command for CI, review, and agent planning. For articles with local images, `render` is a pre-upload preview; `draft` adds uploaded media IDs before calling X.

## 5. Authenticate

md2x should use an official X OAuth2 user-context token. Native OAuth2 PKCE is the recommended path:

```bash
md2x config init --client-id YOUR_X_OAUTH2_CLIENT_ID
md2x auth login
md2x auth status
```

The token must include:

- `tweet.read`
- `tweet.write`
- `users.read`
- `media.write`
- `offline.access`

See [OAuth2 PKCE Tutorial](OAUTH2-PKCE.md) for the X Developer Portal setup.

## 6. Create a Draft

```bash
md2x draft article.md --json
```

The command uploads required media, converts the article body to DraftJS `content_state`, and creates an X Article draft through the official X API.

V1 stops at draft creation. Review and publish in X unless a future version explicitly adds a separate publish command.
