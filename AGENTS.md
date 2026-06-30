# AGENTS.md

Scope: this file applies to the entire repository.

md2x is a Go CLI for converting Markdown into X Articles drafts. Treat the CLI as a stable automation contract for both humans and agents.

## Development Flow

- Read the relevant package and tests before editing.
- Keep changes scoped to the requested behavior.
- Preserve existing CLI contracts unless the change explicitly requires a new contract.
- Prefer adding fields over changing or removing JSON fields.
- Keep human output readable and JSON output machine-stable.
- Do not print secrets. Redact tokens in config, auth, logs, errors, and JSON envelopes.
- For behavior changes, add or update focused tests before or with the implementation.

## Core Commands

Use the project scripts instead of ad hoc command variants:

```bash
make build
go test -count=1 ./...
bash scripts/release-check.sh
bash scripts/quality-gates.sh
npm run pack:check
```

If Go tests need a writable cache, use:

```bash
GOCACHE=/tmp/md2x-go-build go test -count=1 ./...
```

Some tests use `httptest` and need local loopback port binding. If the sandbox blocks local listeners, rerun the same test command with the required approval instead of weakening tests.

## Testing Closed Loop

- Local validation must run before auth resolution in `draft`.
- Offline commands (`inspect`, `render`) must not require credentials or network access.
- Live-path tests should mock X endpoints with `httptest`.
- Keep coverage above the threshold enforced by `scripts/quality-gates.sh`.
- Do not lower quality gates to make a change pass.
- After touching auth, draft, xapi, config, release scripts, or package metadata, run `bash scripts/quality-gates.sh`.

## Documentation Calibration

Docs are part of the product. Update them in the same change as behavior:

- Root `README.md` for the first-run path and project positioning.
- `docs/README.md` when adding or removing docs.
- `docs/QUICKSTART.md` for the shortest human path.
- `docs/AGENT-GUIDE.md` for agent workflow changes.
- `docs/AUTHENTICATION.md` and `docs/OAUTH2-PKCE.md` for auth changes.
- `docs/CONFIG.md` for config fields, env vars, or priority changes.
- `docs/X-API.md` for request body or endpoint changes.
- `docs/TROUBLESHOOTING.md` for new error codes or remediation steps.

Keep docs consistent with implemented commands. When docs mention a command, that command must exist and be covered by release checks when practical.

## Release Checks

Before release or publish, these must pass:

```bash
bash scripts/quality-gates.sh
bash scripts/release-check.sh
npm run pack:check
make build
```

`scripts/release-check.sh` is the source of truth for required repository files, README invariants, docs invariants, package metadata, and workflow expectations.

## Version Release Norms

- Version starts at `1.0.0` and follows semver.
- Before every release, update `CHANGELOG.md` with the target version, release date, user-visible changes, breaking changes, migration notes, and verification summary.
- Keep `VERSION`, `package.json`, and top `CHANGELOG.md` entry aligned.
- GitHub release tags use `vX.Y.Z`.
- Push the release tag to trigger the GitHub release workflow: `git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`.
- Do not publish npm manually unless the GitHub release workflow is unavailable and the same release gates have passed locally.
- Release artifacts must include docs and `assets/banner.webp`.
- npm package metadata must remain `@geekjourneyx/md2x`.
- npm license must remain `AGPL-3.0-only` unless the project owner explicitly changes licensing.
- Commercial licensing notice must remain in `COMMERCIAL.md`.

## Auth And Secrets

- Preferred auth path is native OAuth2 PKCE: `md2x auth login`.
- `X_BEARER_TOKEN` remains the highest-priority override for CI and smoke tests.
- Native OAuth tokens live in the local token store, not in normal documentation examples.
- `~/.config/md2x/config.yaml` is for non-secret defaults and legacy direct token support.
- Token files must be written with private permissions.

## Publishing

- Do not publish from an unverified tree.
- First npm publish creates the package automatically when the scope is owned by the publisher.
- Scoped public npm publish requires `npm publish --access public`.
- The release workflow must run quality gates before publishing npm artifacts.
