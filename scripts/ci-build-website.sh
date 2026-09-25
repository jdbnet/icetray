#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

if [ -z "${VERSION:-}" ]; then
  bash scripts/set-version.sh
  VERSION="$(tr -d '[:space:]' < VERSION)"
fi

export VERSION

echo "==> Building marketing website for v${VERSION}..."
cd website
npm ci
VITE_APP_VERSION="${VERSION}" npm run build
echo "==> Website build complete: website/dist"
