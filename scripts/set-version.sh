#!/usr/bin/env bash
set -euo pipefail

# Sync the project version into wails.json (used for Windows exe metadata and NSIS installers).
cd "$(dirname "$0")/.."

resolve_version() {
  if [ -n "${VERSION:-}" ]; then
    printf '%s' "${VERSION}"
    return
  fi

  if [ -f VERSION ]; then
    tr -d '[:space:]' < VERSION
    return
  fi

  if git describe --tags --abbrev=0 >/dev/null 2>&1; then
    git describe --tags --abbrev=0 | sed 's/^v//'
    return
  fi

  echo "Set VERSION, add a VERSION file, or create a git tag" >&2
  exit 1
}

version="$(resolve_version)"
if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required to update wails.json" >&2
  exit 1
fi

tmp="$(mktemp)"
jq --arg v "${version}" '.info.productVersion = $v' wails.json > "${tmp}"
mv "${tmp}" wails.json

echo "==> Version ${version} synced to wails.json"
