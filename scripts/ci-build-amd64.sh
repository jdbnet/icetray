#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}"

LINUX_PACKAGES="gcc libgtk-3-dev libayatana-appindicator3-dev libasound2-dev \
  libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev \
  libxxf86vm-dev libwayland-dev mingw-w64"

echo "==> Installing build dependencies..."
sudo apt-get update -qq
sudo apt-get install -y -qq ${LINUX_PACKAGES}

echo "==> Building Linux amd64 binaries..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildvcs=false -o "${OUTDIR}/icetray-linux-amd64" .
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o "${OUTDIR}/icetray-headless-linux-amd64" .

echo "==> Building Windows binaries..."
go install github.com/tc-hib/go-winres@latest
export PATH="$(go env GOPATH)/bin:${PATH}"
go-winres make

CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -buildvcs=false -ldflags "-H=windowsgui" -o "${OUTDIR}/icetray-windows-amd64.exe" .
CGO_ENABLED=1 CC=aarch64-w64-mingw32-gcc GOOS=windows GOARCH=arm64 go build -buildvcs=false -ldflags "-H=windowsgui" -o "${OUTDIR}/icetray-windows-arm64.exe" .

rm -f rsrc_*.syso

echo "==> amd64 build complete:"
ls -lh "${OUTDIR}/"icetray-{linux,headless-linux}-amd64 "${OUTDIR}/"icetray-windows-*.exe
