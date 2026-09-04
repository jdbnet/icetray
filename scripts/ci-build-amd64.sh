#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}" bin

LINUX_PACKAGES="gcc pkg-config libgtk-3-dev libayatana-appindicator3-dev libasound2-dev \
  libwebkit2gtk-4.1-dev mingw-w64 nsis"

echo "==> Installing build dependencies..."
sudo apt-get update -qq
sudo apt-get install -y -qq ${LINUX_PACKAGES}

echo "==> Installing Wails v3 CLI..."
# The CLI links wails/v3/internal/operatingsystem, which pkg-configs gtk4
# unless CGO is off. Ubuntu runners do not have gtk4.
CGO_ENABLED=0 go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16
export PATH="$(go env GOPATH)/bin:${PATH}"

bash scripts/set-version.sh
export VERSION="${VERSION:-$(tr -d '[:space:]' < VERSION)}"

echo "==> Building Linux headed binary (Wails v3, gtk3)..."
wails3 task linux:build ARCH=amd64 EXTRA_TAGS=gtk3
install -m 0755 bin/icetray "${OUTDIR}/icetray-linux-amd64"

echo "==> Building Linux headless binary..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o "${OUTDIR}/icetray-headless-linux-amd64" .

echo "==> Building Windows binaries and NSIS installers..."
wails3 task windows:package ARCH=amd64
install -m 0755 bin/icetray.exe "${OUTDIR}/icetray-windows-amd64.exe"
install -m 0755 bin/icetray-amd64-installer.exe "${OUTDIR}/icetray-windows-amd64-setup.exe"

wails3 task windows:package ARCH=arm64
install -m 0755 bin/icetray.exe "${OUTDIR}/icetray-windows-arm64.exe"
install -m 0755 bin/icetray-arm64-installer.exe "${OUTDIR}/icetray-windows-arm64-setup.exe"

echo "==> Building Debian package (amd64)..."
bash scripts/ci-deb.sh amd64

echo "==> amd64 build complete:"
ls -lh "${OUTDIR}/"icetray-linux-amd64 "${OUTDIR}/"icetray-headless-linux-amd64 \
  "${OUTDIR}/"icetray-windows-amd64.exe "${OUTDIR}/"icetray-windows-arm64.exe \
  "${OUTDIR}/"icetray-windows-amd64-setup.exe "${OUTDIR}/"icetray-windows-arm64-setup.exe \
  "${OUTDIR}/"icetray_"${VERSION}"_amd64.deb
