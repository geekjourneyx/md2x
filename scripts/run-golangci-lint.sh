#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

export GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/md2x-go-build}"
export GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-${TMPDIR:-/tmp}/md2x-golangci-lint-cache}"
if [[ " ${GOFLAGS:-} " != *" -buildvcs=false "* ]]; then
  export GOFLAGS="${GOFLAGS:-} -buildvcs=false"
fi

if command -v go >/dev/null 2>&1; then
  go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0 run ./... "$@"
elif command -v golangci-lint >/dev/null 2>&1; then
  echo "go is unavailable; falling back to installed golangci-lint" >&2
  golangci-lint run ./... "$@"
else
  echo "go and golangci-lint are unavailable; cannot run lint gate" >&2
  exit 1
fi
