#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}"

LINUX_PACKAGES="gcc pkg-config libgtk-3-dev libayatana-appindicator3-dev libasound2-dev libwebkit2gtk-4.1-dev"

echo "==> Installing build dependencies..."
sudo apt-get update -qq
sudo apt-get install -y -qq ${LINUX_PACKAGES}

echo "==> Building frontend..."
cd frontend
if [ -f package-lock.json ]; then
  npm ci --no-audit --no-fund
else
  npm install --no-audit --no-fund
fi
npm run build
cd ..

echo "==> Installing Wails CLI..."
go install github.com/wailsapp/wails/v2/cmd/wails@latest
export PATH="$(go env GOPATH)/bin:${PATH}"

echo "==> Building Linux headed binary (Wails)..."
wails build -tags webkit2_41 -clean -o "${OUTDIR}/icetray-linux-arm64"

echo "==> Building Linux headless binary..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o "${OUTDIR}/icetray-headless-linux-arm64" .

echo "==> arm64 build complete:"
ls -lh "${OUTDIR}/"icetray-linux-arm64 "${OUTDIR}/"icetray-headless-linux-arm64
