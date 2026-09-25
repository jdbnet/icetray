#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}" bin

echo "==> Installing Wails v3 CLI..."
CGO_ENABLED=0 go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16
export PATH="$(go env GOPATH)/bin:${PATH}"

bash scripts/set-version.sh
export VERSION="${VERSION:-$(tr -d '[:space:]' < VERSION)}"

echo "==> Building macOS universal .app (adhoc signed on runner)..."
wails3 task darwin:package:universal

if [ ! -d "bin/icetray.app" ]; then
  echo "Expected bin/icetray.app after package:universal" >&2
  exit 1
fi

ZIP_PATH="${OUTDIR}/icetray-macos-universal.zip"
echo "==> Creating ${ZIP_PATH}..."
ditto -c -k --sequesterRsrc --keepParent "bin/icetray.app" "${ZIP_PATH}"

echo "==> macOS build complete:"
ls -lh "${ZIP_PATH}"
