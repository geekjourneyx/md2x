#!/usr/bin/env bash
set -euo pipefail

article="${1:-testdata/articles/images.md}"

if [[ "${MD2X_LIVE_DRAFT:-}" != "1" ]]; then
  echo "set MD2X_LIVE_DRAFT=1 to create a real X Article draft" >&2
  exit 0
fi

go run ./cmd/md2x draft "$article" --app md2x --json
