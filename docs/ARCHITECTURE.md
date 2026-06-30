# Architecture

md2x is compiler-first. The CLI should parse once into a shared article model, then render or draft from that model.

## Pipeline

```text
cmd/md2x
  -> internal/cli
  -> internal/markdown
  -> internal/article
  -> internal/draftjs
  -> internal/xapi
```

## Layers

| Layer | Responsibility |
| --- | --- |
| `cmd/md2x` | process entrypoint |
| `internal/cli` | flags, command routing, exit codes, JSON envelope |
| `internal/markdown` | Markdown and frontmatter parsing |
| `internal/article` | normalized article model and validation |
| `internal/draftjs` | deterministic DraftJS `content_state` rendering |
| `internal/xapi` | X authentication, media upload, draft creation |

## Core Invariant

`inspect`, `render`, and `draft` consume the same parsed article model.

This keeps offline validation, deterministic rendering, and live draft creation aligned. A file that passes `inspect` should not be reparsed differently by `render` or `draft`.

## Boundary

V1 does not introduce a universal intermediate representation beyond the article model. The output target is X Articles DraftJS content state.

`xurl` compatibility is an authentication convenience, not the product identity.
