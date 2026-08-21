#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}"

LINUX_PACKAGES="gcc libgtk-3-dev libayatana-appindicator3-dev libasound2-dev \
  libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev \
  libxxf86vm-dev libwayland-dev"

echo "==> Installing build dependencies..."
sudo apt-get update -qq
sudo apt-get install -y -qq ${LINUX_PACKAGES}

echo "==> Building Linux arm64 binaries..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -buildvcs=false -o "${OUTDIR}/icetray-linux-arm64" .
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o "${OUTDIR}/icetray-headless-linux-arm64" .

echo "==> arm64 build complete:"
ls -lh "${OUTDIR}/"icetray-{linux,headless-linux}-arm64
