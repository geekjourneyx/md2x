#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

export npm_config_cache="${npm_config_cache:-${TMPDIR:-/tmp}/md2x-npm-cache}"
export GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/md2x-go-build}"

fail() {
  echo "release-check: $*" >&2
  exit 1
}

contains() {
  local file="$1"
  local needle="$2"
  local message="$3"
  grep -Fq -- "$needle" "$file" || fail "$message"
}

version="${1:-}"
if [[ -z "$version" ]]; then
  version="$(tr -d '[:space:]' < VERSION)"
fi

[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "invalid semver x.y.z version: $version"

required_files=(
  "VERSION"
  "AGENTS.md"
  "README.md"
  "README_ZH.md"
  "CONTRIBUTING.md"
  "CHANGELOG.md"
  "COMMERCIAL.md"
  "LICENSE"
  "go.mod"
  "Makefile"
  "package.json"
  "assets/banner.webp"
  "docs/AGENT-GUIDE.md"
  "docs/ARCHITECTURE.md"
  "docs/AUTHENTICATION.md"
  "docs/OAUTH2-PKCE.md"
  "docs/CONFIG.md"
  "docs/DESIGN.md"
  "docs/DISCOVERY.md"
  "docs/INSTALL.md"
  "docs/MARKDOWN.md"
  "docs/QUICKSTART.md"
  "docs/RELEASE.md"
  "docs/README.md"
  "docs/SMOKE.md"
  "docs/TROUBLESHOOTING.md"
  "docs/USAGE.md"
  "docs/X-API.md"
  "scripts/install.js"
  "scripts/run.js"
  "scripts/quality-gates.sh"
  "scripts/run-golangci-lint.sh"
  "scripts/smoke-draft.sh"
  ".github/workflows/ci.yml"
  ".github/workflows/release.yml"
)

for file in "${required_files[@]}"; do
  [[ -f "$file" ]] || fail "missing required file: $file"
done

version_file="$(tr -d '[:space:]' < VERSION)"
package_name="$(node -p "require('./package.json').name")"
package_version="$(node -p "require('./package.json').version")"
changelog_version="$(awk '/^## \[[^]]+\]/{gsub(/^## \[|\].*$/, ""); print; exit}' CHANGELOG.md)"
changelog_heading="$(awk '/^## \[[^]]+\] - [0-9]{4}-[0-9]{2}-[0-9]{2}$/{print; exit}' CHANGELOG.md)"

[[ "$version_file" == "$version" ]] || fail "VERSION ${version_file} does not match ${version}"
[[ "$package_name" == "@geekjourneyx/md2x" ]] || fail "package name must be @geekjourneyx/md2x"
[[ "$package_version" == "$version" ]] || fail "package.json version ${package_version} does not match ${version}"
[[ "$changelog_version" == "$version" ]] || fail "top CHANGELOG version ${changelog_version:-<missing>} does not match ${version}"
[[ "$changelog_heading" == "## [${version}] - "* ]] || fail "top CHANGELOG entry must be '## [${version}] - YYYY-MM-DD'"
package_license="$(node -p "require('./package.json').license")"
[[ "$package_license" == "AGPL-3.0-only" ]] || fail "package.json license must be AGPL-3.0-only"
contains "internal/cli/root.go" "var version = \"$version\"" "internal CLI fallback version must match release version"

github_ref_name="${GITHUB_REF_NAME:-}"
github_ref_type="${GITHUB_REF_TYPE:-}"
github_ref="${GITHUB_REF:-}"
if [[ "$github_ref_type" == "tag" || "$github_ref" == refs/tags/* ]]; then
  [[ "$github_ref_name" == "v$version" ]] || fail "git ref ${github_ref_name:-${github_ref}} does not match v${version}"
fi

contains "README.md" "npm install -g @geekjourneyx/md2x" "README must document npm install"
contains "README.md" "README_ZH.md" "README must link Chinese README"
contains "README.md" "Chinese](README_ZH.md)" "README must use English label for Chinese README link"
contains "README.md" "assets/banner.webp" "README must reference assets/banner.webp"
contains "README.md" "AGPL-3.0-only" "README must document AGPL license"
contains "README.md" "COMMERCIAL.md" "README must link commercial license notice"
contains "README.md" "CONTRIBUTING.md" "README must link contributing guide"
contains "README.md" "docs/QUICKSTART.md" "README must link docs/QUICKSTART.md"
contains "README.md" "md2x inspect article.md --json" "README must show inspect command"
contains "README.md" "md2x auth login" "README must show native auth login"
contains "README.md" "md2x draft article.md --json" "README must show native draft command"
contains "package.json" "\"docs/*.md\"" "package must include user docs"
contains "package.json" "\"assets/banner.webp\"" "package must include README banner asset"
contains "package.json" "\"README_ZH.md\"" "package must include Chinese README"
contains "package.json" "\"CONTRIBUTING.md\"" "package must include contributing guide"
contains "package.json" "\"COMMERCIAL.md\"" "package must include commercial license notice"
contains "COMMERCIAL.md" "AGPL-3.0-only" "COMMERCIAL must reference AGPL license"
contains ".github/workflows/release.yml" "cp docs/*.md" "release workflow must include user docs in GitHub artifacts"
contains ".github/workflows/release.yml" "cp assets/banner.webp" "release workflow must include README banner asset in GitHub artifacts"
contains ".github/workflows/release.yml" "cp README.md README_ZH.md CONTRIBUTING.md LICENSE CHANGELOG.md COMMERCIAL.md" "release workflow must include root docs and notices in GitHub artifacts"

contains "docs/README.md" "Quickstart" "docs index must link quickstart"
contains "docs/README.md" "Release Process" "docs index must link release process"
contains "docs/AUTHENTICATION.md" "md2x auth login" "AUTHENTICATION must include native login"
contains "docs/OAUTH2-PKCE.md" "https://docs.x.com/fundamentals/authentication/oauth-2-0/user-access-token" "OAUTH2-PKCE must link X OAuth2 docs"

contains ".github/workflows/ci.yml" "run: bash scripts/quality-gates.sh" "CI must run quality gates"
contains ".github/workflows/ci.yml" "run: npm run pack:check" "CI must run npm pack check"
contains ".github/workflows/release.yml" "run: make quality-gates" "release workflow must run full quality gates before publishing"
contains ".github/workflows/release.yml" "npm pack --pack-destination" "release workflow must smoke-pack npm package before publishing"
contains ".github/workflows/release.yml" "npm publish --access public" "release workflow must publish npm"
contains ".github/workflows/release.yml" "NPM_TOKEN" "release workflow must use NPM_TOKEN"
contains ".github/workflows/release.yml" "npx --yes cnpm sync @geekjourneyx/md2x" "release workflow must sync cnpm after npm publish"
contains ".github/workflows/release.yml" "registry.npmmirror.com" "release workflow must verify npmmirror after cnpm sync"
contains "AGENTS.md" "npx cnpm sync @geekjourneyx/md2x" "AGENTS must document cnpm sync release rule"
contains "docs/RELEASE.md" "npx cnpm sync @geekjourneyx/md2x" "release docs must document cnpm sync"

node -c scripts/install.js >/dev/null
node -c scripts/run.js >/dev/null
npm run pack:check >/dev/null

echo "release-check: OK (version ${version})"
