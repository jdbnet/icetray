#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}" bin

LINUX_PACKAGES="gcc pkg-config libgtk-3-dev libayatana-appindicator3-dev libasound2-dev libwebkit2gtk-4.1-dev"

echo "==> Installing build dependencies..."
sudo apt-get update -qq
sudo apt-get install -y -qq ${LINUX_PACKAGES}

echo "==> Installing Wails v3 CLI..."
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16
export PATH="$(go env GOPATH)/bin:${PATH}"

bash scripts/set-version.sh
export VERSION="${VERSION:-$(tr -d '[:space:]' < VERSION)}"

echo "==> Building Linux headed binary (Wails v3, gtk3)..."
wails3 task linux:build ARCH=arm64 EXTRA_TAGS=gtk3
install -m 0755 bin/icetray "${OUTDIR}/icetray-linux-arm64"

echo "==> Building Linux headless binary..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o "${OUTDIR}/icetray-headless-linux-arm64" .

echo "==> Building Debian package (arm64)..."
bash scripts/ci-deb.sh arm64

echo "==> arm64 build complete:"
ls -lh "${OUTDIR}/"icetray-linux-arm64 "${OUTDIR}/"icetray-headless-linux-arm64 \
  "${OUTDIR}/"icetray_"${VERSION}"_arm64.deb
