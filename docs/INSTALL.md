# Install

md2x is distributed as a Unix-style CLI.

## npm

```bash
npm install -g @geekjourneyx/md2x
md2x version
```

## npmmirror

Users in regions where the public npm registry is slow can install from npmmirror after a release has synced:

```bash
npm install -g @geekjourneyx/md2x --registry=https://registry.npmmirror.com
md2x version
```

## From Source

```bash
git clone https://github.com/geekjourneyx/md2x.git
cd md2x
make build
./bin/md2x version
```

## Verify the Install

Create `article.md`:

```markdown
---
title: Install Check
---

# Install Check

If inspect and render work, the local compiler path is healthy.
```

Run:

```bash
md2x inspect article.md --json
md2x render article.md --format draftjs --json
```

Both commands should exit `0`. They are offline checks and do not require X authentication.

## Next Step

Continue with [Quickstart](QUICKSTART.md) for the draft workflow.
