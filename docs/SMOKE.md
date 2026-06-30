# Smoke Tests

Smoke tests prove the local compiler path and live draft path separately.

## Offline Smoke

```bash
md2x inspect testdata/articles/images.md --json
md2x render testdata/articles/images.md --format draftjs --json
```

Expected result:

- both commands exit `0`
- JSON envelopes contain `"success": true`
- no network credentials are required

## Live Draft Smoke

Run only after native OAuth2 PKCE has been configured:

```bash
md2x auth status
MD2X_LIVE_DRAFT=1 bash scripts/smoke-draft.sh testdata/articles/images.md
```

Legacy xurl token stores are still supported for developers who already use `xurl auth oauth2 --app md2x`.

Expected result:

- media uploads complete
- an X Article draft is created
- nothing is published by default

If `MD2X_LIVE_DRAFT` is not set to `1`, live draft smoke should be skipped.
