# Discovery and JSON Contracts

md2x JSON output uses a stable envelope so agents can discover capabilities and handle failures without scraping text.

## Envelope

```json
{
  "success": true,
  "schema_version": "v1",
  "status": "completed",
  "code": "OK",
  "message": "rendered DraftJS content_state",
  "data": {}
}
```

On failure:

```json
{
  "success": false,
  "schema_version": "v1",
  "status": "failed",
  "code": "X_DRAFT_FAILED",
  "message": "create draft returned 401 Unauthorized",
  "error": {
    "code": "X_DRAFT_FAILED",
    "message": "create draft returned 401 Unauthorized"
  }
}
```

Failure `message` values are diagnostic text from the failing operation. Agents should branch on `code`, not exact message strings. Input validation failures may include `error.diagnostics` with every local issue found in one run.

With `--json`, md2x writes both success and failure envelopes to stdout. Non-JSON human errors are written to stderr.

## V1 Compatibility Note

For `1.x`, the top-level envelope keys are stable:

- `success`
- `schema_version`
- `status`
- `code`
- `message`
- `data`
- `error`

New fields may be added inside `data` or `error`. Agents should ignore unknown fields.

## Capability Discovery

Use:

```bash
md2x --help
md2x inspect --help
md2x render --help
md2x draft --help
```

Prefer command help for flags and this document for JSON shape.

## API Payload Discovery

Use `render` before `draft` when an agent needs to inspect the X Article text payload without touching the network:

```bash
md2x render article.md --format draftjs --json
```

For articles with local images, `render` is a pre-upload preview. The live `draft` command uploads media and then injects `media_id` values into image entities. Use [Markdown Syntax](MARKDOWN.md) for supported source syntax and [X API Contract](X-API.md) for the JSON body sent to X.
