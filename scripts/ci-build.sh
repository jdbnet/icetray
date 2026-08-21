#!/usr/bin/env bash
set -euo pipefail

# Local fallback: build amd64 natively, arm64 via Docker/QEMU when not on an ARM host.
cd "$(dirname "$0")/.."

OUTDIR="${OUTDIR:-dist}"
export OUTDIR

bash scripts/ci-build-amd64.sh

if [ "$(uname -m)" = "aarch64" ]; then
  bash scripts/ci-build-arm64.sh
else
  echo "==> Building Linux arm64 binaries via Docker (QEMU)..."
  docker run --rm --platform linux/arm64 -v "$(pwd):/workspace" -w /workspace golang:1.25-bookworm bash -c "
    set -euo pipefail
    apt-get update -qq && apt-get install -y -qq gcc libgtk-3-dev libayatana-appindicator3-dev libasound2-dev \
      libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev libwayland-dev

    GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -buildvcs=false -o ${OUTDIR}/icetray-linux-arm64 .
    GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -buildvcs=false -tags headless -o ${OUTDIR}/icetray-headless-linux-arm64 .
  "
fi

echo "==> Build complete:"
ls -lh "${OUTDIR}/"
