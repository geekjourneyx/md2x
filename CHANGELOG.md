# Changelog

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
