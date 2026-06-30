# Design Principles

md2x is a small CLI for getting Markdown into X Articles drafts with as little hidden state as possible.

## Draft-First

V1 creates drafts and does not publish by default. Publishing is a separate human decision unless a future version adds an explicit command and contract.

## Offline-First

`inspect` and `render` work without network access. Agents can validate content and review DraftJS output before using live credentials.

## Unix-Style IO

Commands should be scriptable:

- structured JSON on stdout with `--json`, including failures
- human diagnostics on stderr for non-JSON output
- meaningful exit codes
- no interactive prompts in automation paths

## No Universal IR in V1

V1 does not attempt to define a universal publishing IR. It uses a normalized article model internally and targets X Articles DraftJS content state directly.

## Small Contracts

Prefer stable command contracts over clever inference. Markdown remains the source of truth, and the CLI should make each transformation inspectable.
