# Release Process

md2x releases are tag-driven. The GitHub release workflow is the normal release path and must remain the source of truth for GitHub artifacts, npm publishing, and cnpm/npmmirror synchronization.

## Required Local Checks

Run these before creating a release tag:

```bash
bash scripts/quality-gates.sh
bash scripts/release-check.sh
npm run pack:check
make build
```

Do not lower gates to make a release pass. Fix the failing invariant instead.

## Tag Release

Keep `VERSION`, `package.json`, and the top `CHANGELOG.md` entry aligned, then create and push an annotated tag:

```bash
git tag -a vX.Y.Z -m "vX.Y.Z"
git push origin vX.Y.Z
```

The tag triggers `.github/workflows/release.yml`.

## Workflow Contract

The release workflow must:

- run quality gates before publishing
- build Linux, macOS, and Windows binaries
- smoke-test the npm installer against release artifacts
- publish GitHub release assets and `SHA256SUMS`
- publish `@geekjourneyx/md2x` to npm with `npm publish --access public`
- sync cnpm/npmmirror after npm publish with `npx cnpm sync @geekjourneyx/md2x`
- verify `https://registry.npmmirror.com` reports the same released version

The required GitHub secret is `NPM_TOKEN`.

## Manual Fallback

Manual npm publishing is only a fallback when the GitHub release workflow is unavailable and all local gates have passed.

```bash
npm publish --access public
npx cnpm sync @geekjourneyx/md2x
npm view @geekjourneyx/md2x version
npm view @geekjourneyx/md2x version --registry=https://registry.npmmirror.com
```

Both npm and npmmirror must report the release version before the release is considered complete.
