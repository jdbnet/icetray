#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}"

LINUX_PACKAGES="gcc pkg-config libgtk-3-dev libayatana-appindicator3-dev libasound2-dev \
  libwebkit2gtk-4.1-dev mingw-w64"

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
wails build -tags webkit2_41 -clean -o "${OUTDIR}/icetray-linux-amd64"

echo "==> Building Linux headless binary..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o "${OUTDIR}/icetray-headless-linux-amd64" .

echo "==> Building Windows headed binaries (Wails)..."
wails build -platform windows/amd64 -clean -o "${OUTDIR}/icetray-windows-amd64.exe"
wails build -platform windows/arm64 -clean -o "${OUTDIR}/icetray-windows-arm64.exe"

echo "==> amd64 build complete:"
ls -lh "${OUTDIR}/"icetray-linux-amd64 "${OUTDIR}/"icetray-headless-linux-amd64 "${OUTDIR}/"icetray-windows-*.exe 2>/dev/null || ls -lh "${OUTDIR}/"
