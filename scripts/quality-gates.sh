#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

export npm_config_cache="${npm_config_cache:-${TMPDIR:-/tmp}/md2x-npm-cache}"
export GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/md2x-go-build}"
if [[ " ${GOFLAGS:-} " != *" -buildvcs=false "* ]]; then
  export GOFLAGS="${GOFLAGS:-} -buildvcs=false"
fi

fail() {
  echo "quality-gates: $*" >&2
  exit 1
}

version="$(tr -d '[:space:]' < VERSION)"
package_version="$(node -p "require('./package.json').version")"
changelog_version="$(awk '/^## \[[^]]+\]/{gsub(/^## \[|\].*$/, ""); print; exit}' CHANGELOG.md)"

[[ -n "$version" ]] || fail "VERSION is empty"
[[ "$package_version" == "$version" ]] || fail "package.json version ${package_version} does not match VERSION ${version}"
[[ "$changelog_version" == "$version" ]] || fail "top CHANGELOG version ${changelog_version:-<missing>} does not match VERSION ${version}"

unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  echo "gofmt required:" >&2
  echo "$unformatted" >&2
  exit 1
fi

go vet ./...
bash scripts/run-golangci-lint.sh

go test -count=1 -coverprofile=coverage.out ./...
coverage="$(go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/, "", $3); print $3}')"
threshold="${MD2X_COVERAGE_THRESHOLD:-70.0}"
number_re='^[0-9]+([.][0-9]+)?$'
[[ "$coverage" =~ $number_re ]] || fail "could not parse total coverage: ${coverage:-<empty>}"
[[ "$threshold" =~ $number_re ]] || fail "MD2X_COVERAGE_THRESHOLD must be a non-negative number"
awk -v coverage="$coverage" -v threshold="$threshold" 'BEGIN { exit !(coverage + 0 >= threshold + 0) }' ||
  fail "coverage ${coverage}% is below threshold ${threshold}%"

node -c scripts/install.js >/dev/null
node -c scripts/run.js >/dev/null
npm run pack:check >/dev/null

bash scripts/release-check.sh

echo "quality-gates: OK"
