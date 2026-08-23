#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}"

LINUX_PACKAGES="gcc pkg-config libgtk-3-dev libayatana-appindicator3-dev libasound2-dev libwebkit2gtk-4.1-dev"

echo "==> Installing build dependencies..."
sudo apt-get update -qq
sudo apt-get install -y -qq ${LINUX_PACKAGES}

echo "==> Installing Wails CLI..."
go install github.com/wailsapp/wails/v2/cmd/wails@latest
export PATH="$(go env GOPATH)/bin:${PATH}"

bash scripts/set-version.sh

echo "==> Building Linux headed binary (Wails)..."
wails build -tags webkit2_41 -clean -o icetray-linux-arm64
install -m 0755 build/bin/icetray-linux-arm64 "${OUTDIR}/icetray-linux-arm64"

echo "==> Building Linux headless binary..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o "${OUTDIR}/icetray-headless-linux-arm64" .

echo "==> arm64 build complete:"
ls -lh "${OUTDIR}/"icetray-linux-arm64 "${OUTDIR}/"icetray-headless-linux-arm64
