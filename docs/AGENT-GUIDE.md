# Agent Guide

md2x is designed for automation. Treat it as a small deterministic CLI contract around a live draft API.

## Deterministic CLI API Rules

- Always pass `--json` in agent workflows.
- Run `inspect` before `render` or `draft`.
- Use `md2x config show --json` to inspect effective non-secret defaults.
- Use `md2x auth status --json` before live `draft` calls.
- Run `render` before `draft` when reviewing generated text content.
- Treat stdout as the machine contract for every `--json` command, including failures.
- Treat stderr as human diagnostic text for non-JSON output only.
- Never parse terminal color or progress output.
- Do not assume `draft` published anything. V1 creates drafts only.
- Stop on non-zero exit codes.

## Safe Workflow

```bash
md2x inspect article.md --json
md2x render article.md --format draftjs --json
md2x auth status --json
md2x draft article.md --json
```

If credentials are unavailable, stop after `render` and report that live draft creation was skipped.

For articles with local images, `render` is a pre-upload preview. The final `draft` request includes uploaded `media_id` values that do not exist until after the media upload step.

Read `inspect --json` fields before calling `draft`:

- `unique_media_count`: number of unique image contents that will be uploaded.
- `estimated_x_requests.media_upload`: estimated media upload calls.
- `estimated_x_requests.total`: estimated media upload calls plus the draft creation call.

Duplicate image contents are uploaded once within a single `draft` command. Do not cache media IDs between commands; X media IDs can expire.

## Retry Rules

- Do not retry validation errors without changing input.
- Retry authentication errors only after refreshing the token.
- Retry X API failures only when the response is transient or rate-limit related.
- Preserve the original Markdown file as the source of truth.

## File Handling

Use relative media paths from the Markdown file location. Avoid moving images between `inspect` and `draft`; the parsed media list should stay stable across commands.
