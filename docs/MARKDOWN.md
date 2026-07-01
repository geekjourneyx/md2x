# Markdown Syntax

md2x intentionally supports the Markdown subset that maps cleanly to X Articles. The goal for V1 is predictable draft creation, not full GitHub Flavored Markdown emulation.

## Supported Metadata

Use YAML frontmatter at the top of the file:

```markdown
---
title: My Article
cover: ./cover.png
---
```

| Field | Required | Behavior |
| --- | --- | --- |
| `title` | No | Used as the X Article title. If omitted, md2x uses the first heading, then `Untitled Article`. |
| `cover` | No | Local cover image path. `draft` uploads it before creating the article draft. |

## Supported Blocks

| Markdown | md2x article model | X Article block type |
| --- | --- | --- |
| Paragraph | `paragraph` | `unstyled` |
| `# Heading` | `heading` level 1 | `header-one` |
| `## Heading` | `heading` level 2 | `header-two` |
| `### Heading` and deeper | `heading` level 3+ | `header-three` |
| `> Quote` | `blockquote` | `blockquote` |
| `- Item` / `* Item` | `unordered-list-item` | `unordered-list-item` |
| `1. Item` | `ordered-list-item` | `ordered-list-item` |
| Image-only paragraph | `image` asset | `atomic` image entity after upload |
| Fenced code block | `code` with warning | `unstyled` plain text |

Ordered list source numbers are normalized by the Markdown parser for internal inspection. md2x does not send those parser-only numbers to X; X controls final numbering in the article editor.

## Supported Inline Syntax

| Markdown | X Article representation |
| --- | --- |
| `**bold**` / `__bold__` | Inline style `bold` |
| `*italic*` / `_italic_` | Inline style `italic` |
| `~~strike~~` | Inline style `strikethrough` |
| `[text](https://example.com)` | Link entity |
| Autolink URL | Plain text URL |

Offsets are converted from UTF-8 byte offsets to UTF-16 code unit offsets before JSON output. This matters for emoji and non-ASCII text.

## Images

Body images are recognized only when the image is the only meaningful content in a paragraph:

```markdown
![Diagram](./diagram.png)
```

During `draft`, md2x uploads each local body image and renders it as an `atomic` block with an `image` entity:

```json
{
  "type": "atomic",
  "text": " ",
  "entity_ranges": [{ "offset": 0, "length": 1, "key": 0 }]
}
```

Supported local image extensions are `.png`, `.jpg`, `.jpeg`, and `.webp`. GIF is rejected in V1 because the media upload path is image-only and deterministic. md2x also checks image headers and rejects files whose contents do not match their extension before authentication or upload.

Image upload cost and request count are based on unique image contents in a single `draft` command. If the same file, or two files with identical bytes and media type, is used more than once, md2x uploads it once and reuses the returned `media_id`.

Use `inspect --json` before `draft` to see the estimated X request count:

```json
{
  "unique_media_count": 2,
  "estimated_x_requests": {
    "media_upload": 2,
    "create_draft": 1,
    "total": 3
  }
}
```

## Not First-Class In V1

These constructs are not treated as native X Article features in V1:

- Tables
- Task lists
- Footnotes
- HTML blocks
- Inline code styling
- Nested list structure beyond flattened list item text
- Mixed text plus image in the same paragraph
- Multiple images in one paragraph

Unsupported or lossy constructs should be checked with:

```bash
md2x inspect article.md --json
md2x render article.md --format draftjs --json
```

`inspect` is the readiness gate. `render` shows the pre-upload `content_state`; `draft` uploads media and then injects final media IDs before sending the request to X.
