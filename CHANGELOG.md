# Changelog

## [1.0.5] - 2026-07-01

- Added a hard timeout and bounded retry window to the release workflow's cnpm/npmmirror sync step.
- Allowed the release workflow to continue when npmmirror already reports the target version after a cnpm sync timeout.
- Updated release checks and release documentation to require bounded cnpm sync behavior.

Breaking changes: none.

Migration notes: no CLI migration is required.

Verification summary: release checks, quality gates, npm pack check, and local build were run before tagging.

## [1.0.4] - 2026-07-01

- Changed V1 image uploads to use the single-step X `POST /2/media/upload` endpoint for `.png`, `.jpg`, `.jpeg`, and `.webp` files.
- Added per-command image deduplication by media type, file size, and SHA-256 so duplicate cover/body images reuse one uploaded `media_id`.
- Added `inspect --json` request estimation fields: `unique_media_count` and `estimated_x_requests`.
- Kept chunked media upload internals covered by tests for future video, GIF, or large-file support.
- Updated X API, Markdown, usage, agent, and troubleshooting docs for the new image upload path and cost-aware workflow.

Breaking changes: none.

Migration notes: users seeing `/2/media/upload/initialize` timeouts for normal image uploads should upgrade and retry. Agents should read `inspect --json` request estimates before live `draft` calls.

Verification summary: full Go tests, quality gates, release checks, npm pack check, and local build were run before tagging.

## [1.0.3] - 2026-07-01

- Fixed X Article draft creation for Markdown without links or media by serializing empty DraftJS `entities` as `[]` instead of `null`.
- Added defensive request normalization so draft payload `blocks` and `entities` remain arrays at the X API boundary.
- Added structured X API failure details for rate limits and transient server errors, including retryability and `x-rate-limit-*` reset metadata when X returns it.
- Improved test isolation so local config and native OAuth token stores cannot leak into no-token tests.
- Documented 429 troubleshooting and retry guidance.

Breaking changes: none.

Migration notes: users who hit X API `400 Bad Request` for `content_state.entities: null found, array expected` should upgrade and retry the same Markdown. Users seeing `429 Too Many Requests` should inspect `error.x_api.rate_limit` in JSON output and retry after `reset_at` when present.

Verification summary: full Go tests, quality gates, release checks, npm pack check, and local build were run before tagging.

## [1.0.2] - 2026-07-01

- Fixed X Article draft creation for ordered lists by omitting parser-only `data.number` fields from DraftJS blocks.
- Added unit and e2e coverage to prevent internal ordered-list metadata from leaking into the live X payload.
- Calibrated Markdown and X API docs with the official `blocks[].data` schema constraints.

Breaking changes: none.

Migration notes: users who hit X API `400 Bad Request` for `content_state.blocks[].data.number` should upgrade and retry the same Markdown.

Verification summary: full Go tests, quality gates, release checks, npm pack check, and local build were run before tagging.

## [1.0.1] - 2026-07-01

- Fixed the default config path to use `${XDG_CONFIG_HOME:-~/.config}/md2x/config.yaml` on every platform.
- Clarified OAuth2 PKCE login progress after the browser callback and before token storage.
- Added release automation and docs for cnpm/npmmirror synchronization after npm publish.

Breaking changes: none.

Migration notes: if md2x previously created a config at `~/Library/Application Support/md2x/config.yaml` on macOS, move it to `~/.config/md2x/config.yaml` or set `MD2X_CONFIG` explicitly.

Verification summary: release checks, package checks, build, focused config/auth tests, and compile-only full package tests were run locally before tagging.

## [1.0.0] - 2026-06-30

- Initial release-ready Go CLI scaffold.
