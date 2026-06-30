# Contributing

Thank you for considering a contribution to md2x.

Before opening a pull request, read [AGENTS.md](AGENTS.md). It is the repository-level operating manual for development, testing, documentation, release checks, versioning, and publishing.

## Development Rules

- Keep changes scoped to one behavior or release concern.
- Preserve CLI and JSON contracts unless the pull request explicitly changes them.
- Add or update tests for behavior changes.
- Do not print or commit secrets.
- Keep `README.md` in English. Use `README_ZH.md` for Chinese-facing guidance.
- Update docs in the same pull request when behavior, configuration, authentication, or release flow changes.

## Required Checks

Run these before opening a pull request:

```bash
go test -count=1 ./...
bash scripts/release-check.sh
npm run pack:check
```

For release-impacting changes, also run:

```bash
bash scripts/quality-gates.sh
make build
```

Some tests use local `httptest` servers. If your environment blocks loopback listeners, rerun the same tests in an environment that permits local port binding.

## Release Changes

For release work:

- Update `CHANGELOG.md` before tagging.
- Keep `VERSION`, `package.json`, and the top `CHANGELOG.md` entry aligned.
- Use annotated tags: `git tag -a vX.Y.Z -m "vX.Y.Z"`.
- Push the tag to trigger the GitHub release workflow.
- Confirm npm and npmmirror both show the released version after the workflow completes.

Do not manually publish npm unless the GitHub release workflow is unavailable and the same gates have passed locally. If a manual npm fallback is required, run `npx cnpm sync @geekjourneyx/md2x` after `npm publish --access public`.
