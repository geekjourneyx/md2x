# X API Contract

md2x V1 targets the official X Articles draft API. It creates reviewable drafts and does not publish by default.

## Official Endpoints

| Operation | Endpoint | md2x V1 behavior |
| --- | --- | --- |
| Upload image media | `POST /2/media/upload` | Implemented by `md2x draft` for cover and body images. |
| Chunked media upload | `POST /2/media/upload/initialize`, `append`, `finalize` | Reserved for future video/GIF/large-file support. Not the default V1 image path. |
| Create draft article | `POST /2/articles/draft` | Implemented by `md2x draft`. |
| Publish article | `POST /2/articles/{article_id}/publish` | Not implemented in V1. Users publish manually from X. |

The draft-first boundary is intentional. Publishing is irreversible enough that a CLI and agent workflow should stop at a human-reviewable draft until the product has more live API soak time.

## Authentication

The X Articles API requires user-context authentication. The official docs list OAuth 2.0 user-context scopes:

- `tweet.read`
- `tweet.write`
- `users.read`

Media upload also needs media write capability. For long-running agent workflows, native OAuth 2.0 PKCE tokens from `md2x auth login` are preferred over manually copied short-lived bearer tokens.

md2x V1 token resolution order:

1. `X_BEARER_TOKEN` environment variable, useful for CI and smoke tests.
2. Native md2x OAuth2 token store from `md2x auth login`.
3. `auth.bearer_token` from the local md2x config file.
4. `xurl` OAuth2 token store, selected with `--xurl-config`, `--app`, and `--username`.

## Create Draft JSON

`md2x draft` sends this high-level request shape to `POST /2/articles/draft`:

```json
{
  "title": "My Article",
  "content_state": {
    "blocks": [],
    "entities": []
  },
  "cover_media": {
    "media_category": "tweet_image",
    "media_id": "1234567890"
  }
}
```

`cover_media` is omitted when the Markdown frontmatter has no `cover` field. Ordinary text and list blocks omit `data` entirely; md2x does not emit `data:null`.

Local file paths and parser-only image metadata are not sent in the final image block payload. Uploaded body images are represented by image entities.

The official X schema only allows known annotation arrays inside `blocks[].data`, such as `cashtags`, `hashtags`, `mentions`, and `urls`. md2x does not send parser-only fields such as ordered-list source numbers.

## Content State Shape

The official X Articles format is a simplified DraftJS-like object with two top-level arrays:

```json
{
  "blocks": [
    {
      "key": "1",
      "text": "Welcome",
      "type": "header-one"
    },
    {
      "key": "2",
      "text": "This is the first paragraph.",
      "type": "unstyled"
    }
  ],
  "entities": []
}
```

md2x uses deterministic base36 block keys (`1`, `2`, ... `a`, `b`) so agents can diff render output across runs.

## Inline Styles

Inline styles are emitted as `inline_style_ranges`:

```json
{
  "text": "Bold text",
  "type": "unstyled",
  "inline_style_ranges": [
    { "offset": 0, "length": 4, "style": "bold" }
  ]
}
```

Supported styles:

- `bold`
- `italic`
- `strikethrough`

Offsets and lengths are UTF-16 code units, matching the X Article JSON expectation.

## Link Entities

Markdown links become `link` entities plus `entity_ranges`:

```json
{
  "blocks": [
    {
      "key": "1",
      "text": "Visit X",
      "type": "unstyled",
      "entity_ranges": [{ "offset": 6, "length": 1, "key": 0 }]
    }
  ],
  "entities": [
    {
      "key": "0",
      "value": {
        "type": "link",
        "mutability": "mutable",
        "data": { "url": "https://x.com" }
      }
    }
  ]
}
```

## Image Entities

Images are uploaded before draft creation. V1 uses the single-step `POST /2/media/upload` endpoint for `.png`, `.jpg`, `.jpeg`, and `.webp` images because the official endpoint supports `tweet_image` directly and avoids the extra initialize/append/finalize requests needed by chunked upload.

Live X API requests use a finite HTTP timeout. The default is `120s`, configurable with `api.timeout`, `MD2X_HTTP_TIMEOUT`, or `draft --api-timeout`.

Within one `draft` command, md2x fingerprints local image files by media type, size, and SHA-256. Duplicate image contents are uploaded once and the resulting `media_id` is reused for every cover or body image reference in that draft.

Body images are then emitted as an `atomic` block and an `image` entity:

```json
{
  "blocks": [
    {
      "key": "1",
      "text": " ",
      "type": "atomic",
      "entity_ranges": [{ "offset": 0, "length": 1, "key": 0 }]
    }
  ],
  "entities": [
    {
      "key": "0",
      "value": {
        "type": "image",
        "mutability": "immutable",
        "data": {
          "caption": "Diagram",
          "media_items": [
            { "media_category": "tweet_image", "media_id": "1234567890" }
          ]
        }
      }
    }
  ]
}
```

Do not persist md2x media IDs as a long-term cache. X media upload responses can expire, so md2x only deduplicates inside the current command.

## Publish Boundary

The publish endpoint exists, but md2x V1 does not call it. The first release keeps one live-writing side effect:

```text
Markdown -> inspect -> render -> upload media -> create draft
```

Publishing can be added later as a separate explicit command, for example `md2x publish <article_id>`, with its own confirmation, exit codes, JSON contract, and live smoke test.
