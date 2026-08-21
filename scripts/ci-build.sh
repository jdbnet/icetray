#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
mkdir -p "${OUTDIR}"

LINUX_PACKAGES="gcc libgtk-3-dev libayatana-appindicator3-dev libasound2-dev \
  libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev \
  libxxf86vm-dev libwayland-dev mingw-w64"

echo "==> Building Linux amd64 and Windows binaries..."
docker run --rm -v "$(pwd):/workspace" -w /workspace golang:1.25-bookworm bash -c "
  set -euo pipefail
  apt-get update && apt-get install -y ${LINUX_PACKAGES}

  GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildvcs=false -o ${OUTDIR}/icetray-linux-amd64 .
  GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o ${OUTDIR}/icetray-headless-linux-amd64 .

  go install github.com/tc-hib/go-winres@latest
  export PATH=\"\$PATH:\$(go env GOPATH)/bin\"
  go-winres make

  CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -buildvcs=false -ldflags \"-H=windowsgui\" -o ${OUTDIR}/icetray-windows-amd64.exe .
  CGO_ENABLED=1 CC=aarch64-w64-mingw32-gcc GOOS=windows GOARCH=arm64 go build -buildvcs=false -ldflags \"-H=windowsgui\" -o ${OUTDIR}/icetray-windows-arm64.exe .
"

rm -f rsrc_*.syso

echo "==> Building Linux arm64 binaries..."
docker run --rm --platform linux/arm64 -v "$(pwd):/workspace" -w /workspace golang:1.25-bookworm bash -c "
  set -euo pipefail
  apt-get update && apt-get install -y gcc libgtk-3-dev libayatana-appindicator3-dev libasound2-dev \
    libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev libwayland-dev

  GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -buildvcs=false -o ${OUTDIR}/icetray-linux-arm64 .
  GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o ${OUTDIR}/icetray-headless-linux-arm64 .
"

echo "==> Build complete:"
ls -lh "${OUTDIR}/"
