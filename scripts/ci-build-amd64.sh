#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}"

LINUX_PACKAGES="gcc pkg-config libgtk-3-dev libayatana-appindicator3-dev libasound2-dev \
  libwebkit2gtk-4.1-dev mingw-w64 nsis"

echo "==> Installing build dependencies..."
sudo apt-get update -qq
sudo apt-get install -y -qq ${LINUX_PACKAGES}

echo "==> Installing Wails CLI..."
go install github.com/wailsapp/wails/v2/cmd/wails@latest
export PATH="$(go env GOPATH)/bin:${PATH}"

bash scripts/set-version.sh
export VERSION="${VERSION:-$(jq -r '.info.productVersion' wails.json)}"

echo "==> Building Linux headed binary (Wails)..."
wails build -tags webkit2_41 -clean -o icetray-linux-amd64
install -m 0755 build/bin/icetray-linux-amd64 "${OUTDIR}/icetray-linux-amd64"

echo "==> Building Linux headless binary..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o "${OUTDIR}/icetray-headless-linux-amd64" .

echo "==> Building Windows binaries and NSIS installers (Wails)..."
wails build -platform windows/amd64,windows/arm64 --nsis -clean
install -m 0755 build/bin/icetray-amd64.exe "${OUTDIR}/icetray-windows-amd64.exe"
install -m 0755 build/bin/icetray-arm64.exe "${OUTDIR}/icetray-windows-arm64.exe"
install -m 0755 build/bin/icetray-amd64-installer.exe "${OUTDIR}/icetray-windows-amd64-setup.exe"
install -m 0755 build/bin/icetray-arm64-installer.exe "${OUTDIR}/icetray-windows-arm64-setup.exe"

echo "==> Building Debian package (amd64)..."
bash scripts/ci-deb.sh amd64

echo "==> amd64 build complete:"
ls -lh "${OUTDIR}/"icetray-linux-amd64 "${OUTDIR}/"icetray-headless-linux-amd64 \
  "${OUTDIR}/"icetray-windows-amd64.exe "${OUTDIR}/"icetray-windows-arm64.exe \
  "${OUTDIR}/"icetray-windows-amd64-setup.exe "${OUTDIR}/"icetray-windows-arm64-setup.exe \
  "${OUTDIR}/"icetray_"${VERSION}"_amd64.deb
